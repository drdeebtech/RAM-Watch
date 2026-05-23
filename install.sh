#!/bin/bash
set -e
export PATH="/opt/homebrew/bin:$PATH"
echo "Building ram-watch..."
go build -o ram-watch-bin .
sudo mv ram-watch-bin /opt/homebrew/bin/ram-watch
echo "Installed to /opt/homebrew/bin/ram-watch"
echo "Run: ram-watch"
