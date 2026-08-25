#!/usr/bin/env bash

set -euo pipefail

mapfile -t files < <(rg --files app/application/service -g '*.go' | grep -v '_test\.go$')

if rg -n '^func New[A-Za-z0-9_]*Service[^\n]*\.\.\.' "${files[@]}"; then
  echo 'application service constructors must use explicit dependencies, not variadic arguments' >&2
  exit 1
fi

echo 'service constructors use explicit dependency parameters'
