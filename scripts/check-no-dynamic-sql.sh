#!/usr/bin/env bash
set -euo pipefail

if rg -n --glob '*.go' '(SQL|Exec|Where|And|Or)\([^)]*fmt\.Sprintf' app; then
  echo "dynamic SQL construction detected; use placeholders and bound arguments" >&2
  exit 1
fi

if rg -n --glob '*.go' \
  '(SQL|Exec|Where|And|Or)\([^)]*\+[^)]*(vo\.|filter\.|keywords|Keywords|username|Username|nickname|Nickname|topic|Topic)' \
  app; then
  echo "request data must not be concatenated into SQL" >&2
  exit 1
fi

if rg -n --glob '*.go' \
  '"[^"]*(SELECT|INSERT|UPDATE|DELETE|WHERE|LIKE|ORDER[[:space:]]+BY|GROUP[[:space:]]+BY)[^"]*"[[:space:]]*\+[[:space:]]*(vo\.|filter\.|keywords|Keywords|username|Username|nickname|Nickname|topic|Topic)' \
  app; then
  echo "request data must not be concatenated into SQL fragments" >&2
  exit 1
fi
