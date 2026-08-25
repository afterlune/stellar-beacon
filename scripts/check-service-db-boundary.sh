#!/usr/bin/env bash
set -euo pipefail

mapfile -t files < <(rg --files app/application/service -g '*.go')

if rg -n 'ormInit|NewSession|GetEngine|xorm\.io/(xorm|builder)|app/infra/persistence/repository' "${files[@]}"; then
  echo 'application services must access persistence only through domain ports' >&2
  exit 1
fi
