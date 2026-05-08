#!/bin/bash
# Install compass-dash as a macOS background service (launchd).
# Starts automatically on login and restarts if it exits.
#
# Usage:
#   ./install-service.sh                   # uses ~/compass as the path
#   ./install-service.sh /path/to/compass  # custom compass path

set -e

BINARY="/usr/local/bin/compass-dash"
LABEL="io.hiipo.compass-dash"
PLIST_DIR="$HOME/Library/LaunchAgents"
PLIST="$PLIST_DIR/$LABEL.plist"
COMPASS_PATH="${1:-$HOME/compass}"
COMPASS_PATH="${COMPASS_PATH/#\~/$HOME}"  # expand ~ if passed literally

if [ ! -f "$BINARY" ]; then
  echo "Error: compass-dash not found at $BINARY"
  echo ""
  echo "Install it first:"
  echo "  sudo curl -L https://github.com/jeffgeiser/compass-dash/releases/latest/download/compass-dash-darwin-arm64 -o /usr/local/bin/compass-dash"
  echo "  sudo chmod +x /usr/local/bin/compass-dash"
  exit 1
fi

if [ ! -d "$COMPASS_PATH" ]; then
  echo "Error: compass path not found: $COMPASS_PATH"
  echo "Create it first or pass the correct path as an argument."
  exit 1
fi

mkdir -p "$PLIST_DIR"

cat > "$PLIST" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$LABEL</string>
    <key>ProgramArguments</key>
    <array>
        <string>$BINARY</string>
        <string>--compass-path</string>
        <string>$COMPASS_PATH</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/compass-dash.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/compass-dash.log</string>
</dict>
</plist>
EOF

# Stop existing instance if running
launchctl unload "$PLIST" 2>/dev/null || true

# Start the service
launchctl load "$PLIST"

echo ""
echo "compass-dash is running."
echo ""
echo "  Dashboard:  http://127.0.0.1:7174"
echo "  Compass:    $COMPASS_PATH"
echo "  Logs:       /tmp/compass-dash.log"
echo ""
echo "It will start automatically at login."
echo ""
echo "To stop:      launchctl unload $PLIST"
echo "To uninstall: bash uninstall-service.sh"
