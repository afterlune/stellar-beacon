#!/usr/bin/env bash
set -euo pipefail

if rg -n -i \
  --glob '!**/node_modules/**' \
  --glob '!**/dist/**' \
  'TencentCaptcha|TCaptcha\.js|captcha\.qq\.com|TENCENT_CAPTCHA' \
  web README.md docs; then
  echo "Tencent CAPTCHA references must not be committed." >&2
  exit 1
fi

echo "No Tencent CAPTCHA references found."
