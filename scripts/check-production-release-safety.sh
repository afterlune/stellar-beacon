#!/usr/bin/env bash

set -euo pipefail

file="scripts/production-caddy-admin-switch.ps1"

if [[ ! -f "$file" ]]; then
  echo "Production static switch script is missing: $file" >&2
  exit 1
fi

for required in \
  "ValidateSet('Plan', 'Apply', 'VerifyRetention')" \
  "AllowProductionSwitch" \
  "ConfirmProductionTarget" \
  "ExpectedIndexSha256" \
  "ArchiveRoot" \
  "ArchiveRoot must already exist as a regular directory before Apply" \
  "retentionUntilUtc" \
  "failed-admin" \
  "Assert-NoReparsePoints -Root \$archiveRootPath -Label 'ArchiveRoot'" \
  "Release manifest must be directly under ArchiveRoot/<releaseId>" \
  "Release manifest directory does not match releaseId" \
  "Get-RequiredFile -Path \$manifestFile.FullName" \
  "Assert-Sha256 -Value ([string]\$manifest.previousIndexSha256)" \
  "no Caddy reload"; do
  if ! rg -q --fixed-strings -- "$required" "$file"; then
    echo "Production static switch script is missing safety invariant: $required" >&2
    exit 1
  fi
done

if rg -n -- "Remove-Item|Clear-Item|Delete" "$file"; then
  echo "Production static switch script must never delete release artifacts." >&2
  exit 1
fi

if ! rg -q -- "Apply requires both -AllowProductionSwitch and -ConfirmProductionTarget" "$file"; then
  echo "Production static switch script is missing its double-authorization guard." >&2
  exit 1
fi

if rg -n --fixed-strings -- "New-Item -ItemType Directory -Path \$archiveRootPath" "$file"; then
  echo "Production static switch script must not create an unverified ArchiveRoot." >&2
  exit 1
fi

echo "Production static switch script retains explicit authorization, archive, hash, and no-delete gates."
