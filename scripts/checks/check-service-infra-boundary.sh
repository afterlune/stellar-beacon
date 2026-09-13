#!/usr/bin/env bash
set -euo pipefail

if rg -n --glob '*.go' '"stellar-beacon/internal/infrastructure/' internal/application/service; then
  echo "application services must depend on domain ports/support, not infra packages" >&2
  exit 1
fi
