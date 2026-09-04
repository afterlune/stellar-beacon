#!/usr/bin/env bash

set -euo pipefail

file="cmd/article_index_backfill.go"

if [[ ! -f "$file" ]]; then
  echo "Article index backfill command is missing: $file" >&2
  exit 1
fi

if ! rg -q --fixed-strings 'PersistentFlags().Bool("allow-writes"' "$file"; then
  echo "Article index backfill commands must expose an explicit --allow-writes gate." >&2
  exit 1
fi

for handler in \
  runArticleIndexBackfillStart \
  runArticleIndexBackfill \
  requestArticleIndexBackfillPause \
  resumeArticleIndexBackfill; do
  start=$(rg -n -m 1 "^func ${handler}\(" "$file" | cut -d: -f1 || true)
  if [[ -z "$start" ]]; then
    echo "Article index backfill handler is missing: $handler" >&2
    exit 1
  fi
  body=$(sed -n "${start},$((start + 28))p" "$file")
  if ! grep -q 'requireArticleIndexBackfillWriteAuthorization' <<<"$body"; then
    echo "Article index backfill handler is missing its write gate: $handler" >&2
    exit 1
  fi
done

swap_start=$(rg -n -m 1 '^func runArticleIndexSwap\(' "$file" | cut -d: -f1 || true)
if [[ -z "$swap_start" ]]; then
  echo "Article index swap handler is missing: runArticleIndexSwap" >&2
  exit 1
fi
swap_body=$(sed -n "${swap_start},$((swap_start + 70))p" "$file")
if ! grep -q 'requireArticleIndexWriteAuthorization' <<<"$swap_body"; then
  echo "Article index swap handler is missing its write gate." >&2
  exit 1
fi

if ! rg -q --fixed-strings 'func showArticleIndexBackfillStatus' "$file" || \
   ! rg -q --fixed-strings 'func requireArticleIndexBackfillWriteAuthorization' "$file"; then
  echo "Article index backfill status or authorization contract is missing." >&2
  exit 1
fi

plan_start=$(rg -n -m 1 '^func runArticleIndexBackfillPlan\(' "$file" | cut -d: -f1 || true)
if [[ -z "$plan_start" ]]; then
  echo "Read-only article index plan handler is missing." >&2
  exit 1
fi
plan_body=$(sed -n "${plan_start},$((plan_start + 45))p" "$file")
if grep -q 'requireArticleIndexBackfillWriteAuthorization' <<<"$plan_body"; then
  echo "Read-only article index plan must not use the backfill write authorization path." >&2
  exit 1
fi

if ! rg -q --fixed-strings 'articleIndexBackfillPlanCmd' "$file" || \
   ! rg -q --fixed-strings 'article-index backfill plan is read-only' "$file"; then
  echo "Read-only article index plan command is missing its explicit no-write guard." >&2
  exit 1
fi

projector="app/infra/task/article_index_projector.go"
if [[ ! -f "$projector" ]]; then
  echo "Article index projector is missing: $projector" >&2
  exit 1
fi
for required in \
  'DefaultArticleIndexMaxArticleContentBytes' \
  'DefaultArticleIndexMaxChunksPerArticle' \
  'DefaultArticleIndexMaxEmbeddingBatchSize' \
  'DefaultArticleIndexMaxEmbeddingBatchBytes' \
  'article content exceeds the projector memory budget' \
  'article chunk count exceeds the projector memory budget' \
  'embedding batch input exceeds the projector memory budget'; do
  if ! rg -q --fixed-strings "$required" "$projector"; then
    echo "Article index projector memory budget guard is missing: $required" >&2
    exit 1
  fi
done

echo "Article index backfill write operations have explicit --allow-writes gates; status remains read-only and projector memory budgets are present."
