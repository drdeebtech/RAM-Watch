#!/bin/bash
cp "$(dirname "$0")/ram-watch.sh" /opt/homebrew/bin/ram-watch
chmod +x /opt/homebrew/bin/ram-watch
echo "Installed. Run: ram-watch"
