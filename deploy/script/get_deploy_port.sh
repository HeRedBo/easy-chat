#!/bin/bash
# ==========================================
# 端口获取工具：标签 + 配置文件双重保障
# ==========================================
# 使用方式：
#   ./get_deploy_port.sh <镜像名> <标签> [部署配置文件]
# 示例：
#   ./get_deploy_port.sh user-api latest ./deploy.conf
# ==========================================

IMAGE_NAME=$1
IMAGE_TAG=$2
DEPLOY_CONF=${3:-""}

# ==========================================
# 1. 优先从镜像标签读取端口
# ==========================================
echo "📦 检查镜像标签..."
LABEL_PORT=$(docker inspect --format='{{index .Config.Labels "service.port"}}' ${IMAGE_NAME}:${IMAGE_TAG} 2>/dev/null)

if [ -n "$LABEL_PORT" ] && [ "$LABEL_PORT" != "<nil>" ]; then
    echo "✅ 从镜像标签获取端口: $LABEL_PORT"
    FINAL_PORT=$LABEL_PORT
else
    echo "⚠️  镜像标签中未找到端口信息"
    LABEL_PORT=""
fi

# ==========================================
# 2. 检查部署配置文件是否有覆盖设置
# ==========================================
SERVICE_NAME=$(docker inspect --format='{{index .Config.Labels "service.name"}}' ${IMAGE_NAME}:${IMAGE_TAG} 2>/dev/null)
SERVICE_TYPE=$(docker inspect --format='{{index .Config.Labels "service.type"}}' ${IMAGE_NAME}:${IMAGE_TAG} 2>/dev/null)

if [ -n "$DEPLOY_CONF" ] && [ -f "$DEPLOY_CONF" ]; then
    echo "📄 读取部署配置文件: $DEPLOY_CONF"
    
    # 从配置文件读取端口覆盖
    if [ -n "$SERVICE_NAME" ]; then
        CONF_PORT=$(grep -A 5 "\[${SERVICE_NAME}-${SERVICE_TYPE}\]" "$DEPLOY_CONF" 2>/dev/null | grep "external_port=" | cut -d'=' -f2 | tr -d ' ')
        
        if [ -n "$CONF_PORT" ]; then
            echo "✅ 从配置文件获取覆盖端口: $CONF_PORT"
            FINAL_PORT=$CONF_PORT
        fi
    fi
fi

# ==========================================
# 3. 输出最终端口
# ==========================================
if [ -z "$FINAL_PORT" ]; then
    echo "❌ 无法获取端口信息"
    echo "   - 镜像标签: ${LABEL_PORT:-未设置}"
    echo "   - 配置文件: ${DEPLOY_CONF:-未指定}"
    exit 1
fi

echo "🎯 最终使用端口: $FINAL_PORT"
echo "$FINAL_PORT"
