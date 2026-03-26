#!/bin/bash

# 获取项目根目录（脚本所在目录的上一级）
ROOT=$(cd "$(dirname "$0")/.." && pwd)
TARGET_DIR="$ROOT/apps/task/mq"

# 打开新终端并执行 air
osascript <<EOF
tell application "Terminal"
    do script "cd '$TARGET_DIR' && air"
    activate
end tell
EOF