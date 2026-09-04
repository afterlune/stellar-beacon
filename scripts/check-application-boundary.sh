#!/usr/bin/env bash
set -euo pipefail

mapfile -t files < <(rg --files app/application -g '*.go' -g '!**/*_test.go')
if ((${#files[@]} == 0)); then
  echo 'no production application Go files found'
  exit 0
fi

if rg -n \
  'benetnasch/app/facade|benetnasch/app/domain/entity|github\.com/gin-gonic/gin|xorm\.io/xorm|benetnasch/app/infra/' \
  "${files[@]}"; then
  echo 'application production code must not depend on facade, domain entity aliases, Gin, xorm, or infra adapters' >&2
  exit 1
fi

domain_files=(app/domain/entity/*.go)
if ((${#domain_files[@]} > 0)) && rg -n 'xorm:' "${domain_files[@]}"; then
  echo 'domain entities must not contain xorm persistence metadata; use infra/persistence/row instead' >&2
  exit 1
fi

echo 'application and domain persistence boundaries are clean'
