#!/bin/bash

# Kafka Topic 检测与创建脚本
# 用途：检查 Topic 是否存在，不存在则自动创建

KAFKA_BOOTSTRAP="127.0.0.1:9092"
TOPICS=("msgChatTransfer" "msgReadTransfer")
PARTITIONS=8
REPLICATION_FACTOR=1

echo "========================================="
echo "Kafka Topic 检测与创建脚本"
echo "========================================="
echo ""

# 检查 Kafka 是否运行
echo "📡 检查 Kafka 连接..."
docker exec -it kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP --list > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "❌ 无法连接到 Kafka: $KAFKA_BOOTSTRAP"
    echo "   请确保 Kafka 服务已启动"
    exit 1
fi
echo "✅ Kafka 连接正常"
echo ""

# 检查 auto.create.topics.enable 配置
echo "⚙️  检查 Kafka 自动创建配置..."
echo "   提示：如果 Topic 不存在，Kafka 可能会自动创建"
echo ""

# 检查并创建 Topic
for TOPIC in "${TOPICS[@]}"; do
    echo "-----------------------------------------"
    echo "📋 检查 Topic: $TOPIC"
    
    # 检查 Topic 是否存在
    EXISTS=$(docker exec -it kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server $KAFKA_BOOTSTRAP --list | grep "^${TOPIC}$")
    
    if [ -z "$EXISTS" ]; then
        echo "⚠️  Topic 不存在，正在创建..."
        
        # 创建 Topic
        docker exec -it kafka /opt/kafka/bin/kafka-topics.sh --create \
            --bootstrap-server $KAFKA_BOOTSTRAP \
            --topic $TOPIC \
            --partitions $PARTITIONS \
            --replication-factor $REPLICATION_FACTOR
        
        if [ $? -eq 0 ]; then
            echo "✅ Topic 创建成功: $TOPIC (partitions=$PARTITIONS)"
        else
            echo "❌ Topic 创建失败: $TOPIC"
            echo "   可能原因："
            echo "   1. Kafka 配置 auto.create.topics.enable=false"
            echo "   2. 权限不足"
            echo "   3. 参数配置错误"
        fi
    else
        echo "✅ Topic 已存在: $TOPIC"
        
        # 显示 Topic 详情
        echo "📊 Topic 详情："
        docker exec -it kafka /opt/kafka/bin/kafka-topics.sh --describe \
            --bootstrap-server $KAFKA_BOOTSTRAP \
            --topic $TOPIC | grep -E "PartitionCount|Topic:"
    fi
    echo ""
done

echo "========================================="
echo "✅ 检测完成"
echo "========================================="
echo ""
echo "💡 提示："
echo "   - 如果 Topic 已存在且 partitions < $PARTITIONS"
echo "   - 可以使用以下命令扩容："
echo "     docker exec -it kafka /opt/kafka/bin/kafka-topics.sh --alter --bootstrap-server $KAFKA_BOOTSTRAP --topic <topic-name> --partitions $PARTITIONS"
echo ""
echo "📝 验证命令："
echo "   docker exec -it kafka /opt/kafka/bin/kafka-topics.sh --describe --bootstrap-server $KAFKA_BOOTSTRAP --topic msgChatTransfer"
echo ""
