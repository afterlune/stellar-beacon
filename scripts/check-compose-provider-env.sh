#!/usr/bin/env bash

set -euo pipefail

provider_vars=(
  OPENAI_API_KEY
  OPENAI_BASE_URL
  OPENAI_MODEL
  ALIBAILIAN_API_KEY
  ALIBAILIAN_BASE_URL
  ALIBAILIAN_MODEL
  ALIBAILIAN_MODEL2
)

service_block() {
  local file="$1"
  local service="$2"

  awk -v service="$service" '
    $0 ~ "^  " service ":[[:space:]]*$" { in_service = 1; next }
    in_service && $0 ~ /^  [A-Za-z0-9_.-]+:[[:space:]]*$/ { exit }
    in_service { print }
  ' "$file"
}

assert_integration_compose() {
  local file="$1"
  local backend_block minio_block variable

  backend_block="$(service_block "$file" backend)"
  minio_block="$(service_block "$file" minio)"
  [[ -n "$backend_block" ]] || { echo "backend service is missing: $file" >&2; exit 1; }
  [[ -n "$minio_block" ]] || { echo "minio service is missing: $file" >&2; exit 1; }

  for variable in "${provider_vars[@]}"; do
    if ! grep -Eq "^      ${variable}:" <<<"$backend_block"; then
      echo "$variable must be injected into the integration backend: $file" >&2
      exit 1
    fi
    if grep -Eq "^      ${variable}:" <<<"$minio_block"; then
      echo "$variable must not be injected into the integration MinIO service: $file" >&2
      exit 1
    fi
  done
}

assert_production_compose() {
  local file="$1"
  local backend_block variable

  backend_block="$(service_block "$file" benetnasch)"
  [[ -n "$backend_block" ]] || { echo "benetnasch service is missing: $file" >&2; exit 1; }

  for variable in "${provider_vars[@]}"; do
    if ! grep -Eq "^[[:space:]]+- ${variable}=" <<<"$backend_block"; then
      echo "$variable must be injected into the production backend: $file" >&2
      exit 1
    fi
  done
}

assert_integration_compose docker-compose.integration.yaml
assert_production_compose docker-compose.yaml

echo 'Compose Provider variables are scoped to backend services.'
