#!/bin/bash

# ============================================
# task-mq-dev 红绿部署脚本 (Docker 部署)
# 无端口服务，仅做 Kafka 消息消费
# 红绿部署确保消息消费不中断
# ============================================

IMAGE="crpi-yw54l3gskjfve0c3.cn-hangzhou.personal.cr.aliyuncs.com/redbo-easy-chat/task-mq-dev:latest"
GREEN_CONTAINER="easy-chat-task-mq-green"
RED_CONTAINER="easy-chat-task-mq-red"

# ============================================
# 1. 拉取最新镜像
# ============================================
echo "=========================================="
echo "拉取最新镜像..."
echo "=========================================="
docker pull "$IMAGE"

if [ $? -ne 0 ]; then
    echo "[ERROR] 镜像拉取失败: $IMAGE"
    exit 1
fi

echo ""

# ============================================
# 2. 判断当前谁在运行，启动另一个颜色
# ============================================
if docker ps | grep -q "$GREEN_CONTAINER"; then
    echo "🟢 当前 GREEN 正在运行"
    echo "🔴 启动 RED 新版本..."
    echo "=========================================="
    
    # 启动 RED 容器（无端口映射）
    docker run -d \
        --name "$RED_CONTAINER" \
        --restart unless-stopped \
        --network host \
        "$IMAGE"
    
    if [ $? -ne 0 ]; then
        echo "[ERROR] RED 容器启动失败!"
        exit 1
    fi
    
    # 等待新消费者加入（Kafka rebalance 需要时间）
    echo "⏳ 等待 Kafka 消费者组重平衡 (15 秒)..."
    sleep 15
    
    # 平滑关闭 GREEN
    echo "🛑 平滑关闭 GREEN 容器..."
    docker stop --time 30 "$GREEN_CONTAINER"
    docker rm "$GREEN_CONTAINER"
    
    echo "✅ 红绿部署完成！当前 RED 对外服务"
    
elif docker ps | grep -q "$RED_CONTAINER"; then
    echo "🔴 当前 RED 正在运行"
    echo "🟢 启动 GREEN 新版本..."
    echo "=========================================="
    
    # 启动 GREEN 容器（无端口映射）
    docker run -d \
        --name "$GREEN_CONTAINER" \
        --restart unless-stopped \
        --network host \
        "$IMAGE"
    
    if [ $? -ne 0 ]; then
        echo "[ERROR] GREEN 容器启动失败!"
        exit 1
    fi
    
    # 等待新消费者加入（Kafka rebalance 需要时间）
    echo "⏳ 等待 Kafka 消费者组重平衡 (15 秒)..."
    sleep 15
    
    # 平滑关闭 RED
    echo "🛑 平滑关闭 RED 容器..."
    docker stop --time 30 "$RED_CONTAINER"
    docker rm "$RED_CONTAINER"
    
    echo "✅ 红绿部署完成！当前 GREEN 对外服务"
    
else
    echo "⚠️  没有运行中的容器，首次部署"
    echo "=========================================="
    
    # 首次部署，启动 GREEN
    docker run -d \
        --name "$GREEN_CONTAINER" \
        --restart unless-stopped \
        --network host \
        "$IMAGE"
    
    if [ $? -eq 0 ]; then
        echo "✅ 首次部署完成！GREEN 已启动"
    else
        echo "[ERROR] 容器启动失败!"
        exit 1
    fi
fi

echo ""
echo "=========================================="
echo "📊 查看状态: docker ps | grep task-mq"
echo "📝 查看日志: docker logs -f $([ \"$(docker ps -q --filter name=$GREEN_CONTAINER)\" != \"\" ] && echo $GREEN_CONTAINER || echo $RED_CONTAINER)"
echo "🛑 停止服务: docker stop $GREEN_CONTAINER $RED_CONTAINER"
echo "=========================================="
