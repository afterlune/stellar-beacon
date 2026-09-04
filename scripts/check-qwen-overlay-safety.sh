#!/usr/bin/env bash

set -euo pipefail

file="docker-compose.integration.qwen.yaml"
base_file="docker-compose.integration.yaml"

if [[ ! -f "$file" ]]; then
  echo "Qwen Compose overlay is missing: $file" >&2
  exit 1
fi
if [[ ! -f "$base_file" ]]; then
  echo "Integration Compose file is missing: $base_file" >&2
  exit 1
fi

for required in \
  "profiles:" \
  "- qwen-low-memory" \
  "restart: \"no\"" \
  "mem_limit: 4g" \
  "memswap_limit: 4g" \
  "cpus: \"2.0\"" \
  "pids_limit: 256" \
  "--mem-fraction-static" \
  "\"0.25\"" \
  "--context-length" \
  "\"4096\"" \
  "--max-total-tokens" \
  "\"2048\"" \
  "--chunked-prefill-size" \
  "\"512\"" \
  "--max-prefill-tokens" \
  "\"2048\"" \
  "--max-running-requests" \
  "\"1\"" \
  "--disable-radix-cache" \
  "read_only: true"; do
  if ! rg -q --fixed-strings -- "$required" "$file"; then
    echo "Qwen overlay is missing memory-safety invariant: $required" >&2
    exit 1
  fi
done

if rg -n -- "restart:[[:space:]]+(always|unless-stopped|on-failure)|memswap_limit:[[:space:]]+(-1|[0-3]g)" "$file"; then
  echo "Qwen overlay must not allow automatic restart or an unbounded/smaller-than-container swap budget." >&2
  exit 1
fi

if rg -n -- "qwen-embedding|QWEN_MODEL_PATH" "$base_file"; then
  echo "Qwen service and model path must remain outside the normal integration Compose file." >&2
  exit 1
fi

if ! rg -q --fixed-strings "check-qwen-memory.ps1') -RequireRunningContainer" "scripts/integration-provider-smoke.ps1"; then
  echo "Local Qwen Provider smoke must verify the actual running container memory cap." >&2
  exit 1
fi
if ! rg -q --fixed-strings 'if settings.LocalEmbeddingEnabled' "app/infra/ai/router.go" ||
  ! rg -q --fixed-strings 'maxConcurrent = 1' "app/infra/ai/router.go"; then
  echo "Local embedding routing must serialize provider requests under the low-memory budget." >&2
  exit 1
fi
if ! rg -q --fixed-strings "docker stats --no-stream --format" "scripts/check-qwen-memory.ps1" ||
  ! rg -q --fixed-strings 'MinimumDockerFreeGiB = 12' "scripts/check-qwen-memory.ps1" ||
  ! rg -q --fixed-strings "nvidia-smi --query-gpu=index,memory.free" "scripts/check-qwen-memory.ps1" ||
  ! rg -q --fixed-strings 'MinimumFreeGPUMiB = 5120' "scripts/check-qwen-memory.ps1"; then
  echo "Qwen memory preflight must reserve Docker capacity and GPU memory before starting." >&2
  exit 1
fi

launcher="scripts/integration-qwen-up.ps1"
if [[ ! -f "$launcher" ]]; then
  echo "Qwen launcher is missing: $launcher" >&2
  exit 1
fi
for required in \
  "check-qwen-memory.ps1" \
  "-MinimumFreeGPUMiB" \
  "-AllowContainerChanges" \
  "-IncludeBackend" \
  "config', '--quiet" \
  "up', '-d', '--no-deps', '--force-recreate" \
  "qwen-embedding"; do
  if ! rg -q --fixed-strings -- "$required" "$launcher"; then
    echo "Qwen launcher is missing safety invariant: $required" >&2
    exit 1
  fi
done
if rg -n -- "up.*backend.*postgresql|up.*backend.*redis|up.*backend.*meilisearch|up.*backend.*minio" "$launcher"; then
  echo "Qwen launcher must not start infrastructure dependencies." >&2
  exit 1
fi

echo "Qwen overlay keeps an explicit low-memory profile, bounded runtime resources, and a guarded launcher."
