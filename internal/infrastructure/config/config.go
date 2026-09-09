package config

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

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
	configFile, ok := findConfigFile("base.yaml")
	if !ok {
		slog.Error("base configuration file not found", "path", "deploy/config/base.yaml")
		return
	}
	if err := loadConfigFile(configFile, false); err != nil {
		configLoadErr = err
		slog.Error("read base configuration failed", "error", err)
		return
	}

	// 根据环境变量加载对应环境的配置文件
	env := configuredEnvironment()
	envConfigName := fmt.Sprintf("%s.yaml", env)
	envConfigPath, _ := findConfigFile(envConfigName)
	if envConfigPath == "" {
		configLoadErr = fmt.Errorf("environment config file not found: %s", envConfigName)
		slog.Error("environment configuration file not found", "environment", env)
		return
	}

	if err := loadConfigFile(envConfigPath, true); err != nil {
		configLoadErr = err
		slog.Error("merge environment configuration failed", "error", err)
	}

	Verification = viper.GetString("verification")
	AllowedOrigins = viper.GetStringSlice("cors.allowed_origins")
	if len(AllowedOrigins) == 0 && Verification != "" {
		AllowedOrigins = []string{"http://" + Verification}
	}
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
	if len(missing) == 0 {
		return nil
	}

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

// findDirectory locates repository assets independent of the process working
// directory. BENETNASCH_*_DIR can point at an absolute container mount while
// local tests use the repository defaults.
func findDirectory(envName, defaultPath string) (string, bool) {
	configured := strings.TrimSpace(os.Getenv(envName))
	if configured != "" {
		if !filepath.IsAbs(configured) {
			if cwd, err := os.Getwd(); err == nil {
				configured = filepath.Join(cwd, configured)
			}
		}
		if info, err := os.Stat(configured); err == nil && info.IsDir() {
			return configured, true
		}
	}

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
			candidate := filepath.Join(root, defaultPath)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
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

// findConfigFile locates a file below the configured configuration directory.
func findConfigFile(name string) (string, bool) {
	root, ok := findDirectory("BENETNASCH_CONFIG_DIR", filepath.Join("deploy", "config"))
	if !ok {
		return "", false
	}
	candidate := filepath.Join(root, name)
	if _, err := os.Stat(candidate); err != nil {
		return "", false
	}
	return candidate, true
}

// ResourcePath resolves a path below the repository/container resources
// directory without depending on the process working directory.
func ResourcePath(name string) (string, error) {
	root, ok := findDirectory("BENETNASCH_RESOURCE_DIR", "resources")
	if !ok {
		return "", fmt.Errorf("resources directory not found")
	}
	return filepath.Join(root, name), nil
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
