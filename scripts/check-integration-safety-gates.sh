#!/usr/bin/env bash

set -euo pipefail

assert_gate() {
  local file="$1"
  shift
  local first_effect gate_line gate

  if [[ ! -f "$file" ]]; then
    echo "Integration script is missing: $file" >&2
    exit 1
  fi

  first_effect=$(rg -n -m 1 \
    'Invoke-RequiredCommand|Invoke-IntegrationCompose|Invoke-IntegrationRequest|Invoke-RestMethod|curl\.exe|go run|npm' \
    "$file" | cut -d: -f1 || true)
  gate_line=$(rg -n -m 1 'if \(-not \$Allow|if \(-not \$AllowContainerChanges' "$file" | cut -d: -f1 || true)

  if [[ -z "$gate_line" || -z "$first_effect" || "$gate_line" -ge "$first_effect" ]]; then
    echo "Integration side-effect gate must run before the first external operation: $file" >&2
    exit 1
  fi

  for gate in "$@"; do
    if ! rg -q --fixed-strings -- "$gate" "$file"; then
      echo "Integration script $file is missing explicit gate $gate" >&2
      exit 1
    fi
  done
}

assert_provider_flag_gate() {
  local file="$1"
  local enabled_line probe_line login_line

  enabled_line=$(rg -n -m 1 "Assert-EnabledFlag -Name 'BENETNASCH_AI_ENABLED'" "$file" | cut -d: -f1 || true)
  probe_line=$(rg -n -m 1 "Assert-EnabledFlag -Name 'BENETNASCH_AI_PROVIDER_PROBE'" "$file" | cut -d: -f1 || true)
  login_line=$(rg -n -m 1 '^function Get-AdminToken' "$file" | cut -d: -f1 || true)

  if [[ -z "$enabled_line" || -z "$probe_line" || -z "$login_line" || "$enabled_line" -ge "$login_line" || "$probe_line" -ge "$login_line" ]]; then
    echo "Provider rollout flags must be checked before isolated admin login: $file" >&2
    exit 1
  fi
}

assert_vision_eval_gate() {
  local file="scripts/integration-vision-eval.ps1"
  local enabled_line vision_line key_line login_line request_line

  if [[ ! -f "$file" ]]; then
    echo "Integration Vision evaluation script is missing: $file" >&2
    exit 1
  fi

  enabled_line=$(rg -n -m 1 "Assert-EnabledFlag -Name 'BENETNASCH_AI_ENABLED'" "$file" | cut -d: -f1 || true)
  vision_line=$(rg -n -m 1 "Assert-EnabledFlag -Name 'BENETNASCH_AI_VISION'" "$file" | cut -d: -f1 || true)
  key_line=$(rg -n -m 1 'OPENAI_API_KEY must be set' "$file" | cut -d: -f1 || true)
  login_line=$(rg -n -m 1 '^function Get-AdminToken' "$file" | cut -d: -f1 || true)
  request_line=$(rg -n -m 1 'api/admin/ai/vision/preview' "$file" | cut -d: -f1 || true)

  if [[ -z "$enabled_line" || -z "$vision_line" || -z "$key_line" || -z "$login_line" || -z "$request_line" ||
    "$enabled_line" -ge "$login_line" || "$vision_line" -ge "$login_line" || "$key_line" -ge "$login_line" ||
    "$login_line" -ge "$request_line" ]]; then
    echo "Vision evaluation must validate rollout flags and the provider key before isolated admin login/request: $file" >&2
    exit 1
  fi

  if ! rg -q --fixed-strings 'Resolve-ExternalPath -Path $outputParent' "$file" ||
    ! rg -q --fixed-strings 'OutputPath already exists' "$file" ||
    ! rg -q --fixed-strings 'FixtureManifestPath' "$file" ||
    ! rg -q --fixed-strings 'Resolve-FixtureFile' "$file"; then
    echo "Vision evaluation must keep fixtures/output outside the repository and refuse output overwrite: $file" >&2
    exit 1
  fi
}

assert_integration_env_isolation() {
  local file="scripts/integration-common.ps1"
  local read_line reset_line name

  if [[ ! -f "$file" ]]; then
    echo "Integration environment loader is missing: $file" >&2
    exit 1
  fi

  read_line=$(rg -n -m 1 'foreach \(\$line in Get-Content -LiteralPath \$script:IntegrationEnvFile\)' "$file" | cut -d: -f1 || true)
  if [[ -z "$read_line" ]]; then
    echo "Integration environment loader no longer reads the explicit env file as expected: $file" >&2
    exit 1
  fi

  for name in \
    POSTGRES_PASSWORD REDIS_PASSWORD MEILI_MASTER_KEY SMTP_PASSWORD \
    MINIO_ROOT_USER MINIO_ROOT_PASSWORD \
    E2E_USER_EMAIL E2E_USER_PASSWORD E2E_ADMIN_EMAIL E2E_ADMIN_PASSWORD E2E_ADMIN_TOKEN \
    OPENAI_API_KEY OPENAI_BASE_URL OPENAI_MODEL \
    ALIBAILIAN_API_KEY ALIBAILIAN_BASE_URL ALIBAILIAN_MODEL ALIBAILIAN_MODEL2 \
    SGLANG_API_KEY SGLANG_BASE_URL SGLANG_MODEL SGLANG_MODEL_ID \
    BENETNASCH_AI_ENABLED BENETNASCH_AI_PROVIDER_PROBE BENETNASCH_AI_LOCAL_EMBEDDING \
    BENETNASCH_AI_LOCAL_EMBEDDING_MODEL_VERSION BENETNASCH_AI_LOCAL_EMBEDDING_INDEX_VERSION \
    BENETNASCH_AI_LOCAL_EMBEDDING_BATCH_SIZE BENETNASCH_AI_VISION BENETNASCH_AI_ARTICLE_INDEXING \
    E2E_BASE_URL E2E_REAL_INTEGRATION E2E_ADMIN_ALLOW_LOGIN E2E_REQUIRE_AGENT_ROUTES \
    VITE_ADMIN_API_TARGET VITE_ADMIN_API_PRESERVE_API_PREFIX; do
    if ! rg -q --fixed-strings "'$name'" "$file"; then
      echo "Integration environment loader is missing scoped variable: $name" >&2
      exit 1
    fi
  done

  reset_line=$(rg -n -m 1 'Remove-Item "Env:\$name" -ErrorAction SilentlyContinue' "$file" | cut -d: -f1 || true)
  if [[ -z "$reset_line" || "$reset_line" -ge "$read_line" ]]; then
    echo "Integration environment variables must be cleared before reading .env.integration: $file" >&2
    exit 1
  fi

  allow_line=$(rg -n -m 1 'if \(\$name -notin \$script:IntegrationAllowedEnvironmentNames\)' "$file" | cut -d: -f1 || true)
  set_line=$(rg -n -m 1 '\[Environment\]::SetEnvironmentVariable\(\$name' "$file" | cut -d: -f1 || true)
  if [[ -z "$allow_line" || -z "$set_line" || "$allow_line" -ge "$set_line" ]]; then
    echo "Integration environment loader must enforce its allowlist before setting process variables: $file" >&2
    exit 1
  fi
  if ! rg -q --fixed-strings "'INTEGRATION_COMPOSE_OVERRIDE'" "$file"; then
    echo "Integration environment loader must explicitly scope INTEGRATION_COMPOSE_OVERRIDE: $file" >&2
    exit 1
  fi
  if ! rg -q --fixed-strings 'IntegrationIgnoredEnvironmentNames' "$file" ||
    ! rg -q --fixed-strings 'if ($name -in $script:IntegrationIgnoredEnvironmentNames)' "$file"; then
    echo "Integration environment loader must explicitly ignore only documented legacy variables: $file" >&2
    exit 1
  fi
}

assert_readonly_search_env_loader() {
  local file="scripts/integration-search-readonly.ps1"
  local load_line request_line

  if [[ ! -f "$file" ]]; then
    echo "Read-only integration search script is missing: $file" >&2
    exit 1
  fi

  load_line=$(rg -n -m 1 'Import-IntegrationEnv' "$file" | cut -d: -f1 || true)
  request_line=$(rg -n -m 1 'Invoke-RestMethod' "$file" | cut -d: -f1 || true)
  if [[ -z "$load_line" || -z "$request_line" || "$load_line" -ge "$request_line" ]]; then
    echo "Read-only integration search must load the scoped env before Meilisearch requests: $file" >&2
    exit 1
  fi
}

assert_readonly_article_plan_gate() {
  local file="scripts/integration-article-index-plan.ps1"
  local target_line command_line

  if [[ ! -f "$file" ]]; then
    echo "Read-only integration article-index plan script is missing: $file" >&2
    exit 1
  fi

  target_line=$(rg -n -m 1 'Set-IntegrationLocalDatabaseTarget' "$file" | cut -d: -f1 || true)
  command_line=$(rg -n -m 1 "& go @arguments" "$file" | cut -d: -f1 || true)
  if [[ -z "$target_line" || -z "$command_line" || "$target_line" -ge "$command_line" ]]; then
    echo "Read-only integration article-index plan must set the loopback database target before go run: $file" >&2
    exit 1
  fi
  for required in '[int]$PageSize = 1' '[ValidateRange(1, 25)]'; do
    if ! rg -F -q -- "$required" "$file"; then
      echo "Read-only integration article-index plan must retain the low-memory page budget: $file (missing $required)" >&2
      exit 1
    fi
  done
  if rg -n -- '--allow-writes|Invoke-IntegrationCompose|Invoke-RestMethod|Invoke-IntegrationRequest' "$file"; then
    echo "Read-only integration article-index plan must not write or call external services: $file" >&2
    exit 1
  fi
}

assert_backfill_provider_gate() {
  local file="scripts/integration-article-index-backfill.ps1"
  local key_line target_line config_line configured_line guard_line command_line

  if [[ ! -f "$file" ]]; then
    echo "Integration article-index backfill script is missing: $file" >&2
    exit 1
  fi

  key_line=$(rg -n -m 1 'ALIBAILIAN_API_KEY must be set' "$file" | cut -d: -f1 || true)
  target_line=$(rg -F -n -m 1 'indexes/$IndexUid' "$file" | cut -d: -f1 || true)
  config_line=$(rg -F -n -m 1 '$sourceConfig = Join-Path $repoRoot' "$file" | cut -d: -f1 || true)
  configured_line=$(rg -F -n -m 1 '$configuredIndexUid = ' "$file" | cut -d: -f1 || true)
  guard_line=$(rg -F -n -m 1 'if ($IndexUid -ne $configuredIndexUid)' "$file" | cut -d: -f1 || true)
  command_line=$(rg -F -n -m 1 "'article-index', 'backfill', \$Action" "$file" | cut -d: -f1 || true)
  if [[ -z "$key_line" || -z "$target_line" || -z "$config_line" || -z "$configured_line" ||
        -z "$guard_line" || -z "$command_line" || "$key_line" -ge "$command_line" ||
        "$target_line" -ge "$command_line" || "$config_line" -ge "$command_line" ||
        "$configured_line" -ge "$command_line" || "$guard_line" -ge "$command_line" ]]; then
    echo "Article-index backfill must check the embedding key, configured target, and isolated target before writes: $file" >&2
    exit 1
  fi
  for required in \
    '[int]$PageSize = 1' \
    '[int]$MaxArticlesPerRun = 1' \
    "'--max-articles-per-run'" \
    "'--run-timeout'"; do
    if ! rg -F -q -- "$required" "$file"; then
      echo "Article-index backfill must retain the low-memory bounded-run defaults: $file (missing $required)" >&2
      exit 1
    fi
  done
}

assert_swap_quality_gate() {
  local file="scripts/integration-article-index-swap.ps1"
  local evidence_line compose_line

  if [[ ! -f "$file" ]]; then
    echo "Integration article-index swap script is missing: $file" >&2
    exit 1
  fi

  evidence_line=$(rg -n -m 1 "Quality evidence did not pass all required gates|QualityEvidencePath is required" "$file" | cut -d: -f1 || true)
  compose_line=$(rg -n -m 1 "'article-index', 'swap'" "$file" | cut -d: -f1 || true)
  if [[ -z "$evidence_line" || -z "$compose_line" || "$evidence_line" -ge "$compose_line" ]]; then
    echo "Article-index swap must validate quality evidence before the Docker swap command: $file" >&2
    exit 1
  fi
  if ! rg -q --fixed-strings 'function Get-RequiredBoolean' "$file"; then
    echo "Article-index swap must require native JSON booleans for quality evidence: $file" >&2
    exit 1
  fi
}

assert_restore_gate() {
  local file="$1"
  local restore_gate_line restore_effect_line

  restore_gate_line=$(rg -n -m 1 'if \(\$AllowRestore\)' "$file" | cut -d: -f1 || true)
  restore_effect_line=$(rg -n -m 1 "'createdb'|'pg_restore'" "$file" | cut -d: -f1 || true)

  if [[ -z "$restore_gate_line" || -z "$restore_effect_line" || "$restore_gate_line" -ge "$restore_effect_line" ]]; then
    echo "Restore operations must remain behind the explicit AllowRestore gate: $file" >&2
    exit 1
  fi
}

assert_readonly_browser_gate() {
  local file="scripts/integration-browser-e2e.ps1"

  if [[ ! -f "$file" ]]; then
    echo "Read-only browser integration script is missing: $file" >&2
    exit 1
  fi
  if ! rg -q --fixed-strings -- "'@readonly'" "$file" ||
    ! rg -q --fixed-strings -- '-ReadOnly' "$file"; then
    echo "Admin read-only browser integration must select only @readonly tests: $file" >&2
    exit 1
  fi
}

assert_native_backend_gate() {
  local file="scripts/integration-native-backend.ps1"
  local gate_line effect_line

  if [[ ! -f "$file" ]]; then
    echo "Native integration backend script is missing: $file" >&2
    exit 1
  fi
  gate_line=$(rg -n -m 1 'if \(-not \$AllowContainerChanges\)' "$file" | cut -d: -f1 || true)
  effect_line=$(rg -n -m 1 "'cp'|'restart'|Invoke-RequiredCommand" "$file" | cut -d: -f1 || true)
  if [[ -z "$gate_line" || -z "$effect_line" || "$gate_line" -ge "$effect_line" ]]; then
    echo "Native backend replacement must gate container/build side effects before use: $file" >&2
    exit 1
  fi
  for required in \
    "benetnasch-integration" \
    "com.docker.compose.project" \
    "com.docker.compose.service" \
    "'cp'" \
    "'restart'" \
    "config-integration.yaml" \
    "must be outside the repository" \
    ".codebuddy"; do
    if ! rg -q --fixed-strings -- "$required" "$file"; then
      echo "Native backend replacement is missing isolation/rollback guard: $file (missing $required)" >&2
      exit 1
    fi
  done
  if rg -q -- '--allow-writes|Invoke-IntegrationRequest|Invoke-RestMethod|pg_restore|/indexes/' "$file"; then
    echo "Native backend replacement must not perform data/index operations: $file" >&2
    exit 1
  fi
}

assert_gate scripts/integration-deploy.ps1 '-AllowContainerChanges' '-AllowWrites'
assert_gate scripts/integration-up.ps1 '-AllowContainerChanges'
assert_gate scripts/integration-down.ps1 '-AllowContainerChanges'
assert_gate scripts/integration-migrate.ps1 '-AllowWrites'
assert_gate scripts/integration-repair-sequences.ps1 '-AllowWrites'
assert_gate scripts/integration-seed.ps1 '-AllowWrites'
assert_gate scripts/integration-smoke.ps1 '-AllowWrites'
assert_gate scripts/admin-next-crud-e2e.ps1 '-AllowWrites'
assert_gate scripts/integration-provider-smoke.ps1 '-AllowWrites' '-AllowExternalProviderCalls'
assert_provider_flag_gate scripts/integration-provider-smoke.ps1
assert_gate scripts/integration-vision-eval.ps1 '-AllowWrites' '-AllowExternalProviderCalls'
assert_vision_eval_gate
assert_integration_env_isolation
assert_readonly_search_env_loader
assert_readonly_article_plan_gate
assert_gate scripts/integration-article-index-backfill.ps1 '-AllowWrites' '-AllowExternalProviderCalls'
assert_backfill_provider_gate
assert_gate scripts/integration-article-index-swap.ps1 '-AllowWrites'
assert_swap_quality_gate
assert_gate scripts/integration-backup-restore.ps1 '-AllowWrites' '-AllowRestore'
assert_restore_gate scripts/integration-backup-restore.ps1
assert_gate scripts/integration-migration-failure-recovery.ps1 '-AllowWrites'
assert_readonly_browser_gate
assert_gate scripts/integration-native-backend.ps1 '-AllowContainerChanges'
assert_native_backend_gate

echo 'Integration side-effect scripts have explicit, early safety gates.'
