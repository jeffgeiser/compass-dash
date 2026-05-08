#!/bin/bash
# Remove the compass-dash background service.

LABEL="io.hiipo.compass-dash"
PLIST="$HOME/Library/LaunchAgents/$LABEL.plist"

if [ ! -f "$PLIST" ]; then
  echo "Service not installed (no plist found at $PLIST)."
  exit 0
fi

launchctl unload "$PLIST" 2>/dev/null || true
rm -f "$PLIST"

echo "compass-dash service removed."
