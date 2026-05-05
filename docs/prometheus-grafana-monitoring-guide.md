# go-zero 微服务 Prometheus + Grafana 监控接入指南

> 适用版本：go-zero v1.10.1 | Prometheus v2.53.0 | Grafana 11.2.0 | Pushgateway v1.9.0

---

## 1. 核心概念：端口角色区分

在接入监控之前，必须先厘清四种端口的职责差异。这是最容易混淆的地方。

### 1.1 端口角色对比表

| 端口 | 所属进程 | 作用 | 谁访问谁 |
|------|---------|------|---------|
| **9090** | Prometheus 容器 | Prometheus 自身的 Web UI 和查询 API | 用户浏览器访问 |
| **9102 ~ 9109** | go-zero 服务（宿主机） | 每个微服务暴露的 `/metrics` 指标数据端点 | Prometheus 主动来抓取 |
| **3000** | Grafana 容器 | 可视化监控面板 | 用户浏览器访问 |
| **9091** | Pushgateway 容器 | 短生命周期任务推送指标（可选） | 批处理任务主动推送 |

### 1.2 数据流图解

```
┌─────────────────────────────────────────────────────────────────────┐
│                           宿主机 (Windows)                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌───────────┐  │
│  │ user-api    │  │ user-rpc    │  │ im-api      │  │ ...       │  │
│  │ :9102/met   │  │ :9103/met   │  │ :9104/met   │  │ :910x/met │  │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬─────┘  │
│         │                │                │               │        │
│         └────────────────┴────────────────┴───────────────┘        │
│                                   ↑                                │
│                                   │ 每 15s 主动抓取                   │
│         ┌─────────────────────────┘                                │
│         ↓                                                          │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Docker 容器组 (monitor-net)                │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐  │   │
│  │  │ Prometheus  │───→│  Grafana    │    │  Pushgateway    │  │   │
│  │  │ :9090       │    │ :3000       │    │ :9091           │  │   │
│  │  └─────────────┘    └─────────────┘    └─────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.3 关键理解

- **Prometheus 的 9090 端口**是它自己的服务端口，用于你打开 UI 查看状态和写 PromQL。它不是用来给 go-zero 服务连接的。
- **go-zero 的 9102~9109 端口**是每个微服务各自独立开放的「数据出口」。Prometheus 像「爬虫」一样，每隔一段时间主动上门来收集这些端口的数据。
- 因为所有 go-zero 服务都跑在同一台宿主机上，所以每个服务的 `metrics` 端口必须互不相同，否则会端口冲突。
- **Grafana 的 3000 端口**是独立的面板服务，它不直接访问 go-zero，而是向 Prometheus（`http://prometheus:9090`，通过 Docker 网络）查询已采集的数据。

---

## 2. 第一步：go-zero 服务配置

### 2.1 配置说明

go-zero 已内置 Prometheus 支持，无需引入任何第三方包，也无需修改代码。只需在各自的 `.yaml` 配置文件中添加 `Prometheus` 块即可：

```yaml
Prometheus:
  Host: 0.0.0.0     # 绑定地址，0.0.0.0 允许外部访问（Prometheus 容器需要能连上）
  Port: 9102        # metrics 端点端口，每个服务必须不同
  Path: /metrics    # 端点路径，固定为 /metrics
```

配置后，服务启动时会自动在 `http://0.0.0.0:<Port>/metrics` 暴露指标数据。

### 2.2 端口分配表

| 服务 | 配置文件路径 | 业务端口 | Metrics 端口 | 配置状态 |
|------|------------|--------|-------------|---------|
| user-api | `apps/user/api/etc/user.yaml` | 8888 | **9102** | 已配置 |
| user-rpc | `apps/user/rpc/etc/user.yaml` | 8090 | 9103 | 待添加 |
| im-api | `apps/im/api/etc/im.yaml` | 8899 | 9104 | 待添加 |
| im-rpc | `apps/im/rpc/etc/im.yaml` | 8092 | 9105 | 待添加 |
| im-ws | `apps/im/ws/etc/im.yaml` | 8093 | 9106 | 待添加 |
| social-api | `apps/social/api/etc/social.yaml` | 8889 | 9107 | 待添加 |
| social-rpc | `apps/social/rpc/etc/social.yaml` | 8091 | 9108 | 待添加 |
| task-mq | `apps/task/mq/etc/task.yaml` | 8094 | 9109 | 待添加 |

### 2.3 各服务添加配置示例

#### user-api（已有，供参考）

文件：`apps/user/api/etc/user.yaml`

```yaml
Name: user
Host: 0.0.0.0
Port: 8888
Mode: dev

Prometheus:
  Host: 0.0.0.0
  Port: 9102
  Path: /metrics
```

#### user-rpc

文件：`apps/user/rpc/etc/user.yaml`

在文件任意位置（建议放在顶部 `ListenOn` 下方）添加：

```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9103
  Path: /metrics
```

#### im-api

文件：`apps/im/api/etc/im.yaml`

```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9104
  Path: /metrics
```

#### im-rpc

文件：`apps/im/rpc/etc/im.yaml`

```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9105
  Path: /metrics
```

#### im-ws

文件：`apps/im/ws/etc/im.yaml`

```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9106
  Path: /metrics
```

#### social-api

文件：`apps/social/api/etc/social.yaml`

```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9107
  Path: /metrics
```

#### social-rpc

文件：`apps/social/rpc/etc/social.yaml`

```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9108
  Path: /metrics
```

#### task-mq

文件：`apps/task/mq/etc/task.yaml`

```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9109
  Path: /metrics
```

> 配置完成后，**重启对应服务**使配置生效。

---

## 3. 第二步：配置 Prometheus 抓取目标

### 3.1 prometheus.yml 配置

用户的 Prometheus 容器已将 `./prometheus` 目录挂载到容器内的 `/etc/prometheus`。因此，在宿主机项目根目录下创建或编辑：

```
./prometheus/prometheus.yml
```

由于 go-zero 服务运行在**宿主机本地**，而 Prometheus 运行在 **Docker 容器**中，容器内无法直接通过 `localhost` 访问宿主机端口。在 Windows 环境下，应使用 `host.docker.internal` 来指向宿主机。

完整的 `prometheus.yml` 如下：

```yaml
global:
  scrape_interval: 15s      # 全局抓取间隔
  evaluation_interval: 15s  # 规则评估间隔

scrape_configs:
  # 监控 Prometheus 自身
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  # user-api
  - job_name: 'user-api'
    static_configs:
      - targets: ['host.docker.internal:9102']
    metrics_path: /metrics

  # user-rpc
  - job_name: 'user-rpc'
    static_configs:
      - targets: ['host.docker.internal:9103']
    metrics_path: /metrics

  # im-api
  - job_name: 'im-api'
    static_configs:
      - targets: ['host.docker.internal:9104']
    metrics_path: /metrics

  # im-rpc
  - job_name: 'im-rpc'
    static_configs:
      - targets: ['host.docker.internal:9105']
    metrics_path: /metrics

  # im-ws
  - job_name: 'im-ws'
    static_configs:
      - targets: ['host.docker.internal:9106']
    metrics_path: /metrics

  # social-api
  - job_name: 'social-api'
    static_configs:
      - targets: ['host.docker.internal:9107']
    metrics_path: /metrics

  # social-rpc
  - job_name: 'social-rpc'
    static_configs:
      - targets: ['host.docker.internal:9108']
    metrics_path: /metrics

  # task-mq
  - job_name: 'task-mq'
    static_configs:
      - targets: ['host.docker.internal:9109']
    metrics_path: /metrics
```

保存文件后，重启 Prometheus 容器或发送热重载信号：

```bash
# 方式一：重启容器
docker restart <prometheus-container-name>

# 方式二：热重载（推荐）
docker exec <prometheus-container-name> kill -HUP 1
```

### 3.2 验证抓取是否成功

1. 打开浏览器，访问：**http://localhost:9090/targets**
2. 在 Targets 页面中，查看每个 `job_name` 对应的状态列
3. 状态为 **UP**（绿色）表示抓取成功；**DOWN**（红色）表示连接失败

如果某个 Target 显示 DOWN，按以下顺序排查：

1. 对应服务是否已启动？
2. 在宿主机浏览器访问 `http://localhost:<metrics端口>/metrics` 是否能返回数据？（如 `http://localhost:9102/metrics`）
3. 防火墙是否拦阻了该端口？

---

## 4. 第三步：Grafana 查看监控数据

### 4.1 配置数据源

1. 打开浏览器，访问：**http://localhost:3000**
2. 使用账号登录：`admin` / `admin123`
3. 左侧菜单 → **Connections** → **Data Sources** → **Add data source**
4. 选择 **Prometheus**
5. 在 URL 中填写：`http://prometheus:9090`
   > 关键点：Grafana 和 Prometheus 处于同一 Docker 网络 `monitor-net` 中，因此 Grafana 可以通过容器名 `prometheus` 直接访问，而不是 `localhost` 或 `host.docker.internal`。
6. 其余选项保持默认，点击 **Save & Test**
7. 出现绿色提示 "Data source is working" 即表示连接成功

### 4.2 导入 go-zero Dashboard

go-zero 官方提供了开箱即用的 Grafana Dashboard（ID：**19909**），可直接导入：

1. 左侧菜单 → **+** → **Import Dashboard**
2. 在 **Import via grafana.com** 输入框中填写 Dashboard ID：`19909`
3. 点击 **Load**
4. 在下一页中，**Prometheus** 数据源下拉框选择刚才配置的数据源
5. 点击 **Import**

导入成功后，即可在 Dashboard 中查看各服务的 HTTP/gRPC QPS、延迟、错误率等核心指标。

### 4.3 手动查询指标

在 Grafana 左侧菜单点击 **Explore**（指南针图标），可直接编写 PromQL 查询：

```promql
# HTTP QPS（按服务分组）
sum(rate(http_server_requests_total[5m])) by (app)

# HTTP P95 延迟
histogram_quantile(0.95, sum(rate(http_server_duration_ms_bucket[5m])) by (le, app))

# gRPC QPS（按方法分组）
sum(rate(rpc_server_requests_total[5m])) by (method)

# HTTP 5xx 错误率
sum(rate(http_server_requests_total{code=~"5.."}[5m])) / sum(rate(http_server_requests_total[5m]))

# 单服务总请求数（以 user-api 为例）
http_server_requests_total{app="user-api"}
```

---

## 5. go-zero 自动采集的指标说明

go-zero 框架内置了 `PrometheusInterceptor`，在配置 `Prometheus` 块后会自动采集以下指标，**无需任何业务代码改动**：

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `http_server_requests_total` | Counter | HTTP 请求总数（标签：`path`, `code`, `app`） |
| `http_server_duration_ms` | Histogram | HTTP 请求耗时分布（单位：毫秒） |
| `rpc_server_requests_total` | Counter | gRPC 请求总数（标签：`method`, `code`, `app`） |
| `rpc_server_duration_ms` | Histogram | gRPC 请求耗时分布（单位：毫秒） |

此外，go-zero 还会自动暴露 Go 运行时指标：

- `go_goroutines`：当前 Goroutine 数量
- `go_memstats_alloc_bytes`：已分配内存字节数
- `process_cpu_seconds_total`：进程 CPU 使用时间

---

## 6. 自定义业务指标（进阶）

如果需要在自动指标之外，追踪自定义业务数据（如登录次数、消息发送量等），可以使用 go-zero 封装的 `metric` 包。

### 6.1 使用 go-zero metric 包

```go
import "github.com/zeromicro/go-zero/core/metric"

// 定义一个带标签的计数器
var LoginTotal = metric.NewCounterVec(&metric.CounterVecOpts{
    Namespace: "user",      // 命名空间
    Subsystem: "service",   // 子系统
    Name:      "login_total", // 指标名
    Help:      "Total login attempts", // 帮助说明
    Labels:    []string{"status"},     // 标签名
})

// 在业务代码中使用
func (l *LoginLogic) Login(req *types.LoginReq) (*types.LoginResp, error) {
    // ... 登录逻辑 ...
    if err != nil {
        LoginTotal.WithLabelValues("failed").Inc()
        return nil, err
    }
    LoginTotal.WithLabelValues("success").Inc()
    return resp, nil
}
```

自定义指标会自动注册到服务的 `/metrics` 端点，Prometheus 抓取后可在 Grafana 中查询：

```promql
# 查询登录成功次数
user_service_login_total{status="success"}

# 查询登录失败次数
user_service_login_total{status="failed"}
```

### 6.2 三种指标类型

| 类型 | 特点 | 适用场景 |
|------|------|---------|
| **Counter** | 只增不减 | 请求总数、错误次数、登录次数 |
| **Gauge** | 可增可减 | 当前在线用户数、队列长度、连接数 |
| **Histogram** | 分布统计 | 请求延迟、响应大小、自定义耗时 |

go-zero `metric` 包对应 API：

- `metric.NewCounterVec()` — Counter
- `metric.NewGaugeVec()` — Gauge
- `metric.NewHistogramVec()` — Histogram

---

## 7. Pushgateway 使用场景（可选）

Pushgateway（端口 9091）适用于**短生命周期、无法被 Prometheus 长期抓取**的任务，典型场景包括：

- 定时批处理脚本（如每日数据统计）
- 一次性任务（如数据迁移）
- 无固定监听端点的进程

**不适合的场景**：

- 常驻服务（如 user-api、im-rpc 等）：这些服务已有 `/metrics` 端点，应让 Prometheus 主动拉取（Pull 模式），而不是通过 Pushgateway 推送。Pushgateway 会成为单点且数据不会自动清理，容易造成指标残留和内存膨胀。

本项目中的 `task-mq` 虽然是 Kafka 消费者，但属于**常驻进程**，因此仍推荐使用标准 Pull 模式（配置 `Prometheus` 块暴露 9109 端口），而非 Pushgateway。

---

## 8. 验证流程清单

按以下步骤逐项验证，确保监控链路完全打通：

| 步骤 | 操作 | 预期结果 |
|------|------|---------|
| 1 | 启动所有 go-zero 服务 | 各服务无报错正常启动 |
| 2 | 浏览器访问 `http://localhost:9102/metrics` | 返回大量以 `# HELP`、`# TYPE` 开头的指标文本 |
| 3 | 依次验证 `9103` ~ `9109` 各端口 | 均能看到对应服务的 metrics 数据 |
| 4 | 访问 `http://localhost:9090/targets` | 所有 Target 状态为 **UP** |
| 5 | 登录 Grafana `http://localhost:3000` | 能正常登录 |
| 6 | 添加 Prometheus 数据源并 Test | 显示 "Data source is working" |
| 7 | 导入 Dashboard ID `19909` | 能看到各服务面板和图表 |
| 8 | 对 user-api 等服务发送一些请求 | Dashboard 中 QPS、延迟等指标从 0 开始变化 |

---

## 9. Windows 宿主机混合部署注意事项

当 go-zero 服务运行在 Windows 宿主机上，而 Prometheus/Grafana 运行在 Docker 容器中时，需要注意以下配置要点：

### 问题：`host.docker.internal` 无法解析

**现象**：Prometheus Targets 页面显示错误：
```
Get "http://host.docker.internal:9102/metrics": dial tcp: lookup host.docker.internal on 127.0.0.11:53: no such host
```

**原因**：Docker 容器内默认不一定能解析 `host.docker.internal` 这个 DNS 名称，即使在 Windows Docker Desktop 环境下也可能失败。

### 解决方案

**方案1（推荐）：在 docker-compose.yml 中添加 `extra_hosts`**

```yaml
services:
  prometheus:
    image: prom/prometheus:v2.53.0
    container_name: prometheus
    extra_hosts:
      - "host.docker.internal:host-gateway"  # 关键配置：映射到宿主机
    volumes:
      - ./prometheus:/etc/prometheus
      - prometheus_data:/prometheus
    # ... 其余配置不变
```

添加后重启容器：
```bash
docker-compose restart prometheus
```

**方案2：使用宿主机实际 IP**

在 `prometheus.yml` 中直接使用宿主机局域网 IP：
```yaml
- job_name: 'user-api'
  static_configs:
    - targets: ['192.168.x.x:9102']   # 替换为实际 IP
```

查看本机 IP（PowerShell）：
```powershell
ipconfig
```

> ⚠️ 注意：使用固定 IP 方式在 DHCP 环境下 IP 可能变化，建议优先使用方案1。

### 不同部署场景的 targets 配置对照表

| 部署场景 | targets 配置 | 说明 |
|---------|-------------|------|
| 服务在宿主机，Prometheus 在 Docker | `host.docker.internal:端口` | 需配置 extra_hosts |
| 服务和 Prometheus 都在同一 Docker 网络 | `容器名:端口` | 如 `user-api:9102` |
| 服务和 Prometheus 都在宿主机 | `localhost:端口` | 如 `localhost:9102` |
| 服务在远程服务器 | `服务器IP:端口` | 确保网络可达和端口开放 |

### 验证 metrics 端点是否正常

在配置 Prometheus 抓取之前，先在本机浏览器验证服务的 metrics 端点是否可访问：

```
http://localhost:9102/metrics    # user-api
http://localhost:9103/metrics    # user-rpc
```

如果能看到类似以下格式的输出，说明服务 metrics 端点正常：
```
# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 2.8e-05
...
```

### Windows 防火墙注意

如果 Prometheus 容器仍无法抓取到数据，检查 Windows 防火墙是否放行了 metrics 端口（9102-9109）：

```powershell
# 查看防火墙规则（PowerShell 管理员）
netsh advfirewall firewall show rule name=all | findstr "9102"

# 添加入站规则放行 metrics 端口
netsh advfirewall firewall add rule name="Prometheus Metrics" dir=in action=allow protocol=TCP localport=9102-9109
```

---

## 10. 常见问题排查

| 问题现象 | 根本原因 | 解决方案 |
|---------|---------|---------|
| Target 显示 **DOWN** | 服务未启动、metrics 端口未监听、或端口不可达 | 确认服务已启动；在宿主机访问 `http://localhost:<port>/metrics` 验证 |
| Grafana **无数据** | 数据源 URL 配置错误 | 确认填写的是 `http://prometheus:9090`（容器名），不是 `localhost` |
| 指标值**始终为 0** | 服务没有接收到请求流量 | 使用 curl/Postman/前端页面触发一些接口调用 |
| `host.docker.internal` **不通** | 旧版 Docker 不支持该域名 | 改用宿主机实际局域网 IP（如 `192.168.x.x:9102`） |
| 修改 `prometheus.yml` 后未生效 | Prometheus 未重新加载配置 | 执行 `docker exec <容器名> kill -HUP 1` 或重启容器 |
| 多个服务 metrics 端口冲突 | 两个服务配置了相同的 `Port` | 按本文第 2.2 节端口分配表检查并修正 |
| 防火墙拦截 | Windows Defender 或第三方防火墙阻止端口 | 在防火墙设置中放行 `9102` ~ `9109` 端口 |

---

## 附录：docker-compose 监控服务参考片段

如果需要在项目的 `docker-compose.yml` 中补充监控栈，可参考以下配置（与用户现有环境对齐）：

```yaml
networks:
  monitor-net:
    driver: bridge

services:
  prometheus:
    image: prom/prometheus:v2.53.0
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus:/etc/prometheus
    networks:
      - monitor-net

  grafana:
    image: grafana/grafana:11.2.0
    container_name: grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin123
    networks:
      - monitor-net

  pushgateway:
    image: prom/pushgateway:v1.9.0
    container_name: pushgateway
    ports:
      - "9091:9091"
    networks:
      - monitor-net
```
