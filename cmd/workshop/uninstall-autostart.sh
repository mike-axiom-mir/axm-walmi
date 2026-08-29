#!/usr/bin/env bash
set -euo pipefail

SERVICE_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
SERVICE_PATH="$SERVICE_DIR/axm-workshop.service"
BIN_PATH="${HOME}/.local/bin/axm-workshop"

systemctl --user disable --now axm-workshop.service 2>/dev/null || true
rm -f "$SERVICE_PATH"
systemctl --user daemon-reload
rm -f "$BIN_PATH"

echo "AXM Local Workshop auto-start removed."
echo "Workshop data was NOT deleted."
