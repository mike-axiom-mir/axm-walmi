#!/bin/sh
set -eu

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH='' cd -- "$script_dir/../.." && pwd)
temporary_base=${RUNNER_TEMP:-${TMPDIR:-/tmp}}
e2e_log="$temporary_base/waldo-v036-pytorch-e2e.log"
output="$temporary_base/waldo-v036-development"

test ! -e "$output" || {
  echo "refusing to overwrite existing A/B output: $output" >&2
  exit 1
}

export WALDO_E2E_KEEP=1
if ! "$repo_root/testing/e2e/model-pytorch.sh" >"$e2e_log" 2>&1; then
  cat "$e2e_log"
  exit 1
fi
cat "$e2e_log"

work=$(sed -n 's/^preserved PyTorch E2E workspace: //p' "$e2e_log" | tail -1)
test -n "$work" && test -d "$work" || {
  echo "could not resolve preserved PyTorch E2E workspace" >&2
  exit 1
}

binary="$work/waldo"
config="$work/config.json"
test -x "$binary" || { echo "trained WALDO binary missing: $binary" >&2; exit 1; }
test -f "$config" || { echo "trained WALDO config missing: $config" >&2; exit 1; }

export WALDO_CONFIG="$config"
python3 "$script_dir/run_ab.py" \
  --waldo "$binary" \
  --model pytorch-smoke \
  --output "$output" \
  --max-tokens 8 \
  --temperature 0 \
  --top-p 1 \
  --seed 36001 \
  --timeout-seconds 300

test -f "$output/PUBLIC-SUMMARY.json"
test -f "$output/preheldout-freeze.json"
echo "v0.36 development output: $output"
