#!/usr/bin/env bash
set -euo pipefail

files=(
  "internal/application/service/article_read_service.go"
  "internal/application/service/article_admin_service.go"
)

if rg -n '"github.com/gin-gonic/gin"|internal/interfaces/http' "${files[@]}"; then
  echo 'typed article application use cases must not depend on Gin or HTTP presentation packages' >&2
  exit 1
fi

echo 'typed article application use cases are independent of HTTP presentation'
