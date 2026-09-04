#!/usr/bin/env bash

set -euo pipefail

seed_components="$({
  rg -o --no-filename "'/[A-Za-z0-9_-]+/[A-Za-z0-9_-]+\.vue'" \
    app/infra/persistence/migration/migrations/*.sql || true
} | tr -d "'" | sort -u)"
registry_components="$({
  rg -o --no-filename "'/[A-Za-z0-9_-]+/[A-Za-z0-9_-]+\.vue'" \
    web/admin-next/src/router/menu.ts || true
} | tr -d "'" | sort -u)"

if [[ -z "$seed_components" ]]; then
  echo "No seeded admin Vue components were found; refusing to pass the registry check." >&2
  exit 1
fi
if [[ -z "$registry_components" ]]; then
  echo "No admin-next Vue components were found in the registry." >&2
  exit 1
fi

missing=$(comm -23 \
  <(printf '%s\n' "$seed_components") \
  <(printf '%s\n' "$registry_components"))
if [[ -n "$missing" ]]; then
  echo "Admin menu components missing from web/admin-next/src/router/menu.ts:" >&2
  printf '%s\n' "$missing" >&2
  exit 1
fi

placeholder_components="$(
  rg -o "'/[A-Za-z0-9_-]+/[A-Za-z0-9_-]+\.vue'[[:space:]]*:[[:space:]]*PlaceholderView" \
    web/admin-next/src/router/menu.ts || true
  )"
if [[ -n "$placeholder_components" ]]; then
  placeholder_components="$(printf '%s\n' "$placeholder_components" | sed -E "s/^'([^']+)'.*/\1/" | sort -u)"
  seeded_placeholders=$(comm -12 \
    <(printf '%s\n' "$seed_components") \
    <(printf '%s\n' "$placeholder_components"))
  if [[ -n "$seeded_placeholders" ]]; then
    echo "Seeded admin Vue components still resolve to PlaceholderView:" >&2
    printf '%s\n' "$seeded_placeholders" >&2
    exit 1
  fi
fi

echo "All seeded admin menu Vue components are registered in admin-next."
