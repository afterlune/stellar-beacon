#!/usr/bin/env bash
set -euo pipefail

if rg -n --glob '*.go' 'golang\.org/x/crypto/openpgp' internal cmd; then
  echo "OpenPGP imports are forbidden; use a maintained implementation only when OpenPGP is explicitly required" >&2
  exit 1
fi

echo "No OpenPGP imports found in Go source."
