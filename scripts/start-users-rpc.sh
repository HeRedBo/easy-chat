#!/bin/bash

ROOT=$(cd "$(dirname "$0")/.." && pwd)
TARGET_DIR="$ROOT/apps/user/rpc"

osascript <<EOF
tell application "Terminal"
    do script "cd '$TARGET_DIR' && air"
    activate
end tell
EOF