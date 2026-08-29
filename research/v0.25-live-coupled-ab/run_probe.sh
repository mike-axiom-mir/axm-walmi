#!/bin/sh
set -eu

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
cd "$script_dir"
python3 probe.py --output artifacts --repeat 2
python3 -m unittest -v test_probe.py

