#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BIN_DIR="${HOME}/.local/bin"
BIN_PATH="$BIN_DIR/axm-workshop"
SERVICE_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
SERVICE_PATH="$SERVICE_DIR/axm-workshop.service"

mkdir -p "$BIN_DIR" "$SERVICE_DIR"

echo "Building AXM Local Workshop..."
(
  cd "$REPO_ROOT"
  go build -o "$BIN_PATH" ./cmd/workshop
)
chmod 700 "$BIN_PATH"

cat > "$SERVICE_PATH" <<'EOF'
[Unit]
Description=AXM Local Workshop
After=default.target

[Service]
Type=simple
ExecStart=%h/.local/bin/axm-workshop
Restart=on-failure
RestartSec=5
Environment=AXM_WORKSHOP_NATIVE_WALDO=1

[Install]
WantedBy=default.target
EOF

systemctl --user daemon-reload
systemctl --user enable --now axm-workshop.service

echo "AXM Local Workshop installed and started."
echo "Open: http://127.0.0.1:7788"
echo "Service: axm-workshop.service"
echo "Binary: $BIN_PATH"
echo "Note: this is a user service and normally starts when this user logs in."
