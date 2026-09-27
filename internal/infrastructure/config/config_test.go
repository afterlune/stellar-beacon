package config

import "testing"

func TestEscapeYAMLDoubleQuoted(t *testing.T) {
	input := "value with \"quotes\", \\slashes\\, and\nline"
	want := "value with \\\"quotes\\\", \\\\slashes\\\\, and\\nline"
	if got := escapeYAMLDoubleQuoted(input); got != want {
		t.Fatalf("escapeYAMLDoubleQuoted() = %q, want %q", got, want)
	}
}

func TestConfiguredValuePrefersRenamedEnvironmentVariable(t *testing.T) {
	t.Setenv("STELLAR_BEACON_ENV", "integration")
	t.Setenv("BENETNASCH_ENV", "dev")

	if got := configuredValue("STELLAR_BEACON_ENV", "BENETNASCH_ENV"); got != "integration" {
		t.Fatalf("configuredValue() = %q, want renamed value", got)
	}
}

func TestConfiguredValueFallsBackToFormerEnvironmentVariable(t *testing.T) {
	t.Setenv("STELLAR_BEACON_CONFIG_DIR", "")
	t.Setenv("BENETNASCH_CONFIG_DIR", t.TempDir())

	got := configuredValue("STELLAR_BEACON_CONFIG_DIR", "BENETNASCH_CONFIG_DIR")
	if got == "" {
		t.Fatal("configuredValue() did not return the former environment variable")
	}
}

func TestFindDirectoryAcceptsFormerEnvironmentPrefix(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("STELLAR_BEACON_CONFIG_DIR", "")
	t.Setenv("BENETNASCH_CONFIG_DIR", directory)

	got, ok := findDirectory("STELLAR_BEACON_CONFIG_DIR", "unused-default")
	if !ok || got != directory {
		t.Fatalf("findDirectory() = (%q, %t), want (%q, true)", got, ok, directory)
	}
}

func TestStandaloneStorageProfilesUseGenericCredentials(t *testing.T) {
	for _, provider := range []string{"minio", "aliyun"} {
		required := requiredEnvironmentVarsFor(provider, "prod-standalone", "/keys", "false")
		if !containsString(required, "OBJECT_STORAGE_ACCESS_KEY_ID") || !containsString(required, "OBJECT_STORAGE_ACCESS_KEY_SECRET") {
			t.Fatalf("standalone %s profile does not require generic object-storage credentials: %v", provider, required)
		}
		if containsString(required, "ALIYUN_OSS_ACCESS_KEY_ID") || containsString(required, "ALIYUN_OSS_ACCESS_KEY_SECRET") {
			t.Fatalf("standalone %s profile unexpectedly requires legacy Aliyun variables: %v", provider, required)
		}
		if containsString(required, "JWT_PRIVATE_KEY") || containsString(required, "JWT_PUBLIC_KEY") {
			t.Fatalf("persistent JWT directory should replace environment key pairs: %v", required)
		}
	}
}

func TestLegacyAliyunProfileStillRequiresAliyunCredentials(t *testing.T) {
	required := requiredEnvironmentVarsFor("aliyun", "prod", "", "false")
	if !containsString(required, "ALIYUN_OSS_ACCESS_KEY_ID") || !containsString(required, "ALIYUN_OSS_ACCESS_KEY_SECRET") {
		t.Fatalf("legacy Aliyun profile credentials are missing: %v", required)
	}
}

func TestStandaloneAliyunProfileUsesGenericCredentials(t *testing.T) {
	for _, name := range []string{"ALIYUN_OSS_ACCESS_KEY_ID", "ALIYUN_OSS_ACCESS_KEY_SECRET"} {
		if !shouldIgnoreLegacyAliyunCredential(name, "aliyun", "prod-standalone") {
			t.Fatalf("standalone Aliyun profile did not ignore legacy variable %s", name)
		}
	}
	if shouldIgnoreLegacyAliyunCredential("OBJECT_STORAGE_ACCESS_KEY_ID", "aliyun", "prod-standalone") {
		t.Fatal("generic object-storage credential was classified as a legacy Aliyun variable")
	}
	if shouldIgnoreLegacyAliyunCredential("ALIYUN_OSS_ACCESS_KEY_ID", "aliyun", "prod") {
		t.Fatal("legacy production profile stopped requiring its Aliyun credential")
	}
	if !shouldIgnoreLegacyAliyunCredential("ALIYUN_OSS_ACCESS_KEY_ID", "minio", "prod") {
		t.Fatal("MinIO profile should ignore legacy Aliyun credentials")
	}
	required := requiredEnvironmentVarsFor("aliyun", "prod-standalone", "/keys", "false")
	if containsString(required, "ALIYUN_OSS_ACCESS_KEY_ID") || containsString(required, "ALIYUN_OSS_ACCESS_KEY_SECRET") {
		t.Fatalf("standalone Aliyun profile unexpectedly requires legacy variables: %v", required)
	}
	if !containsString(required, "OBJECT_STORAGE_ACCESS_KEY_ID") || !containsString(required, "OBJECT_STORAGE_ACCESS_KEY_SECRET") {
		t.Fatalf("standalone Aliyun profile does not require generic credentials: %v", required)
	}
}

func TestStandaloneSettingsRejectSamplesAndRepeatedSecrets(t *testing.T) {
	values := map[string]string{
		"SITE_DOMAIN":                      "blog.test",
		"ADMIN_DOMAIN":                     "admin.blog.test",
		"ACME_EMAIL":                       "ops@blog.test",
		"POSTGRES_PASSWORD":                "postgres-test-secret-1234567890",
		"REDIS_PASSWORD":                   "redis-test-secret-1234567890",
		"MEILI_MASTER_KEY":                 "meili-test-secret-1234567890",
		"SMTP_HOST":                        "smtp.blog.test",
		"SMTP_EMAIL":                       "blog@blog.test",
		"SMTP_PASSWORD":                    "mail-app-password",
		"OBJECT_STORAGE_PUBLIC_URL":        "https://blog.test/storage/stellar-beacon",
		"OBJECT_STORAGE_ACCESS_KEY_ID":     "minio-test-access-key",
		"OBJECT_STORAGE_ACCESS_KEY_SECRET": "storage-test-secret-1234567890",
	}
	if err := validateStandaloneSettings(values); err != nil {
		t.Fatalf("valid standalone settings were rejected: %v", err)
	}

	values["SITE_DOMAIN"] = "example.com"
	if err := validateStandaloneSettings(values); err == nil {
		t.Fatal("example domain was accepted")
	}
	values["SITE_DOMAIN"] = "blog.test"
	values["REDIS_PASSWORD"] = values["POSTGRES_PASSWORD"]
	if err := validateStandaloneSettings(values); err == nil {
		t.Fatal("reused production secret was accepted")
	}
	values["REDIS_PASSWORD"] = "redis-test-secret-1234567890"
	values["SMTP_PASSWORD"] = "replace-with-an-app-password"
	if err := validateStandaloneSettings(values); err == nil {
		t.Fatal("example SMTP password was accepted")
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
