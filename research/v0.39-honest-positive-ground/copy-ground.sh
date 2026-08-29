#!/bin/sh
# Copyright (c) 2026 OpenWALDO Project contributors
# SPDX-License-Identifier: Apache-2.0

set -eu

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
cp "$script_dir/positive-ground.jsonl" positive-ground.jsonl
