#!/usr/bin/env bash
set -euo pipefail

mapfile -t files < <(rg --files app/application -g '*.go')
if ((${#files[@]} == 0)); then
  echo 'no application Go files found'
  exit 0
fi

if rg -n \
  'benetnasch/app/infra/(ai|search|shared|oss)|github\.com/cloudwego/eino|github\.com/openai|anthropic' \
  "${files[@]}"; then
  echo 'application and domain code must depend on project ports, not AI/provider SDKs or infra adapters' >&2
  exit 1
fi

echo 'application AI/provider dependency boundary is clean'
