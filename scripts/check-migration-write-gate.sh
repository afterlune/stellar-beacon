#!/usr/bin/env bash

set -euo pipefail

file="cmd/migrate.go"

if [[ ! -f "$file" ]]; then
  echo "Migration command is missing: $file" >&2
  exit 1
fi

for required in \
  'migrateCmd.Flags().Bool("allow-writes"' \
  'requireMigrationWriteAuthorization' \
  'return runMigrations(cmd.Context())'; do
  if ! rg -q --fixed-strings -- "$required" "$file"; then
    echo "Migration command is missing safety invariant: $required" >&2
    exit 1
  fi
done

gate_line=$(rg -n -m 1 'if err := requireMigrationWriteAuthorization' "$file" | cut -d: -f1 || true)
effect_line=$(rg -n -m 1 'return runMigrations\(cmd\.Context\(\)\)' "$file" | cut -d: -f1 || true)
if [[ -z "$gate_line" || -z "$effect_line" || "$gate_line" -ge "$effect_line" ]]; then
  echo "Migration write authorization must run before the migration runner." >&2
  exit 1
fi

if ! rg -q --fixed-strings 'Use:   "status"' "$file" || \
   ! rg -q --fixed-strings 'inspect migration state without modifying the database' "$file"; then
  echo "Read-only migration status command is missing." >&2
  exit 1
fi

echo "Database migration writes require explicit --allow-writes; status remains read-only."
