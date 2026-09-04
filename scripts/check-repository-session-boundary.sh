#!/usr/bin/env bash
set -euo pipefail

# All persistence adapters must obtain query sessions through the local
# helpers. engine.go is the single implementation point for repoSession and
# repoTx, so it is intentionally excluded from this check.
mapfile -t files < <(
  rg --files app/infra/persistence/repository -g '*.go' -g '!*_test.go' |
    rg -v '(^|/)engine\.go$'
)

if ((${#files[@]} > 0)) && rg -n 'ormInit|WithEngineTx|GetEngine|NewSession\(|engine\.(SQL|Context)\(' "${files[@]}"; then
  echo 'persistence repositories must use repoSession/repoTx for database access' >&2
  exit 1
fi

echo 'persistence repository session boundary is clean'
