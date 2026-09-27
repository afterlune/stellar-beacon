#!/usr/bin/env bash
set -euo pipefail

if rg -n --glob '*.go' '(SQL|Exec|Where|And|Or)\([^)]*fmt\.Sprintf' internal; then
  echo "dynamic SQL construction detected; use placeholders and bound arguments" >&2
  exit 1
fi

if rg -n --glob '*.go' \
  '(SQL|Exec|Where|And|Or)\([^)]*("[^"]*"|`[^`]*`)\s*\+\s*(vo\.|filter\.|keywords|Keywords|username|Username|nickname|Nickname|topic|Topic)' \
  internal; then
  echo "request data must not be concatenated into SQL" >&2
  exit 1
fi

if rg -n --glob '*.go' \
  '"(?i:[^"]*\b(SELECT|INSERT INTO|UPDATE|DELETE FROM|WITH|WHERE|AND|OR|ORDER BY|LIMIT|OFFSET)\b[^"]*)"[[:space:]]*\+[[:space:]]*(vo\.|filter\.|keywords|Keywords|username|Username|nickname|Nickname|topic|Topic)' \
  internal; then
  echo "request data must not be concatenated into SQL fragments" >&2
  exit 1
fi
