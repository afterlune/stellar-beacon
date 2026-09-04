package config

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"bytes"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

var (
	// Verification referer host
	Verification   string
	AllowedOrigins []string
	configLoadErr  error
	missingEnvVars = make(map[string]struct{})
)

var requiredEnvVars = []string{
	"MEILI_MASTER_KEY",
	"POSTGRES_PASSWORD",
	"REDIS_PASSWORD",
	"SMTP_PASSWORD",
}

func init() {
	// 读取基础配置文件
	configFile, ok := findConfigFile("config.yaml")
	if !ok {
		slog.Error("base configuration file not found", "path", "resource/config.yaml")
		return
	}
	if err := loadConfigFile(configFile, false); err != nil {
		configLoadErr = err
		slog.Error("read base configuration failed", "error_code", apperrors.SafeCode(err))
		return
	}

	// 根据环境变量加载对应环境的配置文件
	env := configuredEnvironment()
	envConfigPath, _ := findConfigFile(fmt.Sprintf("config-%s.yaml", env))
	if envConfigPath == "" {
		configLoadErr = fmt.Errorf("environment config file not found: config-%s.yaml", env)
		slog.Error("environment configuration file not found", "environment", env)
		return
	}

	if err := loadConfigFile(envConfigPath, true); err != nil {
		configLoadErr = err
		slog.Error("merge environment configuration failed", "error_code", apperrors.SafeCode(err))
	}
	if err := applyIntegrationDatabaseOverrides(); err != nil {
		configLoadErr = err
		slog.Error("apply integration database overrides failed", "error_code", apperrors.SafeCode(err))
	}

	Verification = viper.GetString("verification")
	AllowedOrigins = viper.GetStringSlice("cors.allowed_origins")
	if len(AllowedOrigins) == 0 && Verification != "" {
		AllowedOrigins = []string{"http://" + Verification}
	}
}

// applyIntegrationDatabaseOverrides lets a local read-only integration CLI
// use the host-published PostgreSQL port while keeping the container config
// unchanged. It is deliberately restricted to integration + loopback so a
// copied environment variable cannot redirect a production process to an
// arbitrary remote database.
func applyIntegrationDatabaseOverrides() error {
	host := strings.TrimSpace(os.Getenv("BENETNASCH_DATABASE_HOST"))
	portText := strings.TrimSpace(os.Getenv("BENETNASCH_DATABASE_PORT"))
	if host == "" && portText == "" {
		return nil
	}
	if !strings.EqualFold(configuredEnvironment(), "integration") {
		return fmt.Errorf("BENETNASCH_DATABASE_HOST/PORT require BENETNASCH_ENV=integration")
	}
	if host == "" || portText == "" {
		return fmt.Errorf("BENETNASCH_DATABASE_HOST and BENETNASCH_DATABASE_PORT must be provided together")
	}
	if host != "localhost" {
		parsedHost := net.ParseIP(host)
		if parsedHost == nil || !parsedHost.IsLoopback() {
			return fmt.Errorf("BENETNASCH_DATABASE_HOST must be localhost or a loopback IP")
		}
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return fmt.Errorf("BENETNASCH_DATABASE_PORT must be a valid TCP port")
	}
	viper.Set("database.host", host)
	viper.Set("database.port", port)
	return nil
}

func loadConfigFile(path string, merge bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file %s: %w", path, err)
	}
	expanded := os.Expand(string(data), func(key string) string {
		value, ok := os.LookupEnv(key)
		if !ok {
			missingEnvVars[key] = struct{}{}
			return "${" + key + "}"
		}
		return escapeYAMLDoubleQuoted(value)
	})

	viper.SetConfigType("yaml")
	reader := bytes.NewReader([]byte(expanded))
	if merge {
		return viper.MergeConfig(reader)
	}
	return viper.ReadConfig(reader)
}

func escapeYAMLDoubleQuoted(value string) string {
	return strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	).Replace(value)
}

// Validate checks all runtime secrets before the server starts. The values are
// intentionally never included in the returned error.
func Validate() error {
	if configLoadErr != nil {
		return configLoadErr
	}
	if _, err := TrustedProxies(); err != nil {
		return fmt.Errorf("invalid trusted proxy configuration: %w", err)
	}

	required := requiredEnvironmentVars()
	missing := make(map[string]struct{}, len(missingEnvVars)+len(required))
	for name := range missingEnvVars {
		missing[name] = struct{}{}
	}
	for _, name := range required {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			missing[name] = struct{}{}
		}
	}
	if len(missing) > 0 {
		return missingEnvironmentError(missing)
	}
	if err := validateLocalEmbeddingConfiguration(); err != nil {
		return err
	}
	space := SpaceCompanion()
	if space.Enabled {
		if space.AgentID != port.MoonfeiPrincipalID {
			return fmt.Errorf("space companion agent id must be %q", port.MoonfeiPrincipalID)
		}
		if len([]byte(strings.TrimSpace(space.ReadToken))) < 32 {
			return fmt.Errorf("space companion read token must contain at least 32 bytes")
		}
		if space.PublishEnabled {
			if len([]byte(strings.TrimSpace(space.PublishToken))) < 32 {
				return fmt.Errorf("space companion publish token must contain at least 32 bytes")
			}
			if subtle.ConstantTimeCompare([]byte(space.ReadToken), []byte(space.PublishToken)) == 1 {
				return fmt.Errorf("space companion read and publish tokens must be different")
			}
		}
	}
	return nil
}

// ValidateDatabase checks only the configuration needed by the explicit
// migration commands. A migration status snapshot must remain usable when
// unrelated runtime services (Redis, Meilisearch, SMTP or object storage) are
// not configured; the server still uses Validate for its full startup gate.
func ValidateDatabase() error {
	if configLoadErr != nil {
		return configLoadErr
	}

	missing := make(map[string]struct{}, 1)
	if strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD")) == "" {
		missing["POSTGRES_PASSWORD"] = struct{}{}
	}
	if len(missing) > 0 {
		return missingEnvironmentError(missing)
	}

	settings := map[string]string{
		"database.driverName": viper.GetString("database.driverName"),
		"database.host":       viper.GetString("database.host"),
		"database.username":   viper.GetString("database.username"),
		"database.baseName":   viper.GetString("database.baseName"),
	}
	missingSettings := make([]string, 0, len(settings))
	for name, value := range settings {
		if strings.TrimSpace(value) == "" || strings.Contains(value, "${") {
			missingSettings = append(missingSettings, name)
		}
	}
	if port := viper.GetInt("database.port"); port <= 0 || port > 65535 {
		missingSettings = append(missingSettings, "database.port")
	}
	if len(missingSettings) > 0 {
		sort.Strings(missingSettings)
		return fmt.Errorf("invalid database configuration: %s", strings.Join(missingSettings, ", "))
	}
	return nil
}

// ValidateMeiliSearch checks only the configuration needed by explicit
// index-management commands. A provision/backfill release step must remain
// usable without requiring unrelated SMTP, Redis, object-storage, or JWT
// secrets; the full server still uses Validate for its startup gate.
func ValidateMeiliSearch() error {
	if configLoadErr != nil {
		return configLoadErr
	}

	if strings.TrimSpace(os.Getenv("MEILI_MASTER_KEY")) == "" {
		return missingEnvironmentError(map[string]struct{}{"MEILI_MASTER_KEY": {}})
	}

	settings := map[string]string{
		"meiliSearch.host":   viper.GetString("meiliSearch.host"),
		"meiliSearch.apiKey": viper.GetString("meiliSearch.apiKey"),
	}
	missingSettings := make([]string, 0, len(settings))
	for name, value := range settings {
		if strings.TrimSpace(value) == "" || strings.Contains(value, "${") {
			missingSettings = append(missingSettings, name)
		}
	}
	if port := viper.GetInt("meiliSearch.port"); port <= 0 || port > 65535 {
		missingSettings = append(missingSettings, "meiliSearch.port")
	}
	if len(missingSettings) > 0 {
		sort.Strings(missingSettings)
		return fmt.Errorf("invalid Meilisearch configuration: %s", strings.Join(missingSettings, ", "))
	}
	return nil
}

func missingEnvironmentError(missing map[string]struct{}) error {
	names := make([]string, 0, len(missing))
	for name := range missing {
		names = append(names, name)
	}
	sort.Strings(names)
	return fmt.Errorf("missing required configuration environment variables: %s", strings.Join(names, ", "))
}

func configuredEnvironment() string {
	if env := strings.TrimSpace(os.Getenv("BENETNASCH_ENV")); env != "" {
		return env
	}
	return viper.GetString("env")
}

func requiredEnvironmentVars() []string {
	required := append([]string(nil), requiredEnvVars...)
	provider := strings.ToLower(strings.TrimSpace(viper.GetString("oss.provider")))
	if provider != "minio" {
		required = append(required, "ALIYUN_OSS_ACCESS_KEY_ID", "ALIYUN_OSS_ACCESS_KEY_SECRET")
	}
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("JWT_ALLOW_EPHEMERAL")), "true") {
		required = append(required, "JWT_PRIVATE_KEY", "JWT_PUBLIC_KEY")
	}
	return required
}

// findConfigFile locates resource files independent of the process working
// directory. This matters for tests, CLI invocations from subdirectories,
// and systemd/container launchers.
func findConfigFile(name string) (string, bool) {
	roots := make([]string, 0, 2)
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, cwd)
	}
	if _, source, _, ok := runtime.Caller(0); ok {
		roots = append(roots, filepath.Dir(source))
	}
	seen := make(map[string]struct{})
	for _, root := range roots {
		for {
			if _, exists := seen[root]; exists {
				break
			}
			seen[root] = struct{}{}
			candidate := filepath.Join(root, "resource", name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, true
			}
			parent := filepath.Dir(root)
			if parent == root {
				break
			}
			root = parent
		}
	}
	return "", false
}

// ResourcePath resolves a path below the repository/container resource
// directory without depending on the process working directory.
func ResourcePath(name string) (string, error) {
	configFile, ok := findConfigFile("config.yaml")
	if !ok {
		return "", fmt.Errorf("resource directory not found")
	}
	return filepath.Join(filepath.Dir(configFile), name), nil
}

func IsAllowedOrigin(origin string) bool {
	for _, allowed := range AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

type Database struct {
	URL        string
	DriverName string
	// ShowSQL is an explicit local-diagnostics switch. It defaults to false so
	// query parameters and user content are never emitted by the ORM logger.
	ShowSQL bool
}

func (base *Database) DataBase() *Database {
	driverName := viper.GetString("database.driverName")
	username := viper.GetString("database.username")
	password := viper.GetString("database.password")
	host := viper.GetString("database.host")
	port := viper.GetInt("database.port")
	baseName := viper.GetString("database.baseName")
	base.URL = (&url.URL{
		Scheme:   driverName,
		User:     url.UserPassword(username, password),
		Host:     net.JoinHostPort(host, strconv.Itoa(port)),
		Path:     "/" + baseName,
		RawQuery: "sslmode=disable",
	}).String()
	base.DriverName = driverName
	base.ShowSQL = viper.GetBool("database.show_sql")
	return base
}

type MeiliSearch struct {
	ApiKey string
	URL    string
}

func (meili *MeiliSearch) MeiliSearch() *MeiliSearch {
	meili.ApiKey = viper.GetString("meiliSearch.apiKey")
	meili.URL = fmt.Sprintf("http://%s", net.JoinHostPort(
		viper.GetString("meiliSearch.host"),
		strconv.Itoa(viper.GetInt("meiliSearch.port")),
	))
	return meili
}

type Redis struct {
	Addr     string
	Password string
	DB       int
}

func (redis *Redis) Redis() *Redis {
	redis.Addr = net.JoinHostPort(
		viper.GetString("redis.host"),
		strconv.Itoa(viper.GetInt("redis.port")),
	)
	redis.Password = viper.GetString("redis.password")
	redis.DB = viper.GetInt("redis.db")
	return redis
}

type Email struct {
	EmailAccount string
	Password     string
	SmtpPort     int
	SmtpName     string
}

func (e *Email) Email() *Email {
	e.EmailAccount = viper.GetString("emailSmtp.email")
	e.Password = viper.GetString("emailSmtp.password")
	e.SmtpPort = viper.GetInt("emailSmtp.port")
	e.SmtpName = viper.GetString("emailSmtp.smtp")
	return e
}

type Oss struct {
	Provider        string
	BucketName      string
	EndPoint        string
	Region          string
	PublicURL       string
	PublicRead      bool
	AccessKeyID     string
	AccessKeySecret string
}

func (o *Oss) Oss() *Oss {
	o.Provider = viper.GetString("oss.provider")
	o.BucketName = viper.GetString("oss.bucketName")
	o.EndPoint = viper.GetString("oss.endPoint")
	o.Region = viper.GetString("oss.region")
	o.PublicURL = viper.GetString("oss.publicUrl")
	o.PublicRead = viper.GetBool("oss.publicRead")
	o.AccessKeyID = viper.GetString("oss.accessKeyID")
	o.AccessKeySecret = viper.GetString("oss.accessKeySecret")
	return o
}

// AI contains only non-secret rollout and resource-limit settings. Provider
// credentials are intentionally not part of the configuration file and must
// be supplied by environment-specific infrastructure. Video frame origins
// are reviewed HTTPS origins, never arbitrary values from a request.
// Every capability is disabled by default so enabling the long-term AI work
// cannot change the existing request paths accidentally.
type AI struct {
	Enabled                     bool
	ArticleIndexingEnabled      bool
	ContentUnderstandingEnabled bool
	PublicChatEnabled           bool
	WritingEnabled              bool
	BehaviorEnabled             bool
	DreamsEnabled               bool
	DreamImagesEnabled          bool
	CapsulesEnabled             bool
	VitalsEnabled               bool
	GalaxyEnabled               bool
	RadioEnabled                bool
	VideosEnabled               bool
	TTSEnabled                  bool
	ProviderProbeEnabled        bool
	LocalEmbeddingEnabled       bool
	VisionEnabled               bool
	ObservabilityEnabled        bool
	SpaceCompanionEnabled       bool
	VideoAllowedOrigins         []string
	EmergencyStop               bool
	DefaultProtocol             string
	Chat                        AIModelRoute
	Embedding                   AIModelRoute
	EmbeddingConfig             AIEmbeddingConfig
	Vision                      AIModelRoute
	Writing                     AIModelRoute
	Moderation                  AIModelRoute
	Behavior                    AIModelRoute
	Dream                       AIModelRoute
	SemanticRatio               float32
	RequestTimeout              time.Duration
	MaxInputTokens              int
	MaxOutputTokens             int
	MaxConcurrent               int
	CircuitFailureThreshold     int
	CircuitResetTimeout         time.Duration
	DailyTokenBudget            int64
	AgentProfile                AIAgentProfileSettings
	AgentReviewPolicy           AIAgentReviewPolicySettings
	AgentMemory                 AIAgentMemorySettings
	AgentRhythm                 AIAgentRhythmSettings
	AgentLimits                 AIAgentLimitsSettings
	AgentBehavior               AIAgentBehaviorSettings
}

// AIAgentProfileSettings is non-secret, versioned public-persona metadata.
// Keeping it separate from provider routes makes changing the opening or
// prompt version an explicit rollout decision and never requires an API key.
type AIAgentProfileSettings struct {
	ID                 string
	Name               string
	PromptVersion      string
	SystemPrompt       string
	Opening            string
	AwakePrompt        string
	DuskPrompt         string
	NightPrompt        string
	PersistenceEnabled bool
}

// AIAgentReviewPolicySettings controls how the review policy is sourced. The
// policy itself is still validated by the domain port; this setting only
// selects configuration-backed or explicitly migrated persistence.
type AIAgentReviewPolicySettings struct {
	ID                 string
	PersistenceEnabled bool
}

// AIAgentMemorySettings controls the authenticated memory review surface.
// It is separate from public chat and behavior so enabling one cannot
// accidentally persist user facts or expose historical assertions.
type AIAgentMemorySettings struct {
	PersistenceEnabled bool
}

// AIAgentRhythmSettings controls the deterministic time-of-day policy. The
// values are plain clock strings so they can be reviewed and changed without
// coupling configuration to a model provider.
type AIAgentRhythmSettings struct {
	Timezone   string
	AwakeStart string
	DuskStart  string
	NightStart string
}

// AIAgentLimitsSettings contains public-chat policy values. They are not
// provider limits: the former protects the visitor-facing use case, while
// provider concurrency remains owned by the model adapter.
type AIAgentLimitsSettings struct {
	GuestDailyTurns   int
	AdminDailyTurns   int
	MaxConcurrent     int
	MaxInputRunes     int
	MaxAnswerRunes    int
	MaxToolCalls      int
	SensitivePatterns []string
}

// AIAgentBehaviorSettings controls the review-only autonomous behavior
// worker. ActorID is deliberately empty by default: enabling the feature
// without explicitly binding a virtual account must fail closed at startup.
type AIAgentBehaviorSettings struct {
	ActorID                string
	Nickname               string
	ProfileID              string
	AllowedActions         []string
	ScanBatchSize          int
	ScanInterval           time.Duration
	MaxOutputTokens        int
	MaxCandidateRunes      int
	SimilarityThreshold    float64
	ReviewTTL              time.Duration
	SensitivePatterns      []string
	DailyLimit             int
	PerArticleLimit        int
	PerActionLimit         int
	ForgottenAfter         time.Duration
	ForgottenMaxCandidates int
}

// AIEmbeddingConfig describes a user-provided embedding model. The model
// version and vector dimension are part of the index identity contract: a
// change must create a new versioned article_chunks index instead of mixing
// incompatible vectors into an existing one.
type AIEmbeddingConfig struct {
	IndexVersion string `mapstructure:"index_version"`
	ModelVersion string `mapstructure:"model_version"`
	Dimension    int    `mapstructure:"dimension"`
	BatchSize    int    `mapstructure:"batch_size"`
}

// AIModelRoute contains non-secret routing metadata. API keys are loaded by
// ProviderSettings from the process environment and never from YAML.
type AIModelRoute struct {
	Provider            string         `mapstructure:"provider"`
	Protocol            string         `mapstructure:"protocol"`
	Model               string         `mapstructure:"model"`
	DataPolicy          string         `mapstructure:"data_policy"`
	Fallbacks           []AIModelRoute `mapstructure:"fallbacks"`
	FallbacksParseError error          `mapstructure:"-" json:"-"`
}

type AIProviderSettings struct {
	APIKey     string
	BaseURL    string
	Model      string
	MaxRetries int
}

type AIProviderSettingsSet struct {
	OpenAI     AIProviderSettings
	Anthropic  AIProviderSettings
	SGLang     AIProviderSettings
	AliBailian AIProviderSettings
}

type AIFeature string

const (
	AIFeatureArticleIndexing      AIFeature = "article_indexing"
	AIFeatureContentUnderstanding AIFeature = "content_understanding"
	AIFeaturePublicChat           AIFeature = "public_chat"
	AIFeatureWriting              AIFeature = "writing"
	AIFeatureBehavior             AIFeature = "behavior"
	AIFeatureDreams               AIFeature = "dreams"
	AIFeatureDreamImages          AIFeature = "dream_images"
	AIFeatureCapsules             AIFeature = "capsules"
	AIFeatureVitals               AIFeature = "vitals"
	AIFeatureGalaxy               AIFeature = "galaxy"
	AIFeatureRadio                AIFeature = "radio"
	AIFeatureVideos               AIFeature = "videos"
	AIFeatureTTS                  AIFeature = "tts"
	AIFeatureProviderProbe        AIFeature = "provider_probe"
	AIFeatureVision               AIFeature = "vision"
	AIFeatureObservability        AIFeature = "observability"
	AIFeatureSpaceCompanion       AIFeature = "space_companion"
)

func (a *AI) AI() *AI {
	a.Enabled = readAIFlag("ai.enabled")
	a.ArticleIndexingEnabled = readAIFlag("ai.article_indexing")
	a.ContentUnderstandingEnabled = readAIFlag("ai.content_understanding")
	a.PublicChatEnabled = readAIFlag("ai.public_chat")
	a.WritingEnabled = readAIFlag("ai.writing")
	a.BehaviorEnabled = readAIFlag("ai.behavior")
	a.DreamsEnabled = readAIFlag("ai.dreams")
	a.DreamImagesEnabled = readAIFlag("ai.dream_images")
	a.CapsulesEnabled = readAIFlag("ai.capsules")
	a.VitalsEnabled = readAIFlag("ai.vitals")
	a.GalaxyEnabled = readAIFlag("ai.galaxy")
	a.RadioEnabled = readAIFlag("ai.radio")
	a.VideosEnabled = readAIFlag("ai.videos")
	a.TTSEnabled = readAIFlag("ai.tts")
	a.ProviderProbeEnabled = readAIFlag("ai.provider_probe")
	a.LocalEmbeddingEnabled = readAIFlag("ai.local_embedding")
	a.VisionEnabled = readAIFlag("ai.vision")
	a.ObservabilityEnabled = readAIFlag("ai.observability")
	a.SpaceCompanionEnabled = readAIFlag("ai.space_companion")
	a.VideoAllowedOrigins = viper.GetStringSlice("ai.video.allowed_origins")
	a.EmergencyStop = viper.GetBool("ai.emergency_stop")
	a.DefaultProtocol = viper.GetString("ai.default_protocol")
	a.Chat = readAIModelRoute("ai.routes.chat")
	a.Embedding = readAIModelRoute("ai.routes.embedding")
	a.EmbeddingConfig = readAIEmbeddingConfig("ai.routes.embedding")
	a.Vision = readAIModelRoute("ai.routes.vision")
	a.Writing = readAIModelRoute("ai.routes.writing")
	a.Moderation = readAIModelRoute("ai.routes.moderation")
	a.Behavior = readAIModelRoute("ai.routes.behavior")
	a.Dream = readAIModelRoute("ai.routes.dream")
	a.SemanticRatio = float32(viper.GetFloat64("ai.search.semantic_ratio"))
	a.RequestTimeout = viper.GetDuration("ai.limits.request_timeout")
	a.MaxInputTokens = viper.GetInt("ai.limits.max_input_tokens")
	a.MaxOutputTokens = viper.GetInt("ai.limits.max_output_tokens")
	a.MaxConcurrent = viper.GetInt("ai.limits.max_concurrent")
	a.CircuitFailureThreshold = viper.GetInt("ai.limits.circuit_failure_threshold")
	a.CircuitResetTimeout = viper.GetDuration("ai.limits.circuit_reset_timeout")
	a.DailyTokenBudget = viper.GetInt64("ai.limits.daily_token_budget")
	a.AgentProfile = readAIAgentProfile()
	a.AgentReviewPolicy = readAIAgentReviewPolicy()
	a.AgentMemory = readAIAgentMemory()
	a.AgentRhythm = readAIAgentRhythm()
	a.AgentLimits = readAIAgentLimits()
	a.AgentBehavior = readAIAgentBehavior()

	if a.DefaultProtocol == "" {
		a.DefaultProtocol = "openai_responses"
	}
	if a.SemanticRatio <= 0 || a.SemanticRatio > 1 {
		a.SemanticRatio = 0.65
	}
	if a.RequestTimeout <= 0 {
		a.RequestTimeout = 30 * time.Second
	}
	if a.MaxInputTokens <= 0 {
		a.MaxInputTokens = 4000
	}
	if a.MaxOutputTokens <= 0 {
		a.MaxOutputTokens = 1000
	}
	if a.MaxConcurrent <= 0 {
		a.MaxConcurrent = 2
	}
	if a.EmbeddingConfig.IndexVersion == "" {
		a.EmbeddingConfig.IndexVersion = "v1"
	}
	if a.EmbeddingConfig.BatchSize <= 0 {
		a.EmbeddingConfig.BatchSize = 32
	}
	if a.CircuitFailureThreshold <= 0 {
		a.CircuitFailureThreshold = 3
	}
	if a.CircuitResetTimeout <= 0 {
		a.CircuitResetTimeout = 10 * time.Second
	}
	if a.DailyTokenBudget < 0 {
		a.DailyTokenBudget = 0
	}
	if strings.TrimSpace(a.AgentProfile.ID) == "" {
		a.AgentProfile.ID = "benetnasch-public"
	}
	if strings.TrimSpace(a.AgentReviewPolicy.ID) == "" {
		a.AgentReviewPolicy.ID = "default"
	}
	if strings.TrimSpace(a.AgentProfile.Name) == "" {
		a.AgentProfile.Name = "Benetnasch"
	}
	if strings.TrimSpace(a.AgentProfile.PromptVersion) == "" {
		a.AgentProfile.PromptVersion = "v1"
	}
	if strings.TrimSpace(a.AgentProfile.AwakePrompt) == "" {
		a.AgentProfile.AwakePrompt = "清醒变体：保持专注、清晰和温和，优先给出可验证的公开文章线索。"
	}
	if strings.TrimSpace(a.AgentProfile.DuskPrompt) == "" {
		a.AgentProfile.DuskPrompt = "黄昏变体：语气放缓一些，适合回顾文章之间的联系，但仍保持事实边界。"
	}
	if strings.TrimSpace(a.AgentProfile.NightPrompt) == "" {
		a.AgentProfile.NightPrompt = "深夜变体：保持安静、简洁和克制，不鼓励熬夜，也不虚构无法查证的内容。"
	}
	if strings.TrimSpace(a.AgentRhythm.Timezone) == "" {
		a.AgentRhythm.Timezone = "Asia/Shanghai"
	}
	if strings.TrimSpace(a.AgentRhythm.AwakeStart) == "" {
		a.AgentRhythm.AwakeStart = "06:00"
	}
	if strings.TrimSpace(a.AgentRhythm.DuskStart) == "" {
		a.AgentRhythm.DuskStart = "18:00"
	}
	if strings.TrimSpace(a.AgentRhythm.NightStart) == "" {
		a.AgentRhythm.NightStart = "22:00"
	}
	if a.AgentLimits.GuestDailyTurns <= 0 {
		a.AgentLimits.GuestDailyTurns = 20
	}
	if a.AgentLimits.AdminDailyTurns <= 0 {
		a.AgentLimits.AdminDailyTurns = 200
	}
	if a.AgentLimits.MaxConcurrent <= 0 {
		a.AgentLimits.MaxConcurrent = 2
	}
	if a.AgentLimits.MaxInputRunes <= 0 {
		a.AgentLimits.MaxInputRunes = 4000
	}
	if a.AgentLimits.MaxAnswerRunes <= 0 {
		a.AgentLimits.MaxAnswerRunes = 16000
	}
	if a.AgentLimits.MaxToolCalls <= 0 {
		a.AgentLimits.MaxToolCalls = 4
	}
	if strings.TrimSpace(a.AgentBehavior.Nickname) == "" {
		a.AgentBehavior.Nickname = "月社妃"
	}
	if strings.TrimSpace(a.AgentBehavior.ProfileID) == "" {
		a.AgentBehavior.ProfileID = a.AgentProfile.ID
	}
	if a.AgentBehavior.ScanBatchSize <= 0 {
		a.AgentBehavior.ScanBatchSize = 20
	}
	if a.AgentBehavior.ScanBatchSize > 100 {
		a.AgentBehavior.ScanBatchSize = 100
	}
	if a.AgentBehavior.ScanInterval <= 0 {
		a.AgentBehavior.ScanInterval = 15 * time.Minute
	}
	if a.AgentBehavior.MaxOutputTokens <= 0 {
		a.AgentBehavior.MaxOutputTokens = 600
	}
	if a.AgentBehavior.MaxCandidateRunes <= 0 {
		a.AgentBehavior.MaxCandidateRunes = 500
	}
	if a.AgentBehavior.SimilarityThreshold <= 0 || a.AgentBehavior.SimilarityThreshold > 1 {
		a.AgentBehavior.SimilarityThreshold = 0.82
	}
	if a.AgentBehavior.ReviewTTL <= 0 {
		a.AgentBehavior.ReviewTTL = 7 * 24 * time.Hour
	}
	if a.AgentBehavior.DailyLimit <= 0 {
		a.AgentBehavior.DailyLimit = 3
	}
	if a.AgentBehavior.PerArticleLimit <= 0 {
		a.AgentBehavior.PerArticleLimit = 1
	}
	if a.AgentBehavior.PerActionLimit <= 0 {
		a.AgentBehavior.PerActionLimit = 3
	}
	if a.AgentBehavior.ForgottenAfter <= 0 {
		a.AgentBehavior.ForgottenAfter = 90 * 24 * time.Hour
	}
	if a.AgentBehavior.ForgottenMaxCandidates <= 0 {
		a.AgentBehavior.ForgottenMaxCandidates = 20
	}
	if a.AgentBehavior.ForgottenMaxCandidates > 500 {
		a.AgentBehavior.ForgottenMaxCandidates = 500
	}
	if a.LocalEmbeddingEnabled {
		applyLocalEmbeddingConfiguration(&a.Embedding, &a.EmbeddingConfig)
	}
	return a
}

// applyLocalEmbeddingConfiguration is an explicit, integration-oriented
// route switch. It deliberately clears the file-configured model contract so
// a local model cannot accidentally reuse the default AliBailian index
// metadata. Indexing with this switch therefore requires explicit local
// model/index versions in the environment.
func applyLocalEmbeddingConfiguration(route *AIModelRoute, embedding *AIEmbeddingConfig) {
	if route == nil || embedding == nil {
		return
	}
	route.Provider = "sglang"
	route.Protocol = "openai_chat_completions"
	route.Model = firstEnvironmentValue("SGLANG_MODEL", "SGLANG_MODEL_ID")
	route.Fallbacks = nil
	route.FallbacksParseError = nil

	embedding.IndexVersion = firstEnvironmentValue("BENETNASCH_AI_LOCAL_EMBEDDING_INDEX_VERSION")
	embedding.ModelVersion = firstEnvironmentValue("BENETNASCH_AI_LOCAL_EMBEDDING_MODEL_VERSION")
	embedding.BatchSize = 1
	if raw := firstEnvironmentValue("BENETNASCH_AI_LOCAL_EMBEDDING_BATCH_SIZE"); raw != "" {
		batchSize, err := strconv.Atoi(raw)
		if err != nil || batchSize < 1 || batchSize > 1024 {
			embedding.BatchSize = 0
		} else {
			embedding.BatchSize = batchSize
		}
	}
}

func validateLocalEmbeddingConfiguration() error {
	if !readAIFlag("ai.local_embedding") {
		return nil
	}
	baseURL := firstEnvironmentValue("SGLANG_BASE_URL")
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("invalid local embedding configuration: SGLANG_BASE_URL must be an HTTP(S) URL")
	}
	if firstEnvironmentValue("SGLANG_MODEL", "SGLANG_MODEL_ID") == "" {
		return fmt.Errorf("invalid local embedding configuration: SGLANG_MODEL is required")
	}
	if readAIFlag("ai.article_indexing") {
		if firstEnvironmentValue("BENETNASCH_AI_LOCAL_EMBEDDING_INDEX_VERSION") == "" {
			return fmt.Errorf("invalid local embedding configuration: BENETNASCH_AI_LOCAL_EMBEDDING_INDEX_VERSION is required when article indexing is enabled")
		}
		if firstEnvironmentValue("BENETNASCH_AI_LOCAL_EMBEDDING_MODEL_VERSION") == "" {
			return fmt.Errorf("invalid local embedding configuration: BENETNASCH_AI_LOCAL_EMBEDDING_MODEL_VERSION is required when article indexing is enabled")
		}
	}
	return nil
}

func readAIAgentProfile() AIAgentProfileSettings {
	return AIAgentProfileSettings{
		ID:                 viper.GetString("ai.agent.profile_id"),
		Name:               viper.GetString("ai.agent.name"),
		PromptVersion:      viper.GetString("ai.agent.prompt_version"),
		SystemPrompt:       viper.GetString("ai.agent.system_prompt"),
		Opening:            viper.GetString("ai.agent.opening"),
		AwakePrompt:        viper.GetString("ai.agent.awake_prompt"),
		DuskPrompt:         viper.GetString("ai.agent.dusk_prompt"),
		NightPrompt:        viper.GetString("ai.agent.night_prompt"),
		PersistenceEnabled: viper.GetBool("ai.agent.persistence_enabled"),
	}
}

func readAIAgentReviewPolicy() AIAgentReviewPolicySettings {
	return AIAgentReviewPolicySettings{
		ID:                 viper.GetString("ai.agent.review_policy.id"),
		PersistenceEnabled: viper.GetBool("ai.agent.review_policy.persistence_enabled"),
	}
}

func readAIAgentMemory() AIAgentMemorySettings {
	return AIAgentMemorySettings{
		PersistenceEnabled: viper.GetBool("ai.agent.memory.persistence_enabled"),
	}
}

func readAIAgentRhythm() AIAgentRhythmSettings {
	return AIAgentRhythmSettings{
		Timezone:   viper.GetString("ai.agent.rhythm.timezone"),
		AwakeStart: viper.GetString("ai.agent.rhythm.awake_start"),
		DuskStart:  viper.GetString("ai.agent.rhythm.dusk_start"),
		NightStart: viper.GetString("ai.agent.rhythm.night_start"),
	}
}

func readAIAgentLimits() AIAgentLimitsSettings {
	return AIAgentLimitsSettings{
		GuestDailyTurns:   viper.GetInt("ai.agent.limits.guest_daily_turns"),
		AdminDailyTurns:   viper.GetInt("ai.agent.limits.admin_daily_turns"),
		MaxConcurrent:     viper.GetInt("ai.agent.limits.max_concurrent"),
		MaxInputRunes:     viper.GetInt("ai.agent.limits.max_input_runes"),
		MaxAnswerRunes:    viper.GetInt("ai.agent.limits.max_answer_runes"),
		MaxToolCalls:      viper.GetInt("ai.agent.limits.max_tool_calls"),
		SensitivePatterns: viper.GetStringSlice("ai.agent.limits.sensitive_patterns"),
	}
}

func readAIAgentBehavior() AIAgentBehaviorSettings {
	return AIAgentBehaviorSettings{
		ActorID:                viper.GetString("ai.agent.behavior.actor_id"),
		Nickname:               viper.GetString("ai.agent.behavior.nickname"),
		ProfileID:              viper.GetString("ai.agent.behavior.profile_id"),
		AllowedActions:         viper.GetStringSlice("ai.agent.behavior.allowed_actions"),
		ScanBatchSize:          viper.GetInt("ai.agent.behavior.scan_batch_size"),
		ScanInterval:           viper.GetDuration("ai.agent.behavior.scan_interval"),
		MaxOutputTokens:        viper.GetInt("ai.agent.behavior.max_output_tokens"),
		MaxCandidateRunes:      viper.GetInt("ai.agent.behavior.max_candidate_runes"),
		SimilarityThreshold:    viper.GetFloat64("ai.agent.behavior.similarity_threshold"),
		ReviewTTL:              viper.GetDuration("ai.agent.behavior.review_ttl"),
		SensitivePatterns:      viper.GetStringSlice("ai.agent.behavior.sensitive_patterns"),
		DailyLimit:             viper.GetInt("ai.agent.behavior.daily_limit"),
		PerArticleLimit:        viper.GetInt("ai.agent.behavior.per_article_limit"),
		PerActionLimit:         viper.GetInt("ai.agent.behavior.per_action_limit"),
		ForgottenAfter:         viper.GetDuration("ai.agent.behavior.forgotten_after"),
		ForgottenMaxCandidates: viper.GetInt("ai.agent.behavior.forgotten_max_candidates"),
	}
}

func readAIEmbeddingConfig(key string) AIEmbeddingConfig {
	return AIEmbeddingConfig{
		IndexVersion: viper.GetString(key + ".index_version"),
		ModelVersion: viper.GetString(key + ".model_version"),
		Dimension:    viper.GetInt(key + ".dimension"),
		BatchSize:    viper.GetInt(key + ".batch_size"),
	}
}

func readAIModelRoute(key string) AIModelRoute {
	route := AIModelRoute{
		Provider:   viper.GetString(key + ".provider"),
		Protocol:   viper.GetString(key + ".protocol"),
		Model:      viper.GetString(key + ".model"),
		DataPolicy: viper.GetString(key + ".data_policy"),
	}
	// Fallbacks are explicit configuration, not an implicit provider list. Keep
	// a parse failure on the route so the router can return a typed validation
	// error when that capability is resolved instead of silently dropping the
	// malformed fallback and using only the primary route.
	fallbacksKey := key + ".fallbacks"
	if rawFallbacks := viper.Get(fallbacksKey); rawFallbacks != nil && !isAIModelRouteList(rawFallbacks) {
		route.FallbacksParseError = fmt.Errorf("fallback configuration must be a list of route objects")
		return route
	}
	if err := viper.UnmarshalKey(fallbacksKey, &route.Fallbacks); err != nil {
		route.FallbacksParseError = fmt.Errorf("decode fallback routes: %w", err)
	}
	return route
}

func isAIModelRouteList(value any) bool {
	rv := reflect.ValueOf(value)
	for rv.IsValid() && rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return true
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return false
	}
	for index := 0; index < rv.Len(); index++ {
		item := rv.Index(index)
		for item.IsValid() && item.Kind() == reflect.Interface {
			if item.IsNil() {
				return false
			}
			item = item.Elem()
		}
		if !item.IsValid() {
			return false
		}
		switch item.Kind() {
		case reflect.Map:
			if item.Type().Key().Kind() != reflect.String {
				return false
			}
		case reflect.Struct:
			// Accept typed route structs for programmatic configuration.
		default:
			return false
		}
	}
	return true
}

// ProviderSettings reads provider credentials at the last possible boundary.
// Empty values are allowed while AI is disabled and are validated by the
// adapter only when the corresponding route is actually resolved.
func ProviderSettings() AIProviderSettingsSet {
	return AIProviderSettingsSet{
		OpenAI: AIProviderSettings{
			APIKey:     os.Getenv("OPENAI_API_KEY"),
			BaseURL:    firstEnvironmentValue("OPENAI_BASE_URL"),
			Model:      firstEnvironmentValue("OPENAI_MODEL", "OPENAI_MODEL_ID"),
			MaxRetries: viper.GetInt("ai.providers.openai.max_retries"),
		},
		Anthropic: AIProviderSettings{
			APIKey:     os.Getenv("ANTHROPIC_API_KEY"),
			BaseURL:    firstEnvironmentValue("ANTHROPIC_BASE_URL"),
			Model:      firstEnvironmentValue("ANTHROPIC_MODEL", "ANTHROPIC_MODEL_ID"),
			MaxRetries: viper.GetInt("ai.providers.anthropic.max_retries"),
		},
		SGLang: AIProviderSettings{
			APIKey:     os.Getenv("SGLANG_API_KEY"),
			BaseURL:    firstEnvironmentValue("SGLANG_BASE_URL"),
			Model:      firstEnvironmentValue("SGLANG_MODEL", "SGLANG_MODEL_ID"),
			MaxRetries: viper.GetInt("ai.providers.sglang.max_retries"),
		},
		AliBailian: AIProviderSettings{
			APIKey:     os.Getenv("ALIBAILIAN_API_KEY"),
			BaseURL:    firstEnvironmentValue("ALIBAILIAN_BASE_URL"),
			Model:      firstEnvironmentValue("ALIBAILIAN_MODEL"),
			MaxRetries: viper.GetInt("ai.providers.alibailian.max_retries"),
		},
	}
}

func firstEnvironmentValue(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

// readAIFlag permits an explicit deployment-time override without changing
// the safe configuration-file defaults. Invalid values fail closed instead
// of silently enabling a capability.
func readAIFlag(configKey string) bool {
	key := strings.TrimPrefix(configKey, "ai.")
	envName := "BENETNASCH_AI_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	if value, ok := os.LookupEnv(envName); ok {
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		return err == nil && parsed
	}
	return viper.GetBool(configKey)
}

// FeatureEnabled applies the global switch as well as the capability switch,
// so a partially edited environment file cannot enable an AI path by itself.
func (a AI) FeatureEnabled(feature AIFeature) bool {
	if !a.Enabled {
		return false
	}
	switch feature {
	case AIFeatureArticleIndexing:
		return a.ArticleIndexingEnabled
	case AIFeatureContentUnderstanding:
		return a.ContentUnderstandingEnabled
	case AIFeaturePublicChat:
		return a.PublicChatEnabled
	case AIFeatureWriting:
		return a.WritingEnabled
	case AIFeatureBehavior:
		return a.BehaviorEnabled
	case AIFeatureDreams:
		return a.DreamsEnabled
	case AIFeatureDreamImages:
		return a.DreamImagesEnabled
	case AIFeatureCapsules:
		return a.CapsulesEnabled
	case AIFeatureVitals:
		return a.VitalsEnabled
	case AIFeatureGalaxy:
		return a.GalaxyEnabled
	case AIFeatureRadio:
		return a.RadioEnabled
	case AIFeatureVideos:
		return a.VideosEnabled
	case AIFeatureTTS:
		return a.TTSEnabled
	case AIFeatureProviderProbe:
		return a.ProviderProbeEnabled
	case AIFeatureVision:
		return a.VisionEnabled
	case AIFeatureObservability:
		return a.ObservabilityEnabled
	case AIFeatureSpaceCompanion:
		return a.SpaceCompanionEnabled
	default:
		return false
	}
}

// SpaceCompanionSettings is the deployment-only configuration for the
// internal Companion protocol. Tokens are intentionally read from the
// environment and never from YAML or a public API response.
type SpaceCompanionSettings struct {
	Enabled         bool
	PublishEnabled  bool
	AgentID         string
	ReadToken       string
	PublishToken    string
	MaxRequestBytes int64
}

func SpaceCompanion() SpaceCompanionSettings {
	settings := SpaceCompanionSettings{
		Enabled:         readAIFlag("ai.enabled") && readAIFlag("ai.space_companion"),
		PublishEnabled:  readAIFlag("ai.enabled") && readAIFlag("ai.space_companion_publish"),
		AgentID:         strings.TrimSpace(os.Getenv("BENETNASCH_SPACE_COMPANION_AGENT_ID")),
		ReadToken:       strings.TrimSpace(os.Getenv("BENETNASCH_SPACE_COMPANION_READ_TOKEN")),
		PublishToken:    strings.TrimSpace(os.Getenv("BENETNASCH_SPACE_COMPANION_PUBLISH_TOKEN")),
		MaxRequestBytes: 256 * 1024,
	}
	if settings.AgentID == "" {
		settings.AgentID = port.MoonfeiPrincipalID
	}
	if settings.PublishEnabled && !settings.Enabled {
		settings.PublishEnabled = false
	}
	return settings
}
