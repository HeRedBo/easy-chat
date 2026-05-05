 gRPC 幂等性拦截器实现文档

## 一、幂等性原理

### 1.1 什么是幂等性

**幂等性（Idempotency）** 是指同一个操作无论执行多少次，其产生的效果都与执行一次完全相同。在数学上，满足 `f(f(x)) = f(x)` 的函数即具有幂等性。

在分布式系统中，幂等性是保障数据一致性和系统稳定性的核心手段。以下场景必须依赖幂等性保护：

- **网络重试**：gRPC 默认开启自动重试，网络抖动会导致同一请求被服务端多次接收
- **用户重复点击**：前端按钮未做防抖，用户快速双击提交按钮
- **消息队列重复投递**：Kafka 消费者组重平衡或生产者重试导致消息被消费多次
- **超时重发**：客户端认为请求超时，实际上服务端已经处理完毕，再次发送造成重复处理

没有幂等性保护，上述场景会导致：重复创建资源、重复扣款、重复发送消息等严重业务问题。

### 1.2 为什么需要状态机

#### 并发场景下的核心问题："正在处理中"的中间状态

假设仅使用简单去重（Set 或 Bloom Filter）：

- 请求 A 到达服务端，开始执行业务逻辑（耗时 2 秒）
- 请求 B（同一笔业务）在 1 秒后到达，此时请求 A 尚未完成
- 简单去重发现该请求"不存在"，于是放行请求 B
- 请求 A 和 B 同时执行业务，造成数据不一致

这个场景的关键在于：**请求在执行业务的过程中存在一个"正在处理中"的中间状态**，简单去重无法识别这种状态。

#### 三种方案对比

| 方案 | 原理 | 能否处理并发重复 | 能否返回历史结果 | 生产可用性 |
|------|------|-----------------|-----------------|-----------|
| 简单去重 | 基于 Set / Bloom Filter 记录已处理请求的 ID | 否（无法识别"处理中"状态） | 否 | 低 |
| 结果缓存 | 业务完成后将结果缓存到 Redis，重复请求直接返回缓存 | 否（并发时可能同时穿透到业务层） | 是 | 中 |
| **状态机方案** | **PROCESSING → SUCCESS / FAILED，配合分布式锁** | **是（PROCESSING 状态 + 锁排斥并发请求）** | **是（SUCCESS/FAILED 状态直接返回缓存结果）** | **高** |

**状态机是实现生产级幂等性的标准工程手段**，因为它同时解决了"并发防重"和"结果复用"两个问题。

### 1.3 状态机设计

本项目采用三状态有限状态机：

```
                    ┌─────────────────┐
                    │   （初始状态）   │
                    └────────┬────────┘
                             │ TryAcquire 加锁成功
                             ▼
                    ┌─────────────────┐
         ┌─────────│   PROCESSING    │◄────────┐
         │         │   正在处理中     │         │
         │         └────────┬────────┘         │
         │                  │                   │
  业务异常 │          业务执行完成              │ 重复请求
  加锁失败 │                  │              检测到锁
         │                  ▼                   │
         │         ┌─────────────────┐          │
         └────────►│    SUCCESS      │          │
                   │   执行成功      │──────────┘
                   └─────────────────┘    返回缓存结果
                   ┌─────────────────┐
                   │     FAILED      │
                   │   执行失败      │
                   └─────────────────┘
```

#### 三种状态含义与触发条件

| 状态 | 含义 | 触发条件 | 后续行为 |
|------|------|---------|---------|
| `PROCESSING` | 请求正在被处理中 | TryAcquire 成功获取分布式锁，首次写入 Hash | 执行业务逻辑 |
| `SUCCESS` | 请求已处理完成且成功 | 业务 handler 执行完毕，无错误 | 返回缓存结果；后续重复请求直接返回缓存数据 |
| `FAILED` | 请求已处理完成但失败 | 业务 handler 执行完毕，有错误 | 返回缓存的错误信息；后续重复请求可再次尝试（视业务需求） |

#### 状态转移规则

```
（空）        ──TryAcquire 成功──►  PROCESSING
PROCESSING    ──业务成功──────────►  SUCCESS
PROCESSING    ──业务失败──────────►  FAILED
SUCCESS       ──重复请求──────────►  SUCCESS（直接返回缓存）
PROCESSING    ──重复请求──────────►  拒绝："操作正在处理中，请勿重复提交"
```

## 二、go-zero 拦截器机制

### 2.1 拦截器概述

gRPC 拦截器（Interceptor）是 **AOP（面向切面编程）思想在 RPC 层的具体实现**。它允许开发者在 RPC 请求的发送/接收前后插入通用逻辑（如日志、鉴权、限流、幂等性校验等），而无需修改具体的业务 handler 代码。

拦截器本质上是一个高阶函数，它在请求真正到达业务代码之前（或之后）执行预处理（或后处理）逻辑，然后将请求传递给下一个处理环节。

#### 服务端拦截器 vs 客户端拦截器

| 维度 | 服务端拦截器（Server Interceptor） | 客户端拦截器（Client Interceptor） |
|------|----------------------------------|----------------------------------|
| 作用位置 | RPC 服务端，请求到达业务 handler 之前 | RPC 客户端，请求发出之前 |
| 典型用途 | 鉴权、限流、日志记录、幂等性校验 | 链路追踪信息注入、重试、超时设置、幂等 ID 注入 |
| gRPC 类型签名 | `grpc.UnaryServerInterceptor` | `grpc.UnaryClientInterceptor` |
| go-zero 注册方式 | `s.AddUnaryInterceptors(...)` | `zrpc.WithUnaryClientInterceptor(...)` |
| 执行顺序 | 按注册顺序依次执行（先注册先执行） | 按注册顺序依次执行 |

### 2.2 go-zero 内置拦截器

go-zero 的 `zrpc` 框架在创建 gRPC Server / Client 时，默认已经注入了一系列生产级拦截器：

| 拦截器名称 | 类型 | 功能说明 |
|-----------|------|---------|
| `RecoverInterceptor` | 服务端 | 拦截业务 handler 的 panic，防止单个请求拖垮整个服务进程 |
| `PrometheusInterceptor` | 服务端 | 自动采集 RPC 请求的 QPS、延迟（P99/P95）、错误率等 Metrics |
| `TracingInterceptor` | 服务端/客户端 | 集成 OpenTelemetry/Jaeger，注入和传递 trace_id、span_id |
| `BreakerInterceptor` | 客户端 | 实现熔断器模式，当后端服务异常率达到阈值时自动熔断，快速失败 |
| `SheddingInterceptor` | 服务端 | 实现自适应负载保护，当服务端负载过高时主动拒绝部分请求 |
| `TimeoutInterceptor` | 客户端 | 为 RPC 调用设置超时时间，防止长时间阻塞 |

以上拦截器由 go-zero 自动装配，开发者无需手动注册即可享受完整的可观测性和稳定性保障。

### 2.3 自定义拦截器注册方式

go-zero 提供了两种自定义拦截器的注册方式：

#### AddUnaryInterceptors（服务端）

```go
s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
    // 注册业务服务
})
// 追加自定义服务端拦截器
s.AddUnaryInterceptors(rpcserver.LogInterceptor)
s.AddUnaryInterceptors(interceptor.NewIdempotenceServer(...))
```

#### WithUnaryClientInterceptor（客户端）

```go
client := socialclient.NewSocial(
    zrpc.MustNewClient(c.SocialRpc,
        zrpc.WithUnaryClientInterceptor(interceptor.DefaultIdempotentClient),
    ),
)
```

#### 执行顺序说明

- **服务端拦截器**：按 `AddUnaryInterceptors` 的调用顺序依次执行。先注册的拦截器先处理请求，后处理响应（洋葱模型）
- **客户端拦截器**：按 `WithUnaryClientInterceptor` 的调用顺序依次执行
- **注意**：自定义拦截器在 go-zero 内置拦截器**之后**执行。即请求先经过 Recover、Prometheus 等内置拦截器，再到达自定义拦截器

## 三、项目实现详解

### 3.1 架构设计

本项目幂等性实现采用**四层架构**：

```
┌─────────────────────────────────────────────────────────────┐
│                      接口定义层（Idempotent）                  │
│  Identify / IsIdempotentMethod / TryAcquire / SaveResp       │
├─────────────────────────────────────────────────────────────┤
│                      客户端拦截器层                           │
│  NewIdempotenceClient → 从 context 提取 ID，写入 gRPC metadata │
├─────────────────────────────────────────────────────────────┤
│                      服务端拦截器层                           │
│  NewIdempotenceServer → 从 metadata 提取 ID，调用 TryAcquire   │
├─────────────────────────────────────────────────────────────┤
│                      默认实现层（Redis）                       │
│  defaultIdempotent → Redis Hash + SetnxEx 分布式锁           │
└─────────────────────────────────────────────────────────────┘
```

#### Idempotent 接口的 4 个方法

```go
type Idempotent interface {
    // Identify 从 context 中提取或生成请求的唯一标识，并与 method 拼接
    // 返回格式: "{uuid}.{fullMethod}"
    Identify(ctx context.Context, method string) string

    // IsIdempotentMethod 判断某个 gRPC 方法是否配置了幂等性保护
    // 只有返回 true 的方法才会进入幂等性校验流程
    IsIdempotentMethod(fullMethod string) bool

    // TryAcquire 幂等性核心校验逻辑：
    //   - 如果已有 SUCCESS/FAILED 结果，返回 (resp, false, nil) → 直接返回缓存
    //   - 如果获取到分布式锁，返回 (nil, true, nil) → 放行执行业务
    //   - 如果未获取到锁，返回 (nil, false, nil) → 正在处理中，拒绝重复
    //   - 如果发生 Redis 错误，返回 (nil, false, err)
    TryAcquire(ctx context.Context, id string) (resp interface{}, isAcquire bool, err error)

    // SaveResp 业务执行完成后，将结果（成功或失败）保存到 Redis
    // 供后续重复请求直接返回，避免重复执行业务
    SaveResp(ctx context.Context, id string, resp interface{}, respErr error) error
}
```

### 3.2 完整数据流（8 个阶段）

以下是客户端发起 HTTP 请求调用 Social RPC `GroupCreate` 方法时，幂等性拦截器的完整时序流程：

```
  HTTP Client          Social API (HTTP)              Social RPC (gRPC)
      │                       │                              │
      │  ① POST /v1/social/group                         │
      │──────────────────────>│                              │
      │                       │ ② IdempotenceMiddleware      │
      │                       │    ContextWithIdempotentID() │
      │                       │    → 生成 UUID 存入 context  │
      │                       │                              │
      │                       │ ③ Client Interceptor         │
      │                       │    Identify() 取出 UUID      │
      │                       │    拼接 method → rpcId       │
      │                       │    metadata.Pairs(DKey, rpcId)
      │                       │                              │
      │                       │────────④ gRPC 网络传输───────>│
      │                       │                              │
      │                       │                              │⑤ Server Interceptor
      │                       │                              │  metadata.ValueFromIncomingContext
      │                       │                              │  提取 DKey → key
      │                       │                              │
      │                       │                              │⑥ TryAcquire(key)
      │                       │                              │  ├─ 查询 Redis Hash
      │                       │                              │  │   ├─ status=SUCCESS/FAILED
      │                       │                              │  │   │   → 返回缓存结果（短路）
      │                       │                              │  │   ├─ status=PROCESSING 或空
      │                       │                              │  │   │   → 尝试 SetnxEx 加锁
      │                       │                              │  │       ├─ 加锁成功
      │                       │                              │  │       │   → 写入 PROCESSING
      │                       │                              │  │       │   → 放行执行业务
      │                       │                              │  │       └─ 加锁失败
      │                       │                              │  │           → 返回"正在处理中"
      │                       │                              │
      │                       │                              │⑦ 执行业务 handler
      │                       │                              │   GroupCreate 业务逻辑
      │                       │                              │
      │                       │                              │⑧ SaveResp(key, resp, err)
      │                       │                              │   写入 SUCCESS/FAILED + data
      │                       │<────────返回响应─────────────│
      │<──────────────────────│                              │
```

#### 阶段详细说明

1. **HTTP 中间件生成 UUID**
   - 请求进入 Social API HTTP 层
   - `IdempotenceMiddleware` 调用 `interceptor.ContextWithIdempotentID(r.Context())`
   - 使用 `utils.NewUuid()` 生成唯一标识，以 `TKey` 为键存入 `context`

2. **UUID 存入 context**
   - `context.WithValue(ctx, TKey, uuid)` 将幂等 ID 绑定到当前请求上下文
   - 后续所有使用同一 context 的 RPC 调用都可以读取到该 ID

3. **客户端拦截器从 context 取出 ID，拼接 method，写入 gRPC metadata**
   - `DefaultIdempotentClient` 拦截 outgoing RPC 请求
   - `Identify()` 从 context 读取 `TKey`，若不存在则兜底生成新的 UUID
   - 拼接格式：`rpcId = fmt.Sprintf("%v.%s", id, method)`
   - 通过 `metadata.NewOutgoingContext(ctx, metadata.Pairs(DKey, rpcId))` 将 ID 放入 gRPC 请求头

4. **网络传输**
   - gRPC 请求携带着 `DKey` 头部，经过 HTTP/2 链路到达 Social RPC 服务端

5. **服务端拦截器从 metadata 提取 ID**
   - `NewIdempotenceServer` 拦截 incoming RPC 请求
   - `metadata.ValueFromIncomingContext(ctx, DKey)` 提取幂等 ID
   - 若 ID 为空或该方法不在幂等列表中，直接放行（不做幂等处理）

6. **TryAcquire 查询 Redis 进行幂等校验（三条路径）**
   - **路径 A（结果缓存命中）**：Redis Hash 中 `status` 为 `SUCCESS` 或 `FAILED`，反序列化 `data` 字段直接返回缓存结果
   - **路径 B（首次请求）**：Redis 无记录或记录已过期，通过 `SetnxEx` 原子加锁成功，写入 `PROCESSING` 状态，放行执行业务
   - **路径 C（并发重复）**：`SetnxEx` 加锁失败，说明已有其他请求正在处理，返回错误 `"操作正在处理中，请勿重复提交"`

7. **执行业务 / 返回缓存 / 拒绝重复**
   - 路径 A：直接返回缓存结果，不执行业务 handler
   - 路径 B：调用 `handler(ctx, req)` 执行业务逻辑
   - 路径 C：直接返回错误

8. **SaveResp 保存结果**
   - 业务 handler 执行完成后，无论成功或失败
   - 将 `status`（`SUCCESS` 或 `FAILED`）、`data`（JSON 序列化的响应）、`error`（错误信息）写入 Redis Hash
   - 设置过期时间为 24 小时

### 3.3 Redis 数据结构

幂等性模块使用两类 Redis Key：

#### Hash Key（主数据）

- **Key 格式**：`{rpcId}`（即 `{uuid}.{fullMethod}`）
- **数据结构**：Redis Hash
- **字段说明**：

| 字段 | 类型 | 含义 |
|------|------|------|
| `status` | String | 当前状态：`PROCESSING` / `SUCCESS` / `FAILED` |
| `data` | String | 业务响应结果的 JSON 序列化字符串 |
| `error` | String | 业务执行失败的错误信息（若有） |
| `time` | String | 状态写入时间（RFC3339 格式） |

- **TTL**：24 小时（`resultTTL = 24 * time.Hour`）

#### Lock Key（分布式锁）

- **Key 格式**：`{rpcId}:lock`
- **数据结构**：Redis String（值为 `"1"`）
- **TTL**：10 秒（`lockTTL = 10 * time.Second`）
- **操作命令**：`SET key value NX EX 10`

#### 设计选择：为什么用两个 Key 而不是一个

若将锁状态也存入 Hash，需要 Lua 脚本才能保证"读取 + 判断 + 写入"的原子性。而使用独立的 `SetnxEx`（`SET NX EX`）命令，Redis 单线程天然保证原子性，无需引入 Lua 脚本复杂度。同时，锁的 TTL（10 秒）远短于结果缓存 TTL（24 小时），分离设计便于独立控制生命周期。

### 3.4 分布式锁实现

#### SetnxEx 原子操作原理

```
客户端请求:  SET {id}:lock 1 NX EX 10
                      │
                      ▼
              ┌───────────────┐
              │   Redis 单线程  │
              │   命令队列      │
              └───────────────┘
                      │
              原子执行以下逻辑：
              1. 检查 Key 是否存在
              2. 若不存在 → 设置 Key，值为 "1"，过期时间 10s → 返回 OK (true)
              3. 若存在   → 不做任何操作 → 返回 nil (false)
```

Redis 的 `SET NX EX` 命令将"检查不存在"和"设置值+过期时间"两个操作合并为单一原子命令，从根本上杜绝了并发竞争窗口。

#### 为什么选择 SetnxEx + Hash 组合而不是 RedLock

| 方案 | 实现复杂度 | 可靠性 | 适用场景 | 本项目选择理由 |
|------|-----------|--------|---------|--------------|
| `SetnxEx` + Hash | 低 | 高（单 Redis 实例） | 单 Redis 实例部署 | 本项目使用单 Redis 实例，`SetnxEx` 已能满足需求 |
| RedLock（红锁） | 高 | 极高（多实例共识） | 多 Redis Master 部署 | 需要 5 个独立 Redis 实例，运维成本高，对于幂等性场景过度设计 |

对于本项目当前的单 Redis 实例架构，`SetnxEx` 是性价比最高的选择。RedLock 主要用于金融级强一致性场景。

#### 并发时序图

以下展示三个并发请求（A、B、C）请求同一幂等 Key 时的执行时序：

```
时间轴 ──────────────────────────────────────────────────────────────>

请求 A:  ├─TryAcquire─┤
         │  Hget: 空   │
         │ SetnxEx: OK │  ← 加锁成功
         │ Hmset: PROC │
         │◄──── true ──┤
         │             │
         │   执行业务   │
         │   (2 秒)    │
         │             │
         │ SaveResp:   │
         │ Hmset: SUCC │
         └─────────────┘

请求 B:          ├──── TryAcquire ────┤
                 │    Hget: PROC      │
                 │   SetnxEx: FAIL    │  ← 锁被 A 持有
                 │◄──── false ────────┤
                 │ 返回"正在处理中"    │
                 └────────────────────┘

请求 C:                              ├──── TryAcquire ────┤
                                     │   Hget: SUCCESS    │
                                     │ 反序列化 data      │
                                     │◄──── resp ─────────┤
                                     │ 直接返回缓存结果    │
                                     └────────────────────┘
```

- 请求 A 首次到达，加锁成功，执行业务，最后保存结果
- 请求 B 在 A 业务执行期间到达，加锁失败，被优雅拒绝
- 请求 C 在 A 完成后到达，命中结果缓存，直接返回历史结果

### 3.5 核心代码解读

#### 3.5.1 客户端拦截器：将幂等 ID 注入 gRPC Metadata

```go
// pkg/interceptor/idempotence.go

// NewIdempotenceClient 客户端拦截器：把ID塞入请求头
func NewIdempotenceClient(idempotent Idempotent) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply any,
        cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        // 获取唯一的key（从 context 读取 TKey，若不存在则生成新 UUID）
        identify := idempotent.Identify(ctx, method)
        // 在 RPC 请求的 metadata 中添加 DKey 头部
        ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(DKey, identify))
        // 继续执行后续拦截器和真正的 RPC 调用
        return invoker(ctx, method, req, reply, cc, opts...)
    }
}
```

**关键点**：
- `Identify()` 将 UUID 与 `fullMethod`（如 `/social.social/GroupCreate`）拼接，确保同一笔业务在不同方法上的幂等 Key 不冲突
- `metadata.Pairs(DKey, identify)` 使用 gRPC metadata 机制在 HTTP/2 头部中透传幂等 ID

#### 3.5.2 服务端拦截器：幂等性核心入口

```go
// pkg/interceptor/idempotence.go

// NewIdempotenceServer 服务端拦截器: 幂等核心入口
func NewIdempotenceServer(idempotent Idempotent) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler) (resp any, err error) {
        // 从 gRPC metadata 中提取幂等 ID
        identify := metadata.ValueFromIncomingContext(ctx, DKey)
        // 若 ID 为空，或该方法不在幂等方法列表中，直接放行
        if len(identify) == 0 || !idempotent.IsIdempotentMethod(info.FullMethod) {
            return handler(ctx, req)
        }
        key := identify[0]
        logx.Infof("【幂等拦截】请求进入 → method: %s, id: %s", info.FullMethod, key)

        // 进入幂等校验
        r, acquire, err := idempotent.TryAcquire(ctx, key)
        if err != nil {
            logx.Infof("【幂等拦截】重复请求，直接返回结果 → id: %s, err: %v", key, err)
            return nil, err
        }

        // 路径 A：已有结果，直接返回缓存
        if r != nil {
            fmt.Println("--- 任务已经执行完了 ", identify)
            return r, nil
        }

        // 路径 C：未获取到锁，正在执行中
        if !acquire {
            return nil, errors.New("操作正在处理中，请勿重复提交")
        }

        // 路径 B：获取到锁，执行业务
        resp, err = handler(ctx, req)
        fmt.Println("---- 执行任务", key)
        // 保存结果，忽略保存错误（不影响业务结果）
        _ = idempotent.SaveResp(ctx, key, resp, err)
        return resp, err
    }
}
```

**关键点**：
- 三条路径清晰分离：结果缓存命中（`r != nil`）、并发拒绝（`!acquire`）、首次放行（`acquire == true`）
- `SaveResp` 的错误被忽略（`_ = ...`），避免缓存写入失败影响业务正常返回

#### 3.5.3 TryAcquire：幂等性校验核心逻辑

```go
// pkg/interceptor/idempotence.go

func (d *defaultIdempotent) TryAcquire(ctx context.Context, id string) (resp interface{}, isAcquire bool, err error) {
    // 1. 查询数据（redis:nil 代表不存在，不是错误，因此忽略 err）
    resultData, _ := d.Redis.HgetCtx(ctx, id, "data")
    status, _ := d.Redis.HgetCtx(ctx, "status")

    // 路径 A：已经有结果（SUCCESS 或 FAILED），直接返回缓存
    if status == statusSuccess || status == statusFailed {
        var resp any
        err = json.Unmarshal([]byte(resultData), &resp)
        if err != nil {
            return nil, false, err
        }
        return resp, false, nil
    }

    // 路径 B/C：尝试加锁，进入 PROCESSING 状态
    // 使用 SetnxEx 保证"检查+加锁+设置过期时间"的原子性
    ok, err := d.Redis.SetnxExCtx(ctx, id+":lock", "1", int(lockTTL.Seconds()))
    if err != nil {
        return nil, false, err
    }
    // 拿不到锁 → 路径 C：正在执行中
    if !ok {
        return nil, false, nil
    }

    // 拿到锁 → 路径 B：写入 PROCESSING 状态，放行业务
    err = d.Redis.HmsetCtx(ctx, id, map[string]string{
        "status": statusProcessing,
        "time": time.Now().Format(time.RFC3339),
    })
    if err != nil {
        return nil, false, err
    }
    // 设置 Hash Key 的过期时间为 24 小时
    _ = d.Redis.ExpireCtx(ctx, id, int(resultTTL.Seconds()))
    return nil, true, nil
}
```

**关键点**：
- `HgetCtx` 的 err 被忽略，因为 Redis 返回 `nil`（键不存在）在业务上是正常场景，不代表错误
- `SetnxExCtx` 的原子性是整个幂等性正确性的基石
- 加锁成功后立即写入 `PROCESSING` 状态，确保后续并发请求在加锁失败后能通过 Hget 看到正在处理中的状态（虽然当前实现中，加锁失败直接拒绝，不二次查 Hash）

#### 3.5.4 SaveResp：结果缓存写入

```go
// pkg/interceptor/idempotence.go

func (d *defaultIdempotent) SaveResp(ctx context.Context, id string, resp interface{}, respErr error) error {
    status := statusSuccess
    if respErr != nil {
        status = statusFailed
    }
    data, err := json.Marshal(resp)
    if err != nil {
        return err
    }
    _ = d.Redis.HmsetCtx(ctx, id, map[string]string{
        "status": status,
        "data":   string(data),
        "error":  fmt.Sprintf("%v", respErr),
    })
    _ = d.Redis.ExpireCtx(ctx, id, int(resultTTL))
    return nil
}
```

**关键点**：
- 成功和失败状态都会缓存，确保重复请求能得到一致的响应
- 响应数据通过 `json.Marshal` 序列化后存入 Hash 的 `data` 字段

## 四、项目中的集成方式

### 4.1 各服务拦截器配置

| 服务 | 类型 | 已注册拦截器 | 是否启用幂等性 |
|------|------|-------------|--------------|
| User RPC (`apps/user/rpc/user.go`) | 服务端 | `rpcserver.LogInterceptor` | 否 |
| Social RPC (`apps/social/rpc/social.go`) | 服务端 | `rpcserver.LogInterceptor` + `interceptor.NewIdempotenceServer` | **是** |
| IM RPC (`apps/im/rpc/im.go`) | 服务端 | `rpcserver.LogInterceptor` | 否 |
| Social API (`apps/social/api/internal/svc/servicecontext.go`) | 客户端 | `interceptor.DefaultIdempotentClient`（仅 Social RPC 客户端） | 客户端注入 |

**配置解读**：
- 只有 **Social RPC** 服务端启用了幂等性拦截器，这是因为目前项目中只有社交相关的写操作（如创建群组）配置了幂等保护
- **Social API** 在创建 `Social` RPC 客户端时，通过 `zrpc.WithUnaryClientInterceptor(interceptor.DefaultIdempotentClient)` 注入了客户端幂等拦截器，确保 outgoing 请求携带幂等 ID
- **User RPC** 和 **IM RPC** 当前未启用幂等性，但未来可以通过同样的方式（`AddUnaryInterceptors` + `NewDefaultIdempotent`）快速接入

### 4.2 完整调用链路图

以 **Social API 创建群组** 为例，完整调用链路如下：

```
┌─────────────┐      HTTP/1.1       ┌─────────────────────┐
│   前端 /    │ ──POST /v1/social/group──> │   Social API Server  │
│  API 调用方  │                     │   (apps/social/api)  │
└─────────────┘                     └──────────┬──────────┘
                                               │
                                               │ ① RequestIDMiddleware
                                               │    生成 X-Request-ID
                                               │
                                               │ ② IdempotenceMiddleware
                                               │    ContextWithIdempotentID()
                                               │    生成幂等 UUID 存入 context
                                               │
                                               │ ③ Handler 处理
                                               │    调用 GroupCreateHandler
                                               │
                                               │ ④ RPC Client 调用 Social.GroupCreate
                                               │    DefaultIdempotentClient
                                               │    → Identify() 生成 rpcId
                                               │    → metadata.Pairs(DKey, rpcId)
                                               │
                                               │────────gRPC HTTP/2─────────>
                                                          │
                                               ┌──────────┴──────────┐
                                               │   Social RPC Server  │
                                               │  (apps/social/rpc)   │
                                               └──────────┬──────────┘
                                                          │
                                                          │ ⑤ LogInterceptor
                                                          │    记录 RPC 日志
                                                          │
                                                          │ ⑥ IdempotenceServer
                                                          │    TryAcquire(key)
                                                          │    ├─ 命中缓存 → 直接返回
                                                          │    ├─ 加锁成功 → 执行业务
                                                          │    └─ 加锁失败 → 拒绝重复
                                                          │
                                                          │ ⑦ SocialServer.GroupCreate
                                                          │    实际业务逻辑
                                                          │
                                                          │ ⑧ SaveResp()
                                                          │    缓存结果到 Redis
                                                          │
                                                          │<──────返回响应──────────
                                               ┌──────────┴──────────┐
                                               │   Social API Server  │
                                               │    包装 HTTP 响应     │
                                               └──────────┬──────────┘
                                                          │
                                               <──────────┘
```

### 4.3 当前配置的幂等方法

```go
// pkg/interceptor/idempotence.go

var defaultIdempotentMethods = map[string]bool{
    "/social.social/GroupCreate": true,
}
```

当前仅有 **`/social.social/GroupCreate`**（创建群组）一个方法启用了幂等性保护。

新增幂等方法的方式：在创建 `defaultIdempotent` 实例时，通过 `methods ...string` 可变参数追加：

```go
// 示例：在 social.go 中扩展更多幂等方法
idempotent := interceptor.NewDefaultIdempotent(c.Redisx,
    "/social.social/GroupCreate",
    "/social.social/FriendPutIn",
    "/user.user/UserRegister",
)
s.AddUnaryInterceptors(interceptor.NewIdempotenceServer(idempotent))
```

## 五、后续改进方向

### 5.1 高可用改进

#### 当前单 Redis 实例的局限性

当前幂等性依赖单点 Redis，若 Redis 实例故障：
- 所有幂等校验将失败（或穿透到业务层）
- 已缓存的幂等结果丢失（虽然不影响正确性，但会丢失防重能力）

#### RedLock 算法方案

若后续架构演进为 Redis Cluster 或多 Master 部署，可考虑引入 RedLock 算法：
- 在 N（通常 N=5）个独立的 Redis Master 上同时尝试加锁
- 只有当在大多数节点（`N/2 + 1`）上加锁成功，且总耗时小于锁 TTL 时，才认为加锁成功
- 释放锁时需要向所有节点发送解锁指令

#### Redis Cluster 下的考虑

- 若使用 Redis Cluster，需要确保幂等 Key 的哈希标签（Hash Tag）落在同一 Slot，以保证 `Hmset` 和 `Expire` 的事务性
- 或使用 Redis Cluster 下的 Redisson 等成熟分布式锁实现

### 5.2 功能扩展

#### 扩展更多幂等方法

当前仅 `GroupCreate` 配置了幂等保护。建议将以下写操作纳入幂等方法列表：

- `/social.social/FriendPutIn`（发起好友申请）
- `/social.social/FriendPutInHandle`（处理好友申请）
- `/social.social/GroupPutIn`（申请入群）
- `/user.user/UserRegister`（用户注册）
- `/im.im/CreateConversation`（创建会话）

**配置方式**：在调用 `NewDefaultIdempotent` 时追加方法名即可：

```go
idempotent := interceptor.NewDefaultIdempotent(c.Redisx,
    "/social.social/GroupCreate",
    "/social.social/FriendPutIn",
)
```

#### 客户端重试 + 幂等性的配合使用

本项目已在 `pkg/zrpcx/retry.go` 中实现了全局 gRPC 重试配置。当客户端重试与幂等性配合时：
- 首次请求超时，客户端自动重试
- 服务端实际已处理完成，第二次请求命中幂等缓存，直接返回结果
- **注意**：若服务端仍在 `PROCESSING` 状态，重试请求会被拒绝。建议客户端重试策略配合退避算法（Backoff），给服务端留出足够的处理时间

#### 幂等性 Key 的自定义策略

当前 `Identify()` 的实现完全依赖 UUID + Method 拼接。某些场景下可以基于业务参数生成更语义化的幂等 Key：

```go
// 示例：基于用户ID + 业务参数生成幂等 Key
func (d *defaultIdempotent) Identify(ctx context.Context, method string) string {
    // 从 context 或请求参数中提取业务标识
    userID := ctxdata.GetUid(ctx) // 假设能从 context 获取当前用户 ID
    bizKey := ctx.Value(BizKey).(string)
    return fmt.Sprintf("%s:%s:%s", userID, method, bizKey)
}
```

基于业务参数的幂等 Key 优势：同一用户在同一业务场景下的重复请求天然去重，无需依赖 UUID 的传递。

### 5.3 可观测性增强

#### 添加幂等命中率监控指标

当前实现缺少对幂等性拦截效果的量化监控。建议添加以下 Metrics：

| 指标名 | 类型 | 含义 |
|--------|------|------|
| `idempotency_hit_total` | Counter | 命中结果缓存的请求数 |
| `idempotency_reject_total` | Counter | 因"正在处理中"被拒绝的请求数 |
| `idempotency_miss_total` | Counter | 首次请求（未命中缓存）的请求数 |
| `idempotency_lock_wait_ms` | Histogram | 锁等待时间分布 |

#### 日志优化

当前代码中存在两处 `fmt.Println`，应统一替换为 `logx`：

```go
// 当前实现（需要优化）
fmt.Println("--- 任务已经执行完了 ", identify)
fmt.Println("---- 执行任务", key)

// 建议替换为
logx.WithContext(ctx).Infof("【幂等拦截】任务已执行完成，直接返回缓存 → id: %s", identify)
logx.WithContext(ctx).Infof("【幂等拦截】首次请求，开始执行业务 → id: %s", key)
```

使用 `logx` 的好处：
- 统一日志格式，支持 JSON 输出
- 与 go-zero 的日志收集体系集成
- 支持日志分级（Info / Error / Slow 等）

#### 添加 Prometheus Metrics

在 `NewIdempotenceServer` 中注入统计逻辑：

```go
// 伪代码示例
func NewIdempotenceServer(idempotent Idempotent) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
        // ... 幂等校验逻辑 ...
        if r != nil {
            metricx.IdempotencyHit.Inc() // 命中缓存
            return r, nil
        }
        if !acquire {
            metricx.IdempotencyReject.Inc() // 并发拒绝
            return nil, errors.New("操作正在处理中，请勿重复提交")
        }
        metricx.IdempotencyMiss.Inc() // 首次请求
        // ...
    }
}
```

### 5.4 健壮性改进

#### SaveResp 中 ExpireCtx 的参数 Bug

```go
// pkg/interceptor/idempotence.go  当前实现（存在 Bug）
_ = d.Redis.ExpireCtx(ctx, id, int(resultTTL))
```

**问题分析**：
- `resultTTL = 24 * time.Hour`，在 Go 中其底层值为 `86400000000000`（纳秒数，int64）
- `int(resultTTL)` 直接将该纳秒值转为 int，得到 `86400000000000`
- Redis 的 `EXPIRE` 命令参数单位为**秒**，且最大支持值约为 `2^31 - 1`（约 68 年，但部分 Redis 版本对极大值会截断或报错）
- 正确的做法应为 `int(resultTTL.Seconds())`，即 `86400`（24 小时的秒数）

**修复方案**：

```go
// 修复后
_ = d.Redis.ExpireCtx(ctx, id, int(resultTTL.Seconds()))
```

> 注：`TryAcquire` 方法中已正确使用 `int(resultTTL.Seconds())`，`SaveResp` 中的不一致属于代码笔误。

#### 锁释放时机优化

当前分布式锁的释放完全依赖 TTL（10 秒）自动过期。在业务执行时间远小于 10 秒的场景下，锁会 unnecessarily 长时间持有，影响用户体验。

**优化方案**：业务执行完成后主动释放锁：

```go
// 优化后的服务端拦截器逻辑
resp, err = handler(ctx, req)
// 业务完成后立即释放锁，让后续请求可以尽快查询结果
_ = d.Redis.DelCtx(ctx, key+":lock")
// 保存结果
_ = idempotent.SaveResp(ctx, key, resp, err)
```

> 注意：释放锁和保存结果之间仍有一个极小的时间窗口，若此时有并发请求进入，可能会看到"无锁且无结果"的状态而穿透到业务层。更严谨的方案是在 `SaveResp` 中原子性地完成"写结果 + 删锁"（需 Lua 脚本）。

#### PROCESSING 状态超时兜底

若业务 handler 在执行过程中发生进程崩溃（如 OOM、机器宕机），Redis 中的 `PROCESSING` 状态和锁将一直保留（锁 10 秒后过期，但 Hash 的 `PROCESSING` 状态会保留 24 小时）。

**后果**：同一笔业务的重复请求在 24 小时内都会因查询到 `PROCESSING` 状态后加锁失败而被拒绝（或看到空结果后加锁成功但 Hash 中旧状态存在）。

**优化方案**：

1. **PROCESSING 状态附带时间戳，增加超时清理逻辑**：
   ```go
   // TryAcquire 中检查 PROCESSING 状态的持续时间
   if status == statusProcessing {
       t, _ := d.Redis.HgetCtx(ctx, id, "time")
       processTime, _ := time.Parse(time.RFC3339, t)
       if time.Since(processTime) > 5*time.Minute {
           // 超过 5 分钟仍处理中，视为异常，允许重新执行
           _ = d.Redis.DelCtx(ctx, id)
           // 重新走首次请求流程
       }
   }
   ```

2. **引入定时任务扫描并清理异常 PROCESSING 状态**：
   - 使用 go-zero 的 `cron` 或独立 job 服务
   - 扫描 Redis 中所有幂等 Key，检查 `status == PROCESSING` 且 `time` 超过阈值的记录
   - 将其状态重置或删除

3. **缩短结果缓存 TTL**：
   - 对于可能长时间 PROCESSING 的业务，可适当缩短 `resultTTL`（如从 24 小时改为 1 小时），降低异常状态的影响范围

## 六、总结

### 核心设计思想

本项目幂等性方案的核心设计可以概括为 **"状态机 + 分布式锁 + 结果缓存"** 的三位一体：

1. **状态机**（PROCESSING → SUCCESS / FAILED）：精确描述请求的生命周期，解决并发场景下"处理中"状态的识别问题
2. **分布式锁**（Redis SetnxEx）：确保同一笔业务在分布式环境下只有一个实例在执行
3. **结果缓存**（Redis Hash）：业务执行完成后缓存结果，让重复请求直接返回历史响应，避免重复执行业务逻辑

### 适用场景

- **网络自动重试**：gRPC 客户端重试导致同一请求多次到达服务端
- **用户重复提交**：前端按钮未做防抖，用户快速点击
- **消息队列消费**：Kafka 消息重复投递，消费者重复处理
- **超时重发**：客户端判定超时后重新发起请求，但原请求实际已处理完成
- **微服务间调用**：服务 A 调用服务 B，B 处理成功但 A 未收到响应，A 再次调用

### 关键配置参数一览表

| 参数名 | 默认值 | 位置 | 说明 |
|--------|--------|------|------|
| `lockTTL` | `10 * time.Second` | `pkg/interceptor/idempotence.go` | 分布式锁过期时间，防止死锁 |
| `resultTTL` | `24 * time.Hour` | `pkg/interceptor/idempotence.go` | 结果缓存过期时间 |
| `TKey` | `"easy-chat-idempotence-task-id"` | `pkg/interceptor/idempotence.go` | context 中存储 UUID 的 Key |
| `DKey` | `"easy-chat-idempotence-dispatch-key"` | `pkg/interceptor/idempotence.go` | gRPC metadata 中传输 ID 的 Key |
| `defaultIdempotentMethods` | `{"/social.social/GroupCreate": true}` | `pkg/interceptor/idempotence.go` | 默认启用幂等性的 gRPC 方法列表 |

---

> 本文档基于 EasyChat 项目 `pkg/interceptor/idempotence.go` 及相关服务启动文件的当前代码状态编写，旨在为项目开发者提供幂等性实现的技术参考，并指导后续的优化和扩展工作。
