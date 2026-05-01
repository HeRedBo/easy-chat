# EasyChat gRPC 重试机制配置实践

> 基于 go-zero + gRPC-Go 的 RPC 客户端重试方案，解决配置分散、负载均衡被覆盖、生产参数不规范等问题。

---

## 一、背景与核心问题

在早期的 `ServiceContext` 实现中，重试策略以硬编码 JSON 字符串的形式散落在各 API 服务的 `svc/servicecontext.go` 中：

```go
var retryPolicy = `{
  "methodConfig": [{
    "name": [{"service": "user.User"}],
    "retryPolicy": { "maxAttempts": 3, ... }
  }]
}`
```

这种方式带来了三个层面的问题：

| 层面 | 具体问题 |
|------|---------|
| **工程规范** | 配置与代码耦合，无法按环境（dev/staging/prod）差异化调整 |
| **负载均衡** | `grpc.WithDefaultServiceConfig(retryPolicy)` 会**完整覆盖** go-zero 注入的 `p2c` 负载均衡策略，导致所有 RPC 调用回退到 `pick_first` |
| **参数合理性** | 不同服务的退避时间、重试次数、可重试状态码缺乏统一标准，甚至出现 `0.001s` 退避、`UNKNOWN` 状态码重试等危险配置 |

---

## 二、gRPC 重试机制的底层原理

### 2.1 两种重试类型

gRPC-Go 客户端存在两种重试机制，二者独立工作：

#### ① 透明重试（Transparent Retries）

- **触发条件**：RPC 请求在发送到网络层之前失败（如连接断开），或服务端未处理完请求就返回错误
- **行为**：gRPC 内部自动重试一次
- **次数限制**：**不占用** `MaxAttempts` 配额
- **配置**：无需任何配置，始终启用（除非显式调用 `grpc.WithDisableRetry()`）

#### ② ServiceConfig 重试（配置化重试）

- **触发条件**：RPC 调用返回的状态码匹配 `RetryableStatusCodes`
- **行为**：按配置的退避策略等待后重新发起请求
- **次数限制**：`MaxAttempts` **包含首次调用**（即 `MaxAttempts=3` = 1 次原始 + 2 次重试）
- **配置**：通过 `grpc.WithDefaultServiceConfig(json)` 或 Name Resolver 返回的 ServiceConfig 注入

### 2.2 配置重试的完整执行流程

```
1. 建立 grpc.ClientConn
   └── 解析 ServiceConfig JSON → 生成 methodConfig 路由表

2. 发起 RPC 调用（如 /user.User/Login）
   └── 匹配 methodConfig.name（按 service / method 通配）

3. 调用失败
   └── 检查 status code 是否在 RetryableStatusCodes 中
       └── 否 → 直接返回错误
       └── 是 → 计算退避时间

4. 退避计算
   backoff = min(initialBackoff × multiplier^(attempt-1), maxBackoff)

5. 等待 → 重新发起请求（重复 3~5，直到成功或达到 MaxAttempts）
```

### 2.3 重试节流（Retry Throttling）

gRPC 内置了令牌桶机制防止重试风暴压垮脆弱的服务端：

```json
{
  "retryThrottling": {
    "maxTokens": 10,
    "tokenRatio": 0.1
  }
}
```

- 每次失败请求消耗 1 个 token
- 每次成功请求返还 `tokenRatio` 个 token
- token 不足时，即使配置了重试也会直接失败

> 当前项目实现中暂未启用重试节流，可根据压测结果后续补充。

---

## 三、go-zero 与重试的关系

**重要澄清：go-zero zrpc 客户端本身没有重试拦截器。**

用户代码中使用的重试机制，本质上是 **gRPC-Go 原生的 ServiceConfig 重试**（gRPC-Go v1.40+ 支持），而非 go-zero 框架提供。

go-zero 在客户端侧实际提供的拦截器只有以下 5 个：

| 拦截器 | 作用 | 与重试的关系 |
|--------|------|-------------|
| `TracingInterceptor` | 链路追踪（OpenTelemetry/Jaeger） | 无关 |
| `DurationInterceptor` | 请求耗时统计 | 无关 |
| `PrometheusInterceptor` | 指标采集（QPS/耗时/错误率） | 无关 |
| `BreakerInterceptor` | **熔断**（Google SRE 自适应算法） | **熔断是快速失败，不是重试** |
| `TimeoutInterceptor` | 超时控制（基于 context.WithTimeout） | 控制单次 RPC 的超时时间 |

**熔断与重试的协作关系：**

- 重试解决的是「偶发性故障」——通过多次尝试提高成功率
- 熔断解决的是「持续性故障」——当错误率达到阈值时，后续请求直接拒绝，避免拖垮下游
- 二者是**正交**的保护机制，生产环境应该**同时启用**

---

## 四、项目实现方案

### 4.1 设计目标

1. **配置集中化**：重试策略由 `etc/*.yaml` 统一管理，支持按环境差异化
2. **代码零侵入**：`svc` 层只负责「选择策略」，不负责「定义策略」
3. **负载均衡安全**：任何重试配置都不允许覆盖 go-zero 的 `p2c` 负载均衡
4. **参数标准化**：统一退避时间、重试次数、可重试状态码的生产级标准

### 4.2 核心实现：`pkg/zrpcx/retry.go`

```go
package zrpcx

import (
	"encoding/json"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

// RetryPolicy 重试策略配置
type RetryPolicy struct {
	MaxAttempts          int      `json:"MaxAttempts"`
	InitialBackoff       string   `json:"InitialBackoff"`
	MaxBackoff           string   `json:"MaxBackoff"`
	BackoffMultiplier    float64  `json:"BackoffMultiplier"`
	RetryableStatusCodes []string `json:"RetryableStatusCodes"`
}

// BuildGlobalRetryOption 将所有服务的重试策略合并为单个 ClientOption
// 关键：显式保留 "loadBalancingPolicy": "p2c"，防止覆盖 go-zero 的负载均衡配置
func BuildGlobalRetryOption(policies map[string]RetryPolicy) zrpc.ClientOption {
	cfg := serviceConfig{
		LoadBalancingPolicy: "p2c",
	}
	if len(policies) > 0 {
		// 构建 methodConfig 数组...
	}
	raw, _ := json.Marshal(cfg)
	return zrpc.WithDialOption(grpc.WithDefaultServiceConfig(string(raw)))
}
```

**关键设计决策：**

| 决策 | 说明 |
|------|------|
| 合并为单个 Option | 利用 gRPC `methodConfig[].name` 匹配机制，一个 ServiceConfig JSON 可覆盖多个服务的重试策略 |
| 显式保留 `p2c` | `grpc.WithDefaultServiceConfig` 是完整替换，如果不写 `loadBalancingPolicy`，go-zero 的 p2c 会被覆盖为默认的 `pick_first` |
| 空配置安全 | 当 `policies` 为空时，返回仅包含 `{"loadBalancingPolicy":"p2c"}` 的 Option，不影响原有行为 |

### 4.3 `svc` 层使用方式

**改造前（以 social/api 为例）：**

```go
var retryPolicy = `{...}` // 硬编码 JSON

func NewServiceContext(c config.Config) *ServiceContext {
    return &ServiceContext{
        User:   userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
        Social: socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc,
            zrpc.WithUnaryClientInterceptor(interceptor.DefaultIdempotentClient),
            zrpc.WithDialOption(grpc.WithDefaultServiceConfig(retryPolicy)), // 覆盖 p2c！
        )),
        Im: imclient.NewIm(zrpc.MustNewClient(c.Imrpc)),
    }
}
```

**改造后：**

```go
func NewServiceContext(c config.Config) *ServiceContext {
    retryOpt := zrpcx.BuildGlobalRetryOption(c.RpcRetry) // 只新增这一行
    return &ServiceContext{
        User:   userclient.NewUser(zrpc.MustNewClient(c.UserRpc, retryOpt)),
        Social: socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc,
            zrpc.WithUnaryClientInterceptor(interceptor.DefaultIdempotentClient),
            retryOpt, // 复用同一个 Option
        )),
        Im: imclient.NewIm(zrpc.MustNewClient(c.Imrpc, retryOpt)),
    }
}
```

---

## 五、配置说明

### 5.1 配置作用域：按 API 服务隔离

**每个 API 服务的 `etc/*.yaml` 只配置「该服务自己会调用的下游 RPC」的重试策略**，不需要聚合全项目的配置。

**示例：`apps/user/api/etc/user.yaml`**

```yaml
UserRpc:
  Etcd:
    Hosts:
      - 127.0.0.1:2379
    Key: user.rpc

RpcRetry:
  user.User:
    MaxAttempts: 3
    InitialBackoff: "100ms"
    MaxBackoff: "1s"
    BackoffMultiplier: 2.0
    RetryableStatusCodes:
      - UNAVAILABLE
      - DEADLINE_EXCEEDED
```

**示例：`apps/social/api/etc/social.yaml`**

```yaml
RpcRetry:
  user.User:
    MaxAttempts: 3
    InitialBackoff: "100ms"
    MaxBackoff: "1s"
    BackoffMultiplier: 2.0
    RetryableStatusCodes:
      - UNAVAILABLE
  social.social:
    MaxAttempts: 3
    InitialBackoff: "100ms"
    MaxBackoff: "1s"
    BackoffMultiplier: 2.0
    RetryableStatusCodes:
      - UNAVAILABLE
      - DEADLINE_EXCEEDED
  im.Im:
    MaxAttempts: 3
    InitialBackoff: "100ms"
    MaxBackoff: "1s"
    BackoffMultiplier: 2.0
    RetryableStatusCodes:
      - UNAVAILABLE
```

### 5.2 配置字段详解

| 字段 | 类型 | 说明 | 推荐值 |
|------|------|------|--------|
| `MaxAttempts` | int | 最大尝试次数（含首次调用） | `3`（1 次原始 + 2 次重试） |
| `InitialBackoff` | string | 首次重试的退避时间 | `"100ms"` |
| `MaxBackoff` | string | 退避时间的上限 | `"1s"` |
| `BackoffMultiplier` | float | 退避时间的乘数因子 | `2.0`（指数退避） |
| `RetryableStatusCodes` | []string | 触发重试的 gRPC 状态码 | `["UNAVAILABLE", "DEADLINE_EXCEEDED"]` |

### 5.3 为什么按服务隔离而不是全局一份？

不同 API 服务对同一个下游 RPC 的可靠性要求可能不同：

- `social/api` 调用 `user.User` 可能是用户资料查询，可以容忍 3 次重试
- `im/api` 调用 `user.User` 可能是在消息发送链路中，为了降低延迟只允许 2 次重试

按服务隔离后，各自独立演进，互不干扰。

---

## 六、生产级最佳实践

### 6.1 重试策略设计原则

1. **幂等性优先**
   - 只有幂等操作（查询、根据唯一键更新）才能安全重试
   - 非幂等操作（扣款、下单、发消息）如果返回 `UNKNOWN`，**绝对不要重试**——因为服务端可能已经执行成功，只是响应丢失

2. **限制重试次数**
   - 通常 2~3 次（含首次调用）
   - 过多重试会放大故障（Retry Storm），压垮已不稳定的服务端

3. **使用退避策略**
   - 固定间隔退避：简单，但无法有效分散请求峰值
   - **指数退避**（推荐）：`backoff = min(initial × multiplier^(n-1), max)`
   - 避免所有失败请求在同一时刻重试，造成「 thundering herd 」

4. **设置总超时上限**
   - 总耗时 ≈ `MaxAttempts × (单次超时 + 退避时间)`
   - 确保不超过上游（API 层）的超时时间，否则会出现「服务端还在重试，客户端已经超时断开」的浪费

5. **与熔断、限流配合**
   - 重试解决「偶发性故障」
   - 熔断解决「持续性故障」
   - 限流解决「过载保护」
   - 三者缺一不可

### 6.2 可重试状态码选择指南

| 状态码 | 是否建议重试 | 原因 |
|--------|-------------|------|
| `UNAVAILABLE` | ✅ 推荐 | 服务端临时不可用（如重启、过载），通常可恢复 |
| `DEADLINE_EXCEEDED` | ✅ 推荐 | 请求超时，可能因网络抖动，重试有机会成功 |
| `RESOURCE_EXHAUSTED` | ⚠️ 谨慎 | 服务端资源耗尽，立即重试可能加剧问题，建议配合较长退避 |
| `UNKNOWN` | ❌ 不推荐 | 服务端返回未知错误，操作**可能已经成功**，重试有重复执行风险 |
| `INTERNAL` | ❌ 不推荐 | 服务端内部错误，通常是代码 bug，重试无法解决 |
| `INVALID_ARGUMENT` | ❌ 不推荐 | 请求参数错误，重试多少次都一样 |
| `PERMISSION_DENIED` | ❌ 不推荐 | 权限不足，重试无法解决 |

### 6.3 推荐参数模板

```yaml
RpcRetry:
  your.service.Name:
    MaxAttempts: 3
    InitialBackoff: "100ms"
    MaxBackoff: "1s"
    BackoffMultiplier: 2.0
    RetryableStatusCodes:
      - UNAVAILABLE
      - DEADLINE_EXCEEDED
```

---

## 七、常见陷阱与注意事项

### 陷阱 1：覆盖 p2c 负载均衡

**问题：** `grpc.WithDefaultServiceConfig` 是完整替换，后传入的会覆盖先传入的。

go-zero 在 `zrpc/internal/client.go` 中先注入了：
```go
svcCfg := fmt.Sprintf(`{"loadBalancingPolicy":"%s"}`, p2c.Name)
```

如果用户后续又传了一个不含 `loadBalancingPolicy` 的 JSON，p2c 就会丢失，回退到默认的 `pick_first`。

**解决：** `BuildGlobalRetryOption` 始终显式保留 `"loadBalancingPolicy": "p2c"`。

### 陷阱 2：`MaxAttempts` 包含首次调用

**误解：** `MaxAttempts: 3` 表示「失败后再重试 3 次」（共 4 次调用）。

**实际：** `MaxAttempts: 3` 表示「总共最多尝试 3 次」（1 次原始 + 2 次重试）。

### 陷阱 3：退避时间过短等于没退避

**问题：** 早期配置中 `InitialBackoff: "0.001s"`（1ms）几乎起不到分散请求峰值的作用。

**建议：** 至少 50ms~100ms，生产环境通常 100ms~500ms。

### 陷阱 4：在 `UNKNOWN` 上重试非幂等操作

**场景：** 扣款 RPC 返回 `UNKNOWN`，客户端重试 → 用户被扣了两次钱。

**建议：** 只有确认是幂等操作后，才将 `UNKNOWN` 加入 `RetryableStatusCodes`。默认配置中不应包含 `UNKNOWN`。

### 陷阱 5：ServiceConfig JSON 字段大小写

gRPC-Go 对 ServiceConfig JSON 的解析相对宽松，但建议遵循其测试用例中的规范：

```json
{
  "methodConfig": [{
    "name": [{"service": "user.User"}],
    "retryPolicy": {
      "MaxAttempts": 3,
      "InitialBackoff": "100ms",
      "MaxBackoff": "1s",
      "BackoffMultiplier": 2.0,
      "RetryableStatusCodes": ["UNAVAILABLE"]
    }
  }]
}
```

注意 `MaxAttempts` 等字段使用大写开头的驼峰命名。

---

## 八、后续扩展方向

1. **重试节流（Retry Throttling）**
   - 当压测发现重试风暴风险时，可在 `BuildGlobalRetryOption` 中增加 `retryThrottling` 字段

2. **按 Method 级别精细化配置**
   - 当前实现按 `service` 级别通配，如需对某个具体 RPC 方法（如 `user.User/Login`）单独配置，可将 `nameConfig` 扩展为包含 `Method` 字段

3. **动态配置热更新**
   - 结合项目已有的 `configserver` 能力，实现不重启服务即可调整重试参数

4. **重试指标观测**
   - 在 `pkg/zrpcx` 中增加重试次数的 Prometheus 计数器，便于监控「重试率」和「重试成功率」

---

## 九、参考链接

- [gRPC-Go Retry Design Doc](https://github.com/grpc/proposal/blob/master/A6-client-retries.md)
- [gRPC Service Config Protocol](https://github.com/grpc/grpc/blob/master/doc/service_config.md)
- go-zero 源码：`zrpc/internal/client.go`（p2c 负载均衡注入点）
- go-zero 源码：`zrpc/internal/clientinterceptors/breakerinterceptor.go`（熔断拦截器）
