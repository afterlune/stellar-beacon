#!/usr/bin/env bash
set -euo pipefail

if rg -n --glob '*.go' \
  'SQL\([^)]*(fmt\.Sprintf|\+)|Exec\([^)]*(fmt\.Sprintf|\+)|Where\([^)]*(fmt\.Sprintf|\+)' \
  app; then
  echo "dynamic SQL construction detected; use placeholders and bound arguments" >&2
  exit 1
fi

if rg -ni --glob '*.go' \
  "like[[:space:]]*['\"][^'\"]*['\"][[:space:]]*\+|where[[:space:]]*['\"][^'\"]*['\"][[:space:]]*\+" \
  app; then
  echo "SQL string concatenation detected" >&2
  exit 1
fi
