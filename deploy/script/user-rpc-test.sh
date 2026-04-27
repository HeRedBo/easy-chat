#!/bin/bash
reso_addr='crpi-yw54l3gskjfve0c3.cn-hangzhou.personal.cr.aliyuncs.com/redbo-easy-chat/user-rpc-dev'
tag='latest'

# ==========================================
# 部署配置路径（可选）
# ==========================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_CONF="$SCRIPT_DIR/deploy.conf"

# ==========================================
# 获取服务端口（标签 + 配置文件双重保障）
# ==========================================
echo "==== 获取服务端口配置 ===="
SERVICE_PORT=$("$SCRIPT_DIR/get_deploy_port.sh" "$reso_addr" "$tag" "$DEPLOY_CONF" | tail -n 1)

if [ -z "$SERVICE_PORT" ]; then
    echo "❌ 无法获取端口配置"
    exit 1
fi

echo "✅ 服务内部端口: $SERVICE_PORT"

# ==========================================
# 固定两个容器名字：绿色 + 红色
# ==========================================
GREEN_CONTAINER="easy-chat-user-rpc-green"
RED_CONTAINER="easy-chat-user-rpc-red"

# 基于内部端口计算外部端口（避免冲突）
GREEN_PORT=$((SERVICE_PORT + 2000))
RED_PORT=$((GREEN_PORT + 1))

echo "🟢 绿色容器外部端口: $GREEN_PORT"
echo "🔴 红色容器外部端口: $RED_PORT"
echo "========================================="

# ==========================================
# 1. 拉最新镜像
# ==========================================
docker pull ${reso_addr}:${tag}

# ==========================================
# 2. 判断当前谁在运行，启动另一个颜色
# ==========================================
if docker ps | grep -q "${GREEN_CONTAINER}"; then
    echo "==== 当前是 GREEN 对外提供服务 ===="
    echo "==== 启动 RED 新版本 ===="
    
    docker run -d \
        -p ${RED_PORT}:${SERVICE_PORT} \
        --name ${RED_CONTAINER} \
        -e PORT=${SERVICE_PORT} \
        --add-host=host.docker.internal:host-gateway \
        ${reso_addr}:${tag}

    # 等待启动 + 健康检查（关键！）
    sleep 10

    # 平滑关闭旧的 GREEN
    echo "==== 平滑关闭 GREEN ===="
    docker stop --time 30 ${GREEN_CONTAINER}
    docker rm ${GREEN_CONTAINER}

else
    echo "==== 当前是 RED 对外提供服务 ===="
    echo "==== 启动 GREEN 新版本 ===="
    
    docker run -d \
        -p ${GREEN_PORT}:${SERVICE_PORT} \
        --name ${GREEN_CONTAINER} \
        -e PORT=${SERVICE_PORT} \
        --add-host=host.docker.internal:host-gateway \
        ${reso_addr}:${tag}

    sleep 10

    echo "==== 平滑关闭 RED ===="
    docker stop --time 30 ${RED_CONTAINER}
    docker rm ${RED_CONTAINER}
fi

echo "==== 红绿部署完成 ===="
echo "✅ 服务端口: $SERVICE_PORT"
echo "✅ 访问端口: $([ -z "$(docker ps | grep $GREEN_CONTAINER)" ] && echo $RED_PORT || echo $GREEN_PORT)"
