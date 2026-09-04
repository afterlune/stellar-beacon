#!/usr/bin/env bash

set -euo pipefail

files=(
  ".env.example"
  ".env.integration.example"
)
for file in "${files[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "Environment template is missing: $file" >&2
    exit 1
  fi
done

# The tracked template may document names and non-sensitive defaults, but every
# credential-like variable must stay empty. Real values belong in an ignored
# local env file or an external secret store.
credential_pattern='^(POSTGRES_PASSWORD|REDIS_PASSWORD|MEILI_MASTER_KEY|SMTP_PASSWORD|ALIYUN_OSS_ACCESS_KEY_ID|ALIYUN_OSS_ACCESS_KEY_SECRET|JWT_PRIVATE_KEY|JWT_PUBLIC_KEY|MINIO_ROOT_USER|MINIO_ROOT_PASSWORD|OPENAI_API_KEY|ALIBAILIAN_API_KEY|SGLANG_API_KEY)=[^[:space:]]+'
if rg -n --pcre2 "$credential_pattern" ".env.example"; then
  echo "Tracked environment template contains a non-empty credential-like value: .env.example" >&2
  exit 1
fi

integration_provider_pattern='^(OPENAI_API_KEY|ALIBAILIAN_API_KEY|SGLANG_API_KEY)=[^[:space:]]+'
if rg -n --pcre2 "$integration_provider_pattern" ".env.integration.example"; then
  echo "Integration environment template contains a non-empty external Provider key." >&2
  exit 1
fi

echo "Tracked environment templates contain no non-empty credentials."
