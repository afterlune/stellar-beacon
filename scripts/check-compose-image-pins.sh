#!/usr/bin/env bash

set -euo pipefail

compose_files=(
  docker-compose.yaml
  docker-compose.integration.yaml
)

checked=0
for file in "${compose_files[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "Compose file is missing: $file" >&2
    exit 1
  fi

  while IFS= read -r image; do
    [[ -n "$image" ]] || continue

    # The application image is a local build tag. Third-party service images
    # must use both a human-readable version and an immutable digest.
    if [[ "$image" == benetnasch:* ]]; then
      continue
    fi

    if [[ "$image" =~ :latest(@|$) ]]; then
      echo "Third-party Compose image uses the latest tag: $file -> $image" >&2
      exit 1
    fi
    if [[ "$image" != *@sha256:* || ! "$image" =~ :[^/@]+@sha256:[[:xdigit:]]{64}$ ]]; then
      echo "Third-party Compose image must contain a version tag and digest: $file -> $image" >&2
      exit 1
    fi

    digest="${image##*@sha256:}"
    if [[ ! "$digest" =~ ^[[:xdigit:]]{64}$ ]]; then
      echo "Compose image has an invalid sha256 digest: $file -> $image" >&2
      exit 1
    fi
    checked=$((checked + 1))
  done < <(sed -nE 's/^[[:space:]]+image:[[:space:]]*([^[:space:]#]+).*$/\1/p' "$file")
done

if [[ "$checked" -eq 0 ]]; then
  echo 'No third-party Compose images were found.' >&2
  exit 1
fi

echo "Compose third-party image digests are pinned ($checked images checked)."
