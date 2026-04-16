#!/bin/bash
reso_addr='crpi-yw54l3gskjfve0c3.cn-hangzhou.personal.cr.aliyuncs.com/redbo-easy-chat/user-api-dev'
tag='latest'

# ==========================================
# 固定两个容器名字：绿色 + 红色
# ==========================================
GREEN_CONTAINER="easy-chat-user-api-green"
RED_CONTAINER="easy-chat-user-api-red"


# 宿主机端口
GREEN_PORT=8080
RED_PORT=8081


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
        -p ${RED_PORT}:8090 \
        --name ${RED_CONTAINER} \
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
        -p ${GREEN_PORT}:8090 \
        --name ${GREEN_CONTAINER} \
        --add-host=host.docker.internal:host-gateway \
        ${reso_addr}:${tag}

    sleep 10

    echo "==== 平滑关闭 RED ===="
    docker stop --time 30 ${RED_CONTAINER}
    docker rm ${RED_CONTAINER}
fi

echo "==== 红绿部署完成 ===="