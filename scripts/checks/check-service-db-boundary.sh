#!/usr/bin/env bash
set -euo pipefail

mapfile -t files < <(rg --files internal/application/service -g '*.go')

if rg -n 'ormInit|NewSession|GetEngine|xorm\.io/(xorm|builder)|internal/infrastructure/persistence' "${files[@]}"; then
  echo 'application services must access persistence only through domain ports' >&2
  exit 1
fi
