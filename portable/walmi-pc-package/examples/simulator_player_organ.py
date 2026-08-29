#!/usr/bin/env python3
"""Contract fixture: one deterministic candidate organ for bridge tests."""

import hashlib
import json
import sys


request = json.load(sys.stdin)
actions = request.get("observation", {}).get("actions", [])
if not actions:
    raise SystemExit("visible action menu is empty")
selector = int(hashlib.sha256(f"{request.get('episodeId')}|{request.get('turn')}|fixture-organ".encode()).hexdigest()[:16], 16)
json.dump(
    {
        "schema": "axm.walmi.simulator-player-response/v1",
        "actionId": actions[selector % len(actions)]["id"],
    },
    sys.stdout,
    separators=(",", ":"),
)
sys.stdout.write("\n")
