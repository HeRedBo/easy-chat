#!/bin/bash
# 进入项目根目录
cd "$(dirname "$0")/../.." || exit

echo "=== 启动服务 ==="

startModule() {
  osascript -e "tell application \"Terminal\" to do script \"cd '$(pwd)/apps/$1/rpc' && air\""
  osascript -e "tell application \"Terminal\" to do script \"cd '$(pwd)/apps/$1/api' && air\""
}

startTask() {
  osascript -e "tell application \"Terminal\" to do script \"cd '$(pwd)/apps/task/mq' && air\""
}

if [ $# -eq 0 ]; then
  startModule user
  startModule im
  startModule social
  startTask
else
  for arg in "$@"; do
    if [ "$arg" = "task" ]; then
      startTask
    else
      startModule "$arg"
    fi
  done
fi

echo "✅ 启动完成"