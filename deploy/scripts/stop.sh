#!/bin/bash
cd "$(dirname "$0")/../.." || exit

echo "=== 关闭本项目所有 air/go 进程 ==="

pkill -f "air.*$(pwd)"
pkill -f "go.*$(pwd)"

echo "✅ 服务已全部关闭"