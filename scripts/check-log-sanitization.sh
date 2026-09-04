#!/usr/bin/env bash
set -euo pipefail

mapfile -t files < <(rg --files -g '*.go' -g '!**/*_test.go' -g '!vendor/**' -g '!**/.codebuddy/**' | sort)
if ((${#files[@]} == 0)); then
  echo 'no production Go files found'
  exit 0
fi

# This guard catches the common direct-attribute regressions. Structured
# logging code must pass a stable code (for example SafeCode or
# safeWorkerError), never an error variable or panic value. The source review
# rule remains broader than this line-oriented check.
pattern='slog\.(Debug|Info|Warn|Error|Log|LogAttrs)(Context)?\([^\r\n]*"error"[[:space:]]*,[[:space:]]*(err|[A-Za-z_][A-Za-z0-9_]*Err|recovered)([[:space:]]*[,)]|[[:space:]]*$)'
matches=$(rg -n --pcre2 "$pattern" "${files[@]}" || true)
if [[ -n "$matches" ]]; then
  printf '%s\n' "$matches" >&2
  echo 'raw error values must not be passed to slog; use a stable error code' >&2
  exit 1
fi

if rg -n --pcre2 'slog\.(Any|String)\("error"[[:space:]]*,[[:space:]]*(err|[A-Za-z_][A-Za-z0-9_]*Err|recovered)([[:space:]]*[,)]|[[:space:]]*$)' "${files[@]}"; then
  echo 'raw error values must not be passed to slog.Any/slog.String' >&2
  exit 1
fi

echo 'slog error attributes use stable, detail-free values'
