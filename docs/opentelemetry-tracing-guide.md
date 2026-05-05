# go-zero 微服务链路追踪实践指南

## 1. Telemetry 配置参数说明

go-zero 通过 `Telemetry` 配置块接入 OpenTelemetry 链路追踪，完整参数如下：

| 参数 | 类型 | 说明 | 示例值 | 必需 |
|------|------|------|--------|------|
| Name | string | 服务名称，在 Jaeger UI 中显示 | `user-api` | 否 |
| Endpoint | string | OTLP 导出端点地址 | `localhost:4317` 或 `http://jaeger:4318` | 是 |
| Sampler | float64 | 采样率，0.0-1.0，默认 1.0 | `1.0`（100%采样） | 否 |
| Batcher | string | 导出协议：`otlpgrpc`、`otlphttp`、`zipkin`、`file` | `otlpgrpc` | 否 |
| OtlpHeaders | map[string]string | 自定义 HTTP 头（用于认证等） | `{"uptrace-dsn": "..."}` | 否 |
| OtlpHttpPath | string | OTLP HTTP 导出路径 | `/v1/traces` | 否 |
| OtlpHttpSecure | bool | 是否启用 TLS | `false` | 否 |
| Disabled | bool | 完全禁用链路追踪 | `false` | 否 |

### 配置注意事项

- **`Batcher: otlpgrpc`** 时，Endpoint 为 `host:port` 格式（无协议前缀），如 `localhost:4317`
- **`Batcher: otlphttp`** 时，Endpoint 应为完整 URL，如 `http://localhost:4318`
- 旧的 `jaeger` 导出协议在 go-zero v1.10+ 中已弃用，应迁移到 `otlpgrpc`

### 项目配置示例

以下为本项目中 user-api 和 user-rpc 的实际配置：

```yaml
# user-api
Telemetry:
  Name: user-api
  Endpoint: host.docker.internal:4317
  Sampler: 1.0
  Batcher: otlpgrpc

# user-rpc
Telemetry:
  Name: user-rpc
  Endpoint: 127.0.0.1:4317
  Sampler: 1.0
  Batcher: otlpgrpc
```

---

## 2. 跨服务链路传播机制（Context Propagation）

### W3C TraceContext 标准

go-zero 采用 [W3C TraceContext](https://www.w3.org/TR/trace-context/) 标准进行跨服务链路传播，核心载体为以下两个 HTTP header：

| Header | 必需 | 说明 |
|--------|------|------|
| `traceparent` | 是 | 携带 TraceID、SpanID 和采样标记 |
| `tracestate` | 否 | 厂商自定义扩展信息 |

### traceparent 格式

```
traceparent: 00-{TraceID}-{SpanID}-{采样标记}
```

示例：

```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
```

各字段说明：

| 字段 | 长度 | 说明 |
|------|------|------|
| version | 2 hex | 版本号，当前固定为 `00` |
| trace-id | 32 hex | 全局唯一的 TraceID |
| parent-id | 16 hex | 当前 Span 的父 SpanID |
| trace-flags | 2 hex | 采样标记，`01` 表示采样，`00` 表示不采样 |

### go-zero 自动传播路径

go-zero 的 `TracingInterceptor` 自动处理链路上下文的注入与提取，**无需任何代码改动**：

```
HTTP 请求 → HTTP 响应     （traceparent header 自动注入/提取）
API 服务 → RPC 调用       （gRPC metadata 自动注入/提取）
RPC 服务 → RPC 调用       （gRPC metadata 自动注入/提取）
```

完整传播链路示例：

```
用户请求 → [user-api] --gRPC metadata--> [user-rpc] --gRPC metadata--> [其他RPC]
               │                              │
          HTTP traceparent            gRPC traceparent
          header 注入                 metadata 注入
```

---

## 3. 自定义 Span

在 go-zero 自动埋点的基础上，业务代码中可以创建自定义 Span 来记录更细粒度的操作。

### 创建自定义 Span

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

func ProcessOrder(ctx context.Context, orderID string) error {
    tracer := otel.Tracer("order-service")
    ctx, span := tracer.Start(ctx, "ProcessOrder",
        trace.WithAttributes(
            attribute.String("order.id", orderID),
        ),
    )
    defer span.End()

    // 业务逻辑...

    return nil
}
```

### 添加属性

```go
span.SetAttributes(
    attribute.String("order.customer", "张三"),
    attribute.Int("order.items_count", 3),
    attribute.Float64("order.total", 299.99),
)
```

### 记录错误

```go
func ProcessOrder(ctx context.Context, orderID string) error {
    tracer := otel.Tracer("order-service")
    ctx, span := tracer.Start(ctx, "ProcessOrder")
    defer span.End()

    order, err := getOrder(orderID)
    if err != nil {
        // 记录错误事件
        span.RecordError(err)
        // 设置 Span 状态为错误
        span.SetStatus(codes.Error, err.Error())
        // 添加错误类型属性，方便筛选
        span.SetAttributes(attribute.String("error.type", "OrderNotFound"))
        return err
    }

    span.SetStatus(codes.Ok, "")
    return nil
}
```

### 在已有 Span 上追加属性

当不需要创建新 Span，只需在当前 Span 上追加信息时：

```go
func ValidateUser(ctx context.Context, userID string) error {
    // 从 context 中获取当前 Span
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(
        attribute.String("user.id", userID),
        attribute.String("user.validation_source", "jwt"),
    )

    // 业务逻辑...
    return nil
}
```

### IsRecording() 性能优化

当采样率较低时，大多数 Span 不会被记录。使用 `IsRecording()` 可以避免不必要的属性计算开销：

```go
span := trace.SpanFromContext(ctx)

if span.IsRecording() {
    // 仅在 Span 被采样记录时，才执行昂贵的属性计算
    span.SetAttributes(
        attribute.String("request.body", serializeRequestBody(req)),
        attribute.String("user.permissions", getPermissionsString(userID)),
    )
}
```

> **注意**：`span.SetAttributes()` 本身开销很小，但属性值的计算（如 JSON 序列化、数据库查询）可能较昂贵。`IsRecording()` 的意义在于跳过值计算，而非跳过 `SetAttributes` 调用。

---

## 4. 采样策略选择

### 推荐采样率

| 环境 | 推荐采样率 | 原因 |
|------|----------|------|
| 开发环境 | 1.0（100%） | 完整调试信息，快速定位问题 |
| 测试/预发布 | 0.5（50%） | 平衡成本和可见性 |
| 生产（低流量 <1k RPS） | 0.1-0.2 | 低流量可承受较高采样率 |
| 生产（高流量 >10k RPS） | 0.01-0.05 | 控制存储成本，避免性能影响 |
| 关键路径（支付等） | 1.0 | 不能丢失任何请求的追踪数据 |

### 分层采样策略

在实际生产中，建议采用分层采样策略：

```
                    ┌─────────────────────┐
                    │   全局基础采样率      │
                    │   Sampler: 0.1       │
                    └─────────┬───────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
     ┌────────▼──────┐ ┌─────▼──────┐ ┌──────▼─────────┐
     │ 关键路径强制采样 │ │ 常规路径    │ │ 健康检查不采样  │
     │ 支付/订单 1.0  │ │ 0.1 采样   │ │ /health 0.0   │
     └───────────────┘ └────────────┘ └────────────────┘
```

- **关键路径**（支付、下单）：通过代码中强制创建 Span 实现 100% 采样
- **常规路径**：使用全局配置的采样率
- **健康检查等**：可通过 go-zero 中间件跳过，避免无意义的数据

配置示例：

```yaml
# 生产环境常规配置
Telemetry:
  Name: user-api
  Endpoint: otel-collector:4317
  Sampler: 0.1
  Batcher: otlpgrpc
```

---

## 5. 最佳实践

### 属性命名规范

使用 `entity.sub_entity.property` 三段式命名，保持全局一致性：

```go
// 推荐 ✅
span.SetAttributes(
    attribute.String("user.profile.nickname", nickname),
    attribute.Int("order.payment.amount", amount),
    attribute.String("chat.message.content_type", "text"),
)

// 不推荐 ❌ 命名模糊，难以检索
span.SetAttributes(
    attribute.String("nickname", nickname),
    attribute.Int("amount", amount),
    attribute.String("type", "text"),
)
```

### 错误标记规范

错误标记应同时使用 `RecordError` + `SetStatus` + 错误类型属性，确保 Jaeger UI 和告警系统都能正确识别：

```go
// 推荐 ✅ 完整的错误标记
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    span.SetAttributes(attribute.String("error.type", "DatabaseTimeout"))
    return err
}

// 不推荐 ❌ 仅记录错误，缺少状态和类型
if err != nil {
    span.RecordError(err)
    return err
}
```

### Span 命名规范

| 分类 | 好的命名 ✅ | 坏的命名 ❌ |
|------|-----------|-----------|
| HTTP Handler | `GET /api/user/login` | `handleLogin` |
| gRPC 方法 | `user.UserService.Login` | `Login` |
| 业务操作 | `OrderService.CreatePayment` | `doStuff` |
| 数据访问 | `MySQL.QueryUserByID` | `sql` |
| 缓存操作 | `Redis.GetUserInfo` | `cache` |

### 性能优化

```go
// 推荐 ✅ 使用 IsRecording() 避免昂贵计算
span := trace.SpanFromContext(ctx)
if span.IsRecording() {
    span.SetAttributes(attribute.String("request.detail", expensiveSerialize(req)))
}

// 不推荐 ❌ 无论是否采样都执行昂贵操作
span.SetAttributes(attribute.String("request.detail", expensiveSerialize(req)))
```

---

## 6. go-zero 自动埋点机制

go-zero 通过内置的 `TracingInterceptor` 自动为以下层级创建 Span，无需手动埋点：

| 层级 | 自动捕获的信息 |
|------|--------------|
| HTTP 入站 | URL、方法、状态码、响应时间 |
| gRPC 出站 | 服务名、方法名、错误码、延迟 |
| SQL 查询 | 查询语句、执行时间 |
| Redis 命令 | 命令名称、键前缀、执行时间 |

### TracingInterceptor 自动注册机制

go-zero 在服务启动时自动注册追踪拦截器：

```
┌──────────────────────────────────────────────────┐
│                   go-zero 服务启动                │
│                                                  │
│  1. 解析 Telemetry 配置                           │
│  2. 创建 OTel TracerProvider                     │
│  3. 注册为全局 TracerProvider                     │
│  4. HTTP 服务 → 自动添加 tracing 中间件           │
│  5. gRPC 服务 → 自动添加 TracingInterceptor       │
│  6. gRPC 客户端 → 自动添加 TracingInterceptor     │
│  7. SQL/Redis → sqlx/sqlx 自动埋点               │
└──────────────────────────────────────────────────┘
```

### 自动生成的 Span 层级关系

一次典型的跨服务调用，自动生成的 Span 层级如下：

```
[Trace] user-api: GET /api/user/login
  ├── user-api: HTTP Server Handler
  │     ├── user-api: Redis.Get user:token:xxx
  │     └── user-api: gRPC Client Call user.UserService.Login
  │           └── user-rpc: gRPC Server Handler
  │                 ├── user-rpc: MySQL.Query SELECT * FROM user WHERE ...
  │                 └── user-rpc: Redis.Set user:info:xxx
```

---

## 7. 链路追踪与日志关联

### go-zero 自动注入 trace_id/span_id

go-zero 日志模块自动将当前请求的 `trace_id` 和 `span_id` 注入日志条目，无需手动操作：

```json
{
  "@timestamp": "2024-01-15T10:30:00.000Z",
  "level": "info",
  "content": "user login success",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7"
}
```

### 手动获取 TraceID

在需要将 TraceID 传递给前端或第三方系统时，可手动获取：

```go
import (
    "go.opentelemetry.io/otel/trace"
)

func GetTraceID(ctx context.Context) string {
    spanCtx := trace.SpanContextFromContext(ctx)
    if spanCtx.HasTraceID() {
        return spanCtx.TraceID().String()
    }
    return ""
}

func GetSpanID(ctx context.Context) string {
    spanCtx := trace.SpanContextFromContext(ctx)
    if spanCtx.HasSpanID() {
        return spanCtx.SpanID().String()
    }
    return ""
}
```

使用示例——在错误响应中返回 TraceID：

```go
func (l *LoginLogic) Login(req *types.LoginReq) (*types.LoginResp, error) {
    // ... 业务逻辑 ...
    if err != nil {
        traceID := GetTraceID(l.ctx)
        return nil, fmt.Errorf("login failed, traceID: %s", traceID)
    }
    // ...
}
```

### Jaeger UI 与日志系统关联查询

通过 TraceID 可以在 Jaeger UI 与日志系统（如 ELK）之间建立关联查询：

```
1. 用户报告错误 → 前端显示 traceID
2. 在 Jaeger UI 搜索 traceID → 查看完整调用链路和耗时
3. 在 Kibana/日志系统搜索 traceID → 查看该请求的所有日志
4. 交叉比对 → 快速定位问题根因
```

ELK 日志查询示例（Kibana）：

```
trace_id: "4bf92f3577b34da6a3ce929d0e0e4736"
```

---

## 8. 常见问题：Redis CLIENT MAINT_NOTIFICATIONS 错误

### 问题描述

在启用链路追踪后，Jaeger 中可能出现如下 Redis 异常事件：

```
event: exception
exception.message: ERR Unknown subcommand or wrong number of arguments for 'maint_notifications'. Try CLIENT HELP
exception.type: github.com/redis/go-redis/v9/internal/proto.RedisError
```

### 根本原因

`go-redis v9.15.0+` 引入了 Client-side Caching 功能，在连接初始化阶段会发送 `CLIENT MAINT_NOTIFICATIONS` 命令。该命令**仅 Redis 7.2+ 支持**。当 Redis 版本低于 7.2 时，Redis 服务端无法识别此命令，返回错误。

由于 go-zero 自动为 Redis 操作创建了 Span，该错误会被链路追踪捕获并记录为 exception 事件。

### 影响评估

| 维度 | 评估 |
|------|------|
| 业务功能 | 无影响，Client-side Caching 是可选功能，初始化失败后自动降级 |
| 缓存功能 | 正常，所有常规 Redis 命令不受影响 |
| 性能 | 无影响，仅在连接初始化时触发一次 |
| 链路追踪 | 会显示 exception 事件，可能干扰异常筛选 |

### 解决方案

#### 方案一：升级 Redis 到 7.2+（推荐）

升级 Redis 至 7.2 或更高版本，原生支持 `CLIENT MAINT_NOTIFICATIONS` 命令。

#### 方案二：保持现状忽略

该错误对业务无任何影响，可直接忽略。如需在 Jaeger 中过滤此类噪声，可在查询时排除 `error.type = RedisError` 且 `exception.message` 包含 `maint_notifications` 的 Span。
