#!/usr/bin/env bash

set -euo pipefail

file="scripts/production-backup-evidence.ps1"

if [[ ! -f "$file" ]]; then
  echo "Production backup evidence script is missing: $file" >&2
  exit 1
fi

for required in \
  "ValidateSet('Plan', 'Verify')" \
  "benetnasch.production-backup.v1" \
  "restoreVerified" \
  "Get-RequiredBoolean" \
  "restoreVerifiedAtUtc is unexpectedly in the future" \
  "retentionUntilUtc" \
  "Production PostgreSQL backup evidence passed (read-only verification)" \
  "never connects to PostgreSQL, Docker, or a production service"; do
  if ! rg -q --fixed-strings -- "$required" "$file"; then
    echo "Production backup evidence script is missing safety invariant: $required" >&2
    exit 1
  fi
done

if rg -n -- "docker|pg_dump|pg_restore|Invoke-RestMethod|Invoke-WebRequest|Set-Content|Add-Content|New-Item|Move-Item|Copy-Item|Remove-Item|Clear-Item|Delete" "$file"; then
  echo "Production backup evidence script must remain read-only and must not invoke infrastructure or file mutation commands." >&2
  exit 1
fi

echo "Production backup evidence script is read-only and keeps database operations outside the repository."
