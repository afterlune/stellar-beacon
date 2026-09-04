#!/usr/bin/env bash

set -euo pipefail

expected_go='1.27'

module_go=$(sed -nE 's/^go ([0-9]+\.[0-9]+)(\.[0-9]+)?$/\1/p' go.mod | head -n 1)
if [[ "$module_go" != "$expected_go" ]]; then
  echo "go.mod declares Go ${module_go:-<missing>}, expected ${expected_go}" >&2
  exit 1
fi

if ! grep -q "FROM golang:${expected_go}.0-alpine" Dockerfile; then
  echo "Dockerfile builder image is not pinned to golang:${expected_go}.0-alpine" >&2
  exit 1
fi

if ! grep -q "GOMAXPROCS=1" Dockerfile || ! grep -q "go build -p 1" Dockerfile; then
  echo "Dockerfile Go build must keep CPU/package parallelism bounded" >&2
  exit 1
fi

if ! grep -q "go-version: '${expected_go}.x'" .github/workflows/ci.yml; then
  echo "CI does not configure Go ${expected_go}.x" >&2
  exit 1
fi

if ! grep -q "CGO_ENABLED: '1'" .github/workflows/ci.yml; then
  echo "CI race job must explicitly enable CGO" >&2
  exit 1
fi

if ! grep -q "go test -race -count=1 ./\.\.\." .github/workflows/ci.yml; then
  echo "CI race job must run an uncached full-repository race test" >&2
  exit 1
fi

echo "Go toolchain is consistently pinned to ${expected_go}."
