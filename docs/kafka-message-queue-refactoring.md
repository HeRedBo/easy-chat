# Kafka 消息队列有序性与可靠性改造方案

> 📅 改造时间：2026-05-01  
> 🎯 改造目标：提升消息处理并发能力，保证会话内消息有序性，优化消息可靠性保障

---

## 📋 目录

- [一、改造背景与问题分析](#一改造背景与问题分析)
- [二、Kafka 核心原理回顾](#二kafka-核心原理回顾)
- [三、改造方案设计](#三改造方案设计)
- [四、代码改造实施](#四代码改造实施)
- [五、配置调整](#五配置调整)
- [六、部署与运维](#六部署与运维)
- [七、监控与验证](#七监控与验证)
- [八、常见问题 FAQ](#八常见问题-faq)
- [九、参考资料](#九参考资料)

---

## 一、改造背景与问题分析

### 1.1 现状分析

**改造前的配置：**
```yaml
# task.yaml
MsgChatTransfer:
  Consumers: 1  # 单消费者
  
MsgReadTransfer:
  Consumers: 1  # 单消费者
```

**改造前的代码：**
```go
// mqclient/msgtransfer.go
func (c *msgChatTransferClient) Push(msg *mq.MsgChatTransfer) error {
    body, _ := json.Marshal(msg)
    return c.pusher.Push(context.Background(), string(body))  // ❌ 无 key
}
```

### 1.2 存在的问题

| 问题 | 影响 | 严重程度 |
|------|------|----------|
| **单消费者处理** | 无法水平扩展，吞吐量受限 | 🔴 高 |
| **无 key 路由** | 消息随机分配 partition，多消费者时乱序 | 🔴 高 |
| **错误处理不当** | 推送失败导致消息无限重试 | 🟡 中 |
| **缺少有序性保障** | 同一会话消息可能乱序处理 | 🔴 高 |

### 1.3 改造目标

```
✅ 提升并发处理能力（Consumers: 1 → 4）
✅ 保证会话内消息有序（PushWithKey + ConversationId）
✅ 优化错误处理机制（区分致命/非致命错误）
✅ 防止消息无限重试（合理返回 nil/error）
```

---

## 二、Kafka 核心原理回顾

### 2.1 Kafka 有序性保证

```
核心规则：
✅ 同一个 partition 内的消息严格有序
✅ 相同 key 的消息总是路由到同一个 partition
❌ 不同 partition 之间无法保证顺序
❌ 多个消费者并发消费同一 partition 会乱序
```

### 2.2 消息路由机制

```go
// 方式1：无 key（轮询分配）
pusher.Push(ctx, message)
// 结果：消息随机分配到 partition 0, 1, 2, 3...
// 风险：同一会话消息可能分散到多个 partition

// 方式2：有 key（hash 路由）
pusher.PushWithKey(ctx, key, message)
// 路由算法：hash(key) % partition_count = partition_id
// 保证：相同 key 的消息永远到同一个 partition
```

### 2.3 Consumer 工作机制

```
启动流程：
1. kq.MustNewQueue(config, handler)
   ↓
2. 创建 Consumers 个数的消费者协程
   ↓
3. 每个消费者订阅 Topic 的某些 partition
   ↓
4. Kafka 自动分配 partition 给消费者（Rebalance）

分区分配规则：
- partition 数量 >= 消费者数量
- 每个 partition 只分配给一个消费者
- 一个消费者可以消费多个 partition
```

### 2.4 消息确认机制（ACK）

```go
func (h *Handler) Consume(ctx context.Context, key, value string) error {
    // 处理消息...
    
    return nil  // ✅ 成功 → 提交 offset（消息处理完成）
    return err  // ❌ 失败 → 不提交 offset（消息会重新消费）
}
```

**消息不丢失的三层保障：**

```
┌─────────────────────────────────────────────────┐
│          Kafka 消息可靠性保障三层机制              │
├─────────────────────────────────────────────────┤
│                                                  │
│  第一层：生产者保障（Producer）                    │
│  ├─ RequiredAcks: all (等待所有副本确认)           │
│  ├─ Retry: 失败重试                               │
│  └─ Push 返回 error 时需处理                      │
│                                                  │
│  第二层：Broker 保障（Kafka）                     │
│  ├─ replication.factor >= 2 (多副本)              │
│  ├─ min.insync.replicas >= 2 (最小同步副本)       │
│  └─ unclean.leader.election.enable=false          │
│                                                  │
│  第三层：消费者保障（Consumer）                    │
│  ├─ Consume 返回 nil → 提交 offset                │
│  ├─ Consume 返回 error → 不提交，重新消费          │
│  └─ commitInterval: 1s（默认）                    │
│                                                  │
└─────────────────────────────────────────────────┘
```

---

## 三、改造方案设计

### 3.1 方案对比

| 方案 | 优点 | 缺点 | 适用场景 |
|------|------|------|----------|
| **单消费者** | 简单，全局有序 | 性能差，无法扩展 | 低并发系统 |
| **按 Key 分区** ✅ | 会话内有序，可并行 | 需正确设计 key | IM 聊天系统 |
| **业务层排序** | 灵活 | 复杂度高 | 特殊业务需求 |

### 3.2 推荐方案：按 ConversationId 分区

**为什么选择 ConversationId 作为 key？**

```
私聊: ConversationId = "user1_user2" 
      → 两人之间所有消息同 key
      → 路由到同一 partition
      → 保证有序 ✓

群聊: ConversationId = "group123"
      → 群内所有消息同 key
      → 路由到同一 partition
      → 保证有序 ✓

效果:
- 同一会话的消息 → 同一 partition → 同一消费者 → 有序处理
- 不同会话的消息 → 不同 partition → 不同消费者 → 并行处理
```

### 3.3 错误处理策略

```go
错误分类：
├─ 致命错误（返回 error，触发重试）
│  ├─ 数据库写入失败
│  └─ 消息格式无法解析（特殊处理）
│
└─ 非致命错误（返回 nil，不重试）
   ├─ WebSocket 推送失败（用户不在线）
   └─ 消息已持久化，推送失败不影响
```

---

## 四、代码改造实施

### 4.1 生产者改造 - 使用 PushWithKey

#### 📝 文件：`apps/task/mq/mqclient/msgtransfer.go`

**改造前：**
```go
func (c *msgChatTransferClient) Push(msg *mq.MsgChatTransfer) error {
    body, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    
    return c.pusher.Push(context.Background(), string(body))  // ❌ 无 key
}
```

**改造后：**
```go
func (c *msgChatTransferClient) Push(msg *mq.MsgChatTransfer) error {
    body, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    // ✅ 使用 ConversationId 作为 key，保证同一会话消息路由到同一 partition
    // 这样同一会话的消息由同一个消费者处理，保证会话内消息有序
    // 即使 Consumers > 1，同一会话的消息也不会乱序
    return c.pusher.PushWithKey(context.Background(), msg.ConversationId, string(body))
}
```

#### 📝 文件：`apps/task/mq/mqclient/msgmarkread.go`

**改造后：**
```go
func (c *msgReadChatTransferClient) Push(msg *mq.MsgMarkRead) error {
    body, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    // ✅ 使用 ConversationId 作为 key，保证同一会话的已读回执路由到同一 partition
    // 避免同一会话的已读消息乱序处理
    return c.pusher.PushWithKey(context.Background(), msg.ConversationId, string(body))
}
```

### 4.2 消费者改造 - 优化错误处理

#### 📝 文件：`apps/task/mq/internal/handler/msgTransfer/msgChatTransfer.go`

**改造前：**
```go
func (m *MsgChatTransfer) Consume(ctx context.Context, key, value string) error {
    var data mq.MsgChatTransfer
    if err := json.Unmarshal([]byte(value), &data); err != nil {
        return err  // ❌ 格式错误也会重试
    }

    if err := m.addChatLog(ctx, MsgId, &data); err != nil {
        return err  // ✅ 数据库错误需要重试
    }

    return m.Transfer(ctx, &ws.Push{...})  // ❌ 推送失败也会重试
}
```

**改造后：**
```go
func (m *MsgChatTransfer) Consume(ctx context.Context, key, value string) error {
    var (
        data  mq.MsgChatTransfer
        MsgId = bson.NewObjectID()
    )
    
    // 1. 消息解析失败 → 返回 nil（避免无限重试无效消息）
    if err := json.Unmarshal([]byte(value), &data); err != nil {
        m.Errorf("Failed to unmarshal message: %v", err)
        return nil  // ✅ 格式错误不重试
    }

    // 2. 数据库写入失败 → 返回 error（必须成功的致命错误）
    if err := m.addChatLog(ctx, MsgId, &data); err != nil {
        m.Errorf("Failed to add chat log: %v", err)
        return err  // ✅ 数据库错误需要重试
    }

    // 3. WebSocket 推送失败 → 返回 nil（非致命错误）
    if err := m.Transfer(ctx, &ws.Push{...}); err != nil {
        // ✅ 消息已持久化到 MongoDB
        // 用户上线时可通过拉取接口获取未读消息，因此返回 nil
        m.Errorf("Failed to push message to user (but message saved): %v", err)
    }

    // 消息已持久化，标记为处理完成
    return nil
}
```

#### 📝 文件：`apps/task/mq/internal/handler/msgTransfer/msgReadTransfer.go`

**关键改进：**
```go
func (m *MsgReadTransfer) Consume(ctx context.Context, key, value string) error {
    var data mq.MsgMarkRead

    // 1. 消息格式错误 → 不重试
    if err := json.Unmarshal([]byte(value), &data); err != nil {
        m.Errorf("Failed to unmarshal message: %v", err)
        return nil
    }

    // 2. 数据库更新失败 → 重试
    ReadRecords, err := m.UpdateChatLogRead(ctx, &data)
    if err != nil {
        m.Errorf("Failed to update chat log read status: %v", err)
        return err
    }

    // 3. 推送已读回执 → 非阻塞处理
    switch data.ChatType {
    case constants.SingleChatType:
        select {
        case m.push <- push:
            // 推送成功
        default:
            // ✅ channel 满了，丢弃推送（已读状态已保存）
            m.Errorf("Push channel full, dropping single chat read ack")
        }
    // ... 其他逻辑
    }

    return nil
}
```

---

## 五、配置调整

### 5.1 消费者并发配置

**📝 文件：`apps/task/mq/etc/task.yaml`**

```yaml
MsgChatTransfer:
  Name: MsgChatTransfer
  Brokers:
    - 127.0.0.1:9092
  Group: kafka
  Topic: msgChatTransfer
  Offset: first
  Consumers: 4  # ✅ 从 1 调整为 4，提升并发处理能力（需确保 Kafka topic partitions >= 4）

MsgReadTransfer:
  Name: MsgReadTransfer
  Brokers:
    - 127.0.0.1:9092
  Group: kafka
  Topic: msgReadTransfer
  Offset: first
  Consumers: 4  # ✅ 从 1 调整为 4，提升并发处理能力
```

### 5.2 Kafka Topic 扩容

**扩容命令（部署前执行）：**
```bash
# 扩容 msgChatTransfer
kafka-topics.sh --alter \
  --bootstrap-server 127.0.0.1:9092 \
  --topic msgChatTransfer \
  --partitions 8

# 扩容 msgReadTransfer
kafka-topics.sh --alter \
  --bootstrap-server 127.0.0.1:9092 \
  --topic msgReadTransfer \
  --partitions 8
```

**验证扩容结果：**
```bash
kafka-topics.sh --describe \
  --bootstrap-server 127.0.0.1:9092 \
  --topic msgChatTransfer

# 期望输出：
# Topic: msgChatTransfer  PartitionCount: 8  ReplicationFactor: 1
#   Topic: msgChatTransfer  Partition: 0    Leader: 0   Replicas: 0   Isr: 0
#   Topic: msgChatTransfer  Partition: 1    Leader: 0   Replicas: 0   Isr: 0
#   ... (共8个partition)
```

---

## 六、部署与运维

### 6.1 部署步骤

```
Step 1: 扩容 Kafka Topic Partition
├─ 执行 kafka-topics.sh --alter 命令
├─ 验证 partition 数量
└─ 等待 5-10 秒让 Kafka 完成元数据同步

Step 2: 部署 task-mq 服务（消费者）
├─ 滚动部署新版本
├─ 观察日志确认消费者启动正常
└─ 验证 partition 分配均匀

Step 3: 部署 im-ws 服务（生产者）
├─ 滚动部署新版本
├─ 验证消息使用 PushWithKey 发送
└─ 监控消息路由是否正确

Step 4: 监控验证
├─ 观察消费延迟（Lag）
├─ 检查错误日志
├─ 验证消息有序性
└─ 持续监控 24 小时
```

### 6.2 扩容影响分析

```
扩容前：
┌─────────────────────────────────┐
│  msgChatTransfer (1 partition)  │
│  └─ partition 0                 │
│     └─ Consumer 1              │
│        ├─ 会话A消息 ✓ 有序      │
│        ├─ 会话B消息 ✓ 有序      │
│        └─ 会话C消息 ✓ 有序      │
│  性能: 单消费者处理              │
└─────────────────────────────────┘

扩容后（Consumers: 4, Partitions: 8）:
┌──────────────────────────────────────────┐
│  msgChatTransfer (8 partitions)          │
│  ├─ partition 0 → Consumer 1            │
│  │  └─ 会话A消息 ✓ 有序                  │
│  ├─ partition 1 → Consumer 1            │
│  │  └─ 会话B消息 ✓ 有序                  │
│  ├─ partition 2 → Consumer 2            │
│  │  └─ 会话C消息 ✓ 有序                  │
│  ├─ partition 3 → Consumer 2            │
│  ├─ partition 4 → Consumer 3            │
│  ├─ partition 5 → Consumer 3            │
│  ├─ partition 6 → Consumer 4            │
│  └─ partition 7 → Consumer 4            │
│                                          │
│  性能: 4倍提升                            │
│  有序性: 会话内保证 ✓                     │
└──────────────────────────────────────────┘
```

### 6.3 回滚方案

```bash
# 如果扩容后出现问题，无法减少 partition
# 但可以：

# 方案1: 减少 Consumers 数量
# 修改 task.yaml
Consumers: 1  # 改回 1

# 方案2: 创建新 Topic，迁移回来
# 极端情况下的兜底方案
kafka-topics.sh --create \
  --bootstrap-server 127.0.0.1:9092 \
  --topic msgChatTransfer_backup \
  --partitions 1 \
  --replication-factor 1
```

---

## 七、监控与验证

### 7.1 关键监控指标

```
必须监控的指标：
├─ Consumer Lag（消费延迟）
│  └─ 命令：kafka-consumer-groups.sh --describe --group kafka
│
├─ Consumer Rebalance 频率
│  └─ 频繁 Rebalance 说明消费者不稳定
│
├─ 消息处理耗时
│  └─ 从 Push 到 Consume 完成的总耗时
│
├─ 错误率
│  └─ 数据库错误、推送失败等
│
└─ 分区分配是否均匀
   └─ 各 partition 的消息量是否均衡
```

### 7.2 验证命令

```bash
# 1. 查看消费组状态
kafka-consumer-groups.sh --describe \
  --bootstrap-server 127.0.0.1:9092 \
  --group kafka

# 2. 查看消费延迟
kafka-consumer-groups.sh --describe \
  --bootstrap-server 127.0.0.1:9092 \
  --group kafka | grep -E "LAG|msgChat"

# 3. 查看 Topic 详情
kafka-topics.sh --describe \
  --bootstrap-server 127.0.0.1:9092 \
  --topic msgChatTransfer

# 4. 查看消息内容（调试用）
kafka-console-consumer.sh \
  --bootstrap-server 127.0.0.1:9092 \
  --topic msgChatTransfer \
  --from-beginning \
  --max-messages 10
```

### 7.3 预期效果

| 指标 | 改造前 | 改造后 | 提升 |
|------|--------|--------|------|
| **并发消费者** | 1 | 4 | 4倍 |
| **理论吞吐量** | 基准 | 4倍 | 4x |
| **消息有序性** | ❌ 无保障 | ✅ 会话内有序 | 质的提升 |
| **消费延迟** | 较高 | 显著降低 | 预期 75%↓ |
| **错误重试** | 无限重试 | 智能重试 | 避免资源浪费 |

---

## 八、常见问题 FAQ

### Q1: Kafka Topic 需要手动创建吗？

**A:** 取决于 Kafka 配置 `auto.create.topics.enable`：
- `true` → 首次 Push 时自动创建（使用默认 partition 数）
- `false` → 必须手动创建（生产环境推荐）

### Q2: 能否直接修改现有 Topic 的 partitions？

**A:** 
- ✅ 可以增加 partition 数量（在线操作）
- ❌ 不能减少 partition 数量
- ⚠️ 扩容后消息路由规则会改变（但不影响会话内有序性）

### Q3: Push 是同步还是异步？

**A:** 同步阻塞模式
- Push 返回 `nil` → 消息已成功写入 Kafka
- Push 返回 `error` → 消息发送失败，需要重试

### Q4: 程序挂掉时如何保障消息不丢失？

**A:** 三层保障机制：
1. **生产者**：Push 返回 error 时重试
2. **Broker**：多副本存储
3. **消费者**：Consume 返回 error 时不提交 offset，重启后重新消费

### Q5: 如何避免消息无限重试？

**A:** 区分错误类型：
- 致命错误（数据库失败）→ 返回 `error`，触发重试
- 非致命错误（推送失败）→ 返回 `nil`，不重试
- 格式错误（无法解析）→ 返回 `nil`，不重试

### Q6: 扩容后会影响已有消息吗？

**A:** 不会
- 历史消息保持在原有 partition
- 新消息按新规则路由
- 同一会话的消息仍会路由到同一 partition（PushWithKey 保证）

### Q7: Consumers 数量应该设置为多少？

**A:** 建议：
- `Consumers <= Partitions`（超过会空闲）
- 根据处理能力调整
- 常见配置：4, 8, 16

### Q8: 如何验证消息有序性？

**A:** 测试方法：
1. 同一会话连续发送 10 条消息
2. 查看消费日志，确认按顺序处理
3. 检查数据库中的 `send_time` 字段是否有序

---

## 九、参考资料

### 9.1 go-queue 官方文档

- [go-queue GitHub](https://github.com/zeromicro/go-queue)
- [Kafka 队列使用指南](https://go-zero.dev/zh-cn/guides/queue/kafka/)

### 9.2 Kafka 核心概念

- [Apache Kafka 官方文档](https://kafka.apache.org/documentation/)
- [Kafka 分区与副本机制](https://kafka.apache.org/documentation/#replication)
- [Kafka 消费者组与 Rebalance](https://kafka.apache.org/documentation/#consumerconfigs)

### 9.3 关键配置参数

**KqConf 配置说明：**
```go
type KqConf struct {
    Brokers    []string  // Kafka 的多个 Broker 节点
    Group      string    // 消费者组
    Topic      string    // 订阅的 Topic 主题
    Offset     string    // "first" | "last"
    Conns      int       // kafka queue 数量，默认 1
    Consumers  int       // 拉取消息的 goroutine 数量，默认 8
    Processors int       // 消费消息的并发 goroutine 数量，默认 8
    MinBytes   int       // fetch 一次返回的最小字节数，默认 10K
    MaxBytes   int       // fetch 一次返回的最大字节数，默认 10M
    Username   string    // Kafka 账号（可选）
    Password   string    // Kafka 密码（可选）
}
```

**Pusher 可选参数：**
```go
kq.NewPusher(addrs, topic,
    kq.PusherChunkSize(100),           // 批量提交大小
    kq.PusherFlushInterval(time.Second), // 刷新间隔
)
```

### 9.4 相关代码文件

```
改造涉及的关键文件：
├─ apps/task/mq/mqclient/msgtransfer.go       # 消息生产者
├─ apps/task/mq/mqclient/msgmarkread.go       # 已读回执生产者
├─ apps/task/mq/internal/handler/msgTransfer/msgChatTransfer.go    # 消息消费者
├─ apps/task/mq/internal/handler/msgTransfer/msgReadTransfer.go    # 已读回执消费者
├─ apps/task/mq/etc/task.yaml                  # 消费者配置
└─ docs/kafka-message-queue-refactoring.md     # 本文档
```

---

## 十、总结与展望

### 10.1 改造成果

✅ **性能提升**：消费者并发数从 1 提升到 4，理论吞吐量提升 4 倍  
✅ **有序保障**：通过 PushWithKey 保证会话内消息严格有序  
✅ **可靠增强**：优化错误处理，避免无效重试，防止消息丢失  
✅ **可维护性**：完善的监控和验证手段，问题可追溯  

### 10.2 后续优化方向

```
短期优化（1-2周）：
├─ 添加 Push 重试机制（3次指数退避）
├─ 添加 MongoDB 唯一索引（幂等保障）
└─ 完善监控告警（消费延迟、错误率）

中期优化（1-2月）：
├─ 引入消息轨迹追踪（全链路监控）
├─ 优化批量提交策略（chunkSize/flushInterval）
└─ 添加消费端限流保护

长期优化（3-6月）：
├─ 评估是否需要更多分区（8 → 16）
├─ 考虑引入消息压缩（降低网络开销）
└─ 探索多机房容灾方案
```

### 10.3 最佳实践总结

```
1. Key 的选择原则：
   - 相同业务逻辑的消息用相同 key
   - IM 系统：按 ConversationId（会话维度）
   - 订单系统：按 OrderId（订单维度）

2. Partition 数量设置：
   - >= 最大消费者数量
   - 考虑未来扩展，预留余量
   - 常见配置：8, 16, 32

3. Consumers 数量设置：
   - <= Partition 数量
   - 根据处理能力调整
   - 超过 partition 数量的消费者会空闲

4. 错误处理原则：
   - 区分致命错误和非致命错误
   - 已持久化的消息推送失败不重试
   - 格式错误的消息直接丢弃

5. 监控要点：
   - 消费延迟（Lag）
   - 分区分配是否均匀
   - 消费者是否健康
   - 错误率和重试次数
```

---

> 📝 **文档维护**：本文档应随代码迭代持续更新  
> 👥 **审核人员**：开发团队、运维团队  
> 🔄 **更新频率**：每次重大改造后更新

---

**🎉 改造完成！祝系统运行稳定高效！**
