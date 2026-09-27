#!/usr/bin/env bash
set -euo pipefail

file="internal/application/service/article_read_service.go"

if rg -n '"github.com/gin-gonic/gin"|internal/interfaces/http' "$file"; then
  echo 'typed public article reader must not depend on Gin or HTTP presentation packages' >&2
  exit 1
fi

echo 'typed public article reader is independent of HTTP presentation'
