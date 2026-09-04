package config

import (
	"benetnasch/app/domain/port"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestEscapeYAMLDoubleQuoted(t *testing.T) {
	input := "value with \"quotes\", \\slashes\\, and\nline"
	want := "value with \\\"quotes\\\", \\\\slashes\\\\, and\\nline"
	if got := escapeYAMLDoubleQuoted(input); got != want {
		t.Fatalf("escapeYAMLDoubleQuoted() = %q, want %q", got, want)
	}
}

func TestReadAIModelRouteParsesExplicitFallbacks(t *testing.T) {
	key := "ai.routes.config_test_fallback"
	viper.Set(key, map[string]any{
		"provider":    "openai",
		"protocol":    "openai_chat_completions",
		"model":       "primary",
		"data_policy": "public",
		"fallbacks": []map[string]any{{
			"provider":    "sglang",
			"protocol":    "openai_chat_completions",
			"model":       "secondary",
			"data_policy": "public",
		}},
	})
	route := readAIModelRoute(key)
	if route.Provider != "openai" || route.DataPolicy != "public" || len(route.Fallbacks) != 1 {
		t.Fatalf("route = %#v, want one explicit fallback", route)
	}
	fallback := route.Fallbacks[0]
	if fallback.Provider != "sglang" || fallback.Protocol != "openai_chat_completions" || fallback.Model != "secondary" || fallback.DataPolicy != "public" {
		t.Fatalf("fallback = %#v", fallback)
	}
}

func TestReadAIModelRouteRetainsMalformedFallbackError(t *testing.T) {
	key := "ai.routes.config_test_malformed_fallback"
	t.Cleanup(func() { viper.Set(key, nil) })
	viper.Set(key, map[string]any{
		"provider":  "openai",
		"protocol":  "openai_chat_completions",
		"model":     "primary",
		"fallbacks": "not-a-list",
	})

	route := readAIModelRoute(key)
	if route.FallbacksParseError == nil {
		t.Fatal("malformed fallback configuration error was discarded")
	}
	if len(route.Fallbacks) != 0 {
		t.Fatalf("malformed fallback configuration produced routes: %#v", route.Fallbacks)
	}
}

func TestAIConfigDefaultsAreSafeAndFeatureGated(t *testing.T) {
	for _, name := range []string{
		"BENETNASCH_AI_ENABLED", "BENETNASCH_AI_ARTICLE_INDEXING", "BENETNASCH_AI_CONTENT_UNDERSTANDING",
		"BENETNASCH_AI_PUBLIC_CHAT", "BENETNASCH_AI_WRITING", "BENETNASCH_AI_BEHAVIOR",
		"BENETNASCH_AI_DREAMS", "BENETNASCH_AI_DREAM_IMAGES", "BENETNASCH_AI_CAPSULES",
		"BENETNASCH_AI_VITALS", "BENETNASCH_AI_GALAXY", "BENETNASCH_AI_RADIO",
		"BENETNASCH_AI_VIDEOS", "BENETNASCH_AI_TTS", "BENETNASCH_AI_PROVIDER_PROBE", "BENETNASCH_AI_LOCAL_EMBEDDING", "BENETNASCH_AI_VISION", "BENETNASCH_AI_OBSERVABILITY",
		"BENETNASCH_AI_SPACE_COMPANION", "BENETNASCH_AI_SPACE_COMPANION_PUBLISH",
	} {
		t.Setenv(name, "")
	}
	ai := new(AI).AI()
	if ai.Enabled || ai.EmergencyStop || ai.ProviderProbeEnabled || ai.LocalEmbeddingEnabled || ai.VisionEnabled || ai.ObservabilityEnabled || ai.ArticleIndexingEnabled || ai.ContentUnderstandingEnabled || ai.CapsulesEnabled || ai.RadioEnabled || ai.VideosEnabled || ai.TTSEnabled || ai.FeatureEnabled(AIFeatureArticleIndexing) || ai.FeatureEnabled(AIFeatureContentUnderstanding) || ai.FeatureEnabled(AIFeaturePublicChat) || ai.FeatureEnabled(AIFeatureCapsules) || ai.FeatureEnabled(AIFeatureRadio) || ai.FeatureEnabled(AIFeatureVideos) || ai.FeatureEnabled(AIFeatureTTS) || ai.FeatureEnabled(AIFeatureProviderProbe) || ai.FeatureEnabled(AIFeatureVision) || ai.FeatureEnabled(AIFeatureObservability) {
		t.Fatalf("AI should be disabled by default: %+v", ai)
	}
	if ai.SemanticRatio != 0.65 || ai.RequestTimeout != 30*time.Second || ai.MaxConcurrent != 2 || ai.CircuitFailureThreshold != 3 || ai.CircuitResetTimeout != 10*time.Second {
		t.Fatalf("unexpected AI defaults: %+v", ai)
	}
	if ai.EmbeddingConfig.IndexVersion != "v2" || ai.EmbeddingConfig.ModelVersion != "qwen3.7-text-embedding" || ai.EmbeddingConfig.BatchSize != 32 || ai.EmbeddingConfig.Dimension != 1024 {
		t.Fatalf("unexpected embedding defaults: %+v", ai.EmbeddingConfig)
	}
	if ai.Chat.Provider != "openai" || ai.Chat.Protocol != string(port.ProviderProtocolOpenAIChatCompletions) || ai.Embedding.Provider != "alibailian" || ai.Embedding.Model != "qwen3.7-text-embedding" || ai.Vision.Provider != "openai" || ai.Vision.Protocol != string(port.ProviderProtocolOpenAIChatCompletions) {
		t.Fatalf("unexpected model routes: chat=%+v embedding=%+v vision=%+v", ai.Chat, ai.Embedding, ai.Vision)
	}
	if ai.AgentLimits.GuestDailyTurns != 20 || ai.AgentLimits.AdminDailyTurns != 200 || ai.AgentLimits.MaxConcurrent != 2 || ai.AgentLimits.MaxInputRunes != 4000 {
		t.Fatalf("unexpected agent limits: %+v", ai.AgentLimits)
	}
	if ai.AgentRhythm.Timezone != "Asia/Shanghai" || ai.AgentRhythm.AwakeStart != "06:00" || ai.AgentRhythm.DuskStart != "18:00" || ai.AgentRhythm.NightStart != "22:00" {
		t.Fatalf("unexpected agent rhythm defaults: %+v", ai.AgentRhythm)
	}
	if ai.AgentProfile.AwakePrompt == "" || ai.AgentProfile.DuskPrompt == "" || ai.AgentProfile.NightPrompt == "" {
		t.Fatalf("agent rhythm prompt variants must have safe defaults: %+v", ai.AgentProfile)
	}
	if ai.AgentProfile.PersistenceEnabled {
		t.Fatal("agent profile persistence must be opt-in")
	}
	if ai.AgentReviewPolicy.ID != "default" || ai.AgentReviewPolicy.PersistenceEnabled {
		t.Fatalf("agent review policy persistence must be opt-in: %+v", ai.AgentReviewPolicy)
	}
	if ai.AgentMemory.PersistenceEnabled {
		t.Fatal("agent memory persistence must be opt-in")
	}
	if ai.AgentBehavior.DailyLimit != 3 || ai.AgentBehavior.PerArticleLimit != 1 || ai.AgentBehavior.PerActionLimit != 3 || ai.AgentBehavior.ForgottenAfter != 90*24*time.Hour || ai.AgentBehavior.ForgottenMaxCandidates != 20 {
		t.Fatalf("unexpected behavior frequency defaults: %+v", ai.AgentBehavior)
	}
}

func TestSpaceCompanionConfigIsExplicitAndUsesSeparateTokens(t *testing.T) {
	t.Setenv("BENETNASCH_AI_ENABLED", "true")
	t.Setenv("BENETNASCH_AI_SPACE_COMPANION", "true")
	t.Setenv("BENETNASCH_AI_SPACE_COMPANION_PUBLISH", "true")
	t.Setenv("BENETNASCH_SPACE_COMPANION_AGENT_ID", "moonfei")
	t.Setenv("BENETNASCH_SPACE_COMPANION_READ_TOKEN", "read-token")
	t.Setenv("BENETNASCH_SPACE_COMPANION_PUBLISH_TOKEN", "publish-token")

	settings := SpaceCompanion()
	if !settings.Enabled || !settings.PublishEnabled || settings.AgentID != port.MoonfeiPrincipalID {
		t.Fatalf("space companion settings = %+v", settings)
	}
	if settings.ReadToken == settings.PublishToken || settings.ReadToken != "read-token" || settings.PublishToken != "publish-token" {
		t.Fatalf("space companion tokens were not kept separate: %+v", settings)
	}
}

func TestAIConfigAcceptsExplicitDeploymentFlagOverrides(t *testing.T) {
	t.Setenv("BENETNASCH_AI_ENABLED", "true")
	t.Setenv("BENETNASCH_AI_PROVIDER_PROBE", "true")
	t.Setenv("BENETNASCH_AI_VISION", "true")
	t.Setenv("BENETNASCH_AI_ARTICLE_INDEXING", "true")
	t.Setenv("BENETNASCH_AI_OBSERVABILITY", "true")
	t.Setenv("BENETNASCH_AI_PUBLIC_CHAT", "not-a-bool")

	ai := new(AI).AI()
	if !ai.Enabled || !ai.ProviderProbeEnabled || !ai.VisionEnabled || !ai.ArticleIndexingEnabled || !ai.ObservabilityEnabled {
		t.Fatalf("explicit AI flag overrides were ignored: %+v", ai)
	}
	if ai.PublicChatEnabled || ai.FeatureEnabled(AIFeaturePublicChat) {
		t.Fatalf("invalid AI flag override should fail closed: %+v", ai)
	}
	if !ai.FeatureEnabled(AIFeatureProviderProbe) || !ai.FeatureEnabled(AIFeatureVision) || !ai.FeatureEnabled(AIFeatureArticleIndexing) {
		t.Fatalf("enabled AI capabilities were not exposed through FeatureEnabled: %+v", ai)
	}
}

func TestAIConfigParsesAgentMemoryPersistenceFlag(t *testing.T) {
	key := "ai.agent.memory.persistence_enabled"
	previous := viper.Get(key)
	wasSet := viper.IsSet(key)
	t.Cleanup(func() {
		if wasSet {
			viper.Set(key, previous)
			return
		}
		viper.Set(key, nil)
	})

	viper.Set(key, true)
	if got := new(AI).AI().AgentMemory.PersistenceEnabled; !got {
		t.Fatal("agent memory persistence flag should be read from configuration")
	}
}

func TestReadAIEmbeddingConfigParsesUserProvidedModelMetadata(t *testing.T) {
	key := "ai.routes.config_test_embedding"
	viper.Set(key+".index_version", "v2")
	viper.Set(key+".model_version", "2026-08-28")
	viper.Set(key+".dimension", 768)
	viper.Set(key+".batch_size", 16)

	got := readAIEmbeddingConfig(key)
	want := AIEmbeddingConfig{IndexVersion: "v2", ModelVersion: "2026-08-28", Dimension: 768, BatchSize: 16}
	if got != want {
		t.Fatalf("readAIEmbeddingConfig() = %+v, want %+v", got, want)
	}
}

func TestProviderSettingsUsesExplicitOpenAIAndAliBailianEnvironmentNames(t *testing.T) {
	values := map[string]string{
		"OPENAI_API_KEY":      "deepseek-test-key",
		"OPENAI_BASE_URL":     "https://api.deepseek.com",
		"OPENAI_MODEL":        "deepseek-v4-flash-vision-exp",
		"OPENAI_MODEL_ID":     "",
		"ALIBAILIAN_API_KEY":  "alibailian-test-key",
		"ALIBAILIAN_BASE_URL": "https://dashscope.aliyuncs.com/compatible-mode/v1",
		"ALIBAILIAN_MODEL":    "qwen3.7-text-embedding",
	}
	previous := make(map[string]string)
	existed := make(map[string]bool)
	for name, value := range values {
		previous[name], existed[name] = os.LookupEnv(name)
		if err := os.Setenv(name, value); err != nil {
			t.Fatal(err)
		}
		name := name
		t.Cleanup(func() {
			if existed[name] {
				_ = os.Setenv(name, previous[name])
				return
			}
			_ = os.Unsetenv(name)
		})
	}

	previousRetries := viper.Get("ai.providers.alibailian.max_retries")
	wasRetriesSet := viper.IsSet("ai.providers.alibailian.max_retries")
	t.Cleanup(func() {
		if wasRetriesSet {
			viper.Set("ai.providers.alibailian.max_retries", previousRetries)
			return
		}
		viper.Set("ai.providers.alibailian.max_retries", nil)
	})
	viper.Set("ai.providers.alibailian.max_retries", 2)

	got := ProviderSettings()
	if got.OpenAI.APIKey != values["OPENAI_API_KEY"] || got.OpenAI.BaseURL != values["OPENAI_BASE_URL"] || got.OpenAI.Model != values["OPENAI_MODEL"] {
		t.Fatalf("OpenAI provider settings = %+v", got.OpenAI)
	}
	if got.AliBailian.APIKey != values["ALIBAILIAN_API_KEY"] || got.AliBailian.BaseURL != values["ALIBAILIAN_BASE_URL"] || got.AliBailian.Model != values["ALIBAILIAN_MODEL"] || got.AliBailian.MaxRetries != 2 {
		t.Fatalf("AliBailian provider settings = %+v", got.AliBailian)
	}
}

func TestAIUsesSGLangOnlyWithExplicitLocalEmbeddingFlag(t *testing.T) {
	t.Setenv("BENETNASCH_AI_LOCAL_EMBEDDING", "false")
	t.Setenv("SGLANG_MODEL", "Qwen/Qwen3-Embedding-0.6B")
	disabled := new(AI).AI()
	if disabled.LocalEmbeddingEnabled || disabled.Embedding.Provider != "alibailian" {
		t.Fatalf("local embedding unexpectedly changed the default route: %+v", disabled)
	}

	t.Setenv("BENETNASCH_AI_LOCAL_EMBEDDING", "true")
	t.Setenv("SGLANG_BASE_URL", "http://qwen-embedding:30000/v1")
	t.Setenv("BENETNASCH_AI_LOCAL_EMBEDDING_MODEL_VERSION", "qwen3-embedding-0.6b")
	t.Setenv("BENETNASCH_AI_LOCAL_EMBEDDING_INDEX_VERSION", "qwen3_0_6b_v1")
	t.Setenv("BENETNASCH_AI_LOCAL_EMBEDDING_BATCH_SIZE", "1")
	enabled := new(AI).AI()
	if !enabled.LocalEmbeddingEnabled || enabled.Embedding.Provider != "sglang" || enabled.Embedding.Protocol != "openai_chat_completions" || enabled.Embedding.Model != "Qwen/Qwen3-Embedding-0.6B" {
		t.Fatalf("local embedding route = %+v", enabled)
	}
	if enabled.EmbeddingConfig.ModelVersion != "qwen3-embedding-0.6b" || enabled.EmbeddingConfig.IndexVersion != "qwen3_0_6b_v1" || enabled.EmbeddingConfig.BatchSize != 1 {
		t.Fatalf("local embedding contract = %+v", enabled.EmbeddingConfig)
	}
}

func TestValidateLocalEmbeddingConfigurationFailsClosed(t *testing.T) {
	t.Setenv("BENETNASCH_AI_LOCAL_EMBEDDING", "true")
	t.Setenv("BENETNASCH_AI_ARTICLE_INDEXING", "false")
	t.Setenv("SGLANG_BASE_URL", "")
	t.Setenv("SGLANG_MODEL", "")
	if err := validateLocalEmbeddingConfiguration(); err == nil {
		t.Fatal("missing local embedding endpoint was accepted")
	}

	t.Setenv("SGLANG_BASE_URL", "http://qwen-embedding:30000/v1")
	if err := validateLocalEmbeddingConfiguration(); err == nil {
		t.Fatal("missing local embedding model was accepted")
	}

	t.Setenv("SGLANG_MODEL", "Qwen/Qwen3-Embedding-0.6B")
	t.Setenv("BENETNASCH_AI_ARTICLE_INDEXING", "true")
	if err := validateLocalEmbeddingConfiguration(); err == nil {
		t.Fatal("local indexing without an isolated contract was accepted")
	}

	t.Setenv("BENETNASCH_AI_LOCAL_EMBEDDING_MODEL_VERSION", "qwen3-embedding-0.6b")
	t.Setenv("BENETNASCH_AI_LOCAL_EMBEDDING_INDEX_VERSION", "qwen3_0_6b_v1")
	if err := validateLocalEmbeddingConfiguration(); err != nil {
		t.Fatalf("complete local embedding configuration rejected: %v", err)
	}
}

func TestAIConfigParsesConfiguredSemanticRatio(t *testing.T) {
	key := "ai.search.semantic_ratio"
	previous := viper.Get(key)
	wasSet := viper.IsSet(key)
	t.Cleanup(func() {
		if wasSet {
			viper.Set(key, previous)
			return
		}
		viper.Set(key, nil)
	})
	viper.Set(key, 0.42)
	if got := new(AI).AI().SemanticRatio; got != 0.42 {
		t.Fatalf("configured semantic ratio = %v, want 0.42", got)
	}
}

func TestDatabaseSQLLoggingRequiresExplicitOptIn(t *testing.T) {
	key := "database.show_sql"
	previous := viper.Get(key)
	wasSet := viper.IsSet(key)
	t.Cleanup(func() {
		if wasSet {
			viper.Set(key, previous)
			return
		}
		viper.Set(key, nil)
	})

	viper.Set(key, false)
	if got := new(Database).DataBase().ShowSQL; got {
		t.Fatal("database SQL logging should be disabled by default")
	}
	viper.Set(key, true)
	if got := new(Database).DataBase().ShowSQL; !got {
		t.Fatal("database SQL logging should only enable after explicit opt-in")
	}
}

func TestValidateDatabaseDoesNotRequireUnrelatedRuntimeSecrets(t *testing.T) {
	previous, existed := os.LookupEnv("POSTGRES_PASSWORD")
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv("POSTGRES_PASSWORD", previous)
			return
		}
		_ = os.Unsetenv("POSTGRES_PASSWORD")
	})

	if err := os.Setenv("POSTGRES_PASSWORD", "migration-test-password"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"MEILI_MASTER_KEY", "REDIS_PASSWORD", "SMTP_PASSWORD", "ALIYUN_OSS_ACCESS_KEY_ID", "ALIYUN_OSS_ACCESS_KEY_SECRET"} {
		oldValue, wasSet := os.LookupEnv(name)
		_ = os.Unsetenv(name)
		t.Cleanup(func() {
			if wasSet {
				_ = os.Setenv(name, oldValue)
				return
			}
			_ = os.Unsetenv(name)
		})
	}

	if err := ValidateDatabase(); err != nil {
		t.Fatalf("ValidateDatabase() = %v, want database-only validation to pass", err)
	}
}

func TestValidateDatabaseReportsOnlyMissingDatabaseSecret(t *testing.T) {
	previous, existed := os.LookupEnv("POSTGRES_PASSWORD")
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv("POSTGRES_PASSWORD", previous)
			return
		}
		_ = os.Unsetenv("POSTGRES_PASSWORD")
	})
	_ = os.Unsetenv("POSTGRES_PASSWORD")

	err := ValidateDatabase()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_PASSWORD") {
		t.Fatalf("ValidateDatabase() = %v, want missing POSTGRES_PASSWORD", err)
	}
	if strings.Contains(err.Error(), "MEILI_MASTER_KEY") || strings.Contains(err.Error(), "SMTP_PASSWORD") {
		t.Fatalf("ValidateDatabase() leaked unrelated requirements: %v", err)
	}
}

func preserveDatabaseConfigForTest(t *testing.T) {
	t.Helper()
	previousHost, hostWasSet := viper.Get("database.host"), viper.IsSet("database.host")
	previousPort, portWasSet := viper.Get("database.port"), viper.IsSet("database.port")
	t.Cleanup(func() {
		if hostWasSet {
			viper.Set("database.host", previousHost)
		} else {
			viper.Set("database.host", nil)
		}
		if portWasSet {
			viper.Set("database.port", previousPort)
		} else {
			viper.Set("database.port", nil)
		}
	})
}

func TestIntegrationDatabaseOverridesUseLoopbackHostAndPublishedPort(t *testing.T) {
	preserveDatabaseConfigForTest(t)
	t.Setenv("BENETNASCH_ENV", "integration")
	t.Setenv("BENETNASCH_DATABASE_HOST", "127.0.0.1")
	t.Setenv("BENETNASCH_DATABASE_PORT", "15432")
	viper.Set("database.host", "postgresql")
	viper.Set("database.port", 5432)

	if err := applyIntegrationDatabaseOverrides(); err != nil {
		t.Fatalf("applyIntegrationDatabaseOverrides() error = %v", err)
	}
	if got := viper.GetString("database.host"); got != "127.0.0.1" {
		t.Fatalf("database host = %q, want 127.0.0.1", got)
	}
	if got := viper.GetInt("database.port"); got != 15432 {
		t.Fatalf("database port = %d, want 15432", got)
	}
}

func TestIntegrationDatabaseOverridesRejectRemoteOrNonIntegrationTargets(t *testing.T) {
	tests := []struct {
		name string
		env  string
		host string
		port string
	}{
		{name: "remote host", env: "integration", host: "192.0.2.10", port: "15432"},
		{name: "non integration", env: "prod", host: "127.0.0.1", port: "15432"},
		{name: "invalid port", env: "integration", host: "127.0.0.1", port: "not-a-port"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			preserveDatabaseConfigForTest(t)
			t.Setenv("BENETNASCH_ENV", test.env)
			t.Setenv("BENETNASCH_DATABASE_HOST", test.host)
			t.Setenv("BENETNASCH_DATABASE_PORT", test.port)
			if err := applyIntegrationDatabaseOverrides(); err == nil {
				t.Fatal("applyIntegrationDatabaseOverrides() error = nil, want rejection")
			}
		})
	}
}

func TestValidateMeiliSearchDoesNotRequireUnrelatedRuntimeSecrets(t *testing.T) {
	t.Setenv("MEILI_MASTER_KEY", "meili-test-key")
	previous := map[string]any{}
	wasSet := map[string]bool{}
	for key, value := range map[string]any{
		"meiliSearch.host":   "127.0.0.1",
		"meiliSearch.port":   17700,
		"meiliSearch.apiKey": "meili-test-key",
	} {
		previous[key] = viper.Get(key)
		wasSet[key] = viper.IsSet(key)
		viper.Set(key, value)
		key := key
		t.Cleanup(func() {
			if wasSet[key] {
				viper.Set(key, previous[key])
				return
			}
			viper.Set(key, nil)
		})
	}

	if err := ValidateMeiliSearch(); err != nil {
		t.Fatalf("ValidateMeiliSearch() = %v, want Meilisearch-only validation to pass", err)
	}
}

func TestValidateMeiliSearchReportsMissingMasterKeyWithoutSecretDetails(t *testing.T) {
	t.Setenv("MEILI_MASTER_KEY", "")
	err := ValidateMeiliSearch()
	if err == nil || !strings.Contains(err.Error(), "MEILI_MASTER_KEY") {
		t.Fatalf("ValidateMeiliSearch() = %v, want missing MEILI_MASTER_KEY", err)
	}
	if strings.Contains(err.Error(), "meili-test-key") {
		t.Fatalf("ValidateMeiliSearch() exposed a secret: %v", err)
	}
}
