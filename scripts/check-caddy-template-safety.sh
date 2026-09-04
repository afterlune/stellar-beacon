#!/usr/bin/env bash

set -euo pipefail

file="Caddyfile"
if [[ ! -f "$file" ]]; then
  echo "Caddy template is missing: $file" >&2
  exit 1
fi

# CORS is owned by the application, which has the reviewed exact-origin
# allowlist. The edge template must not reintroduce wildcard credentials or
# bypass the application's OPTIONS decision.
if rg -n --fixed-strings 'Access-Control-Allow-Origin "*"' "$file"; then
  echo "Caddy template must not allow every Origin." >&2
  exit 1
fi
if rg -n --fixed-strings 'Access-Control-Allow-Credentials "true"' "$file"; then
  echo "Caddy template must not independently enable credentialed CORS." >&2
  exit 1
fi
if rg -n --fixed-strings 'method OPTIONS' "$file"; then
  echo "Caddy template must not short-circuit application CORS preflight handling." >&2
  exit 1
fi

echo "Caddy template delegates CORS and preflight decisions to the application."
