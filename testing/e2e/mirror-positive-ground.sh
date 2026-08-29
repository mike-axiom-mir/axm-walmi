#!/bin/sh
# Copyright (c) 2026 OpenWALDO Project contributors
# SPDX-License-Identifier: Apache-2.0

set -eu

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH='' cd -- "$script_dir/../.." && pwd)
temporary_base=${TMPDIR:-/tmp}
work=$(mktemp -d "$temporary_base/waldo-positive-ground-e2e.XXXXXX")

cleanup() {
  case "$work" in
    "$temporary_base"/waldo-positive-ground-e2e.*) rm -rf -- "$work" ;;
    *) echo "refusing to remove unexpected workspace: $work" >&2 ;;
  esac
}
trap cleanup EXIT HUP INT TERM

binary="$work/waldo"
projection="$work/training.jsonl"
seed="$repo_root/research/v0.39-honest-positive-ground/positive-ground.jsonl"

echo "testing: visible synthetic positive-ground verification and projection"
(cd "$repo_root" && GOCACHE="$work/go-cache" go build -o "$binary" ./cmd/waldo)

receipt=$(
  "$binary" --json mirror ground verify "$seed" \
    --positive-seed \
    --training-to "$projection"
)
printf '%s\n' "$receipt" | grep -Eq '"records"[[:space:]]*:[[:space:]]*32'
printf '%s\n' "$receipt" | grep -Eq '"syntheticRecords"[[:space:]]*:[[:space:]]*32'
printf '%s\n' "$receipt" | grep -Eq '"observedRecords"[[:space:]]*:[[:space:]]*0'
printf '%s\n' "$receipt" | grep -Eq '"positiveSeedValidated"[[:space:]]*:[[:space:]]*true'
printf '%s\n' "$receipt" | grep -Eq '"trainingProjectionMutation"[[:space:]]*:[[:space:]]*true'

[ "$(wc -l < "$projection" | tr -d ' ')" -eq 32 ]
[ "$(stat -c '%a' "$projection")" = 600 ]
grep -q '"data_class":"SYNTHETIC_SEED"' "$projection"
grep -q '"messages":\[{"role":"user"' "$projection"
grep -q '"source_record_sha256":"' "$projection"

if "$binary" mirror ground verify "$seed" --training-to "$projection" >/dev/null 2>&1; then
  echo "Mirror ground projection unexpectedly replaced an existing file" >&2
  exit 1
fi

echo "E2E positive ground passed: 32 sealed synthetic targets verified and projected without claiming observation or training"
