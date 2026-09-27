#!/usr/bin/env bash
set -euo pipefail

if rg -n --glob '*.go' --glob '!**/*_test.go' 'internal/interfaces/http' internal/infrastructure; then
  echo 'infrastructure adapters must not depend on HTTP presentation packages' >&2
  exit 1
fi

echo 'infrastructure adapters do not depend on HTTP presentation packages'
