# 部署端口管理方案

## 📋 方案概述

采用 **镜像标签 + 配置文件** 双重保障机制，实现端口配置的灵活性和可靠性。

### 核心优势

✅ **构建与部署隔离** - 部署时无需项目代码，仅依赖镜像  
✅ **双重保障** - 标签提供默认值，配置文件支持覆盖  
✅ **环境灵活** - 不同环境可配置不同端口映射  
✅ **向后兼容** - 保留原有 port_getter 机制用于构建阶段  

---

## 🏗️ 工作流程

### 1. 构建阶段（开发环境）

```bash
# make build 自动执行
make build

# 流程：
# 1. port_getter.go 读取 yaml 配置文件 → 获取端口 8888
# 2. docker build 时添加标签
#    --label "service.port=8888"
#    --label "service.name=user"
#    --label "service.type=api"
# 3. Dockerfile 将 PORT 设为环境变量
#    ENV PORT=8888
```

### 2. 部署阶段（生产环境）

```bash
# 执行部署脚本
./deploy/script/user-api-test.sh

# 流程：
# 1. get_deploy_port.sh 从镜像标签读取端口 → 8888
# 2. 检查 deploy.conf 是否有覆盖配置
# 3. 使用最终端口启动容器
#    docker run -p 8088:8888 -e PORT=8888 ...
```

---

## 📁 文件说明

### 核心文件

| 文件 | 作用 | 阶段 |
|------|------|------|
| `deploy/make/build.mk` | 构建时添加镜像标签 | 构建 |
| `deploy/make/port_getter.go` | 从 yaml 读取端口 | 构建 |
| `deploy/dockerfile/Dockerfile_local` | 固化 PORT 环境变量 | 构建 |
| `deploy/script/get_deploy_port.sh` | 获取最终端口（标签+配置） | 部署 |
| `deploy/script/deploy.conf` | 部署端口覆盖配置 | 部署 |
| `deploy/script/*-test.sh` | 红绿部署脚本 | 部署 |

---

## 🎯 端口获取优先级

```
配置文件 (deploy.conf)
    ↓ 如果未配置
镜像标签 (service.port)
    ↓ 如果未设置
错误退出
```

### 示例场景

#### 场景 1：使用镜像标签（默认）

```bash
# 构建时标签包含 service.port=8888
# deploy.conf 中未配置 [user-api]

# 结果：使用 8888
```

#### 场景 2：配置文件覆盖

```bash
# deploy.conf 中配置：
[user-api]
external_port=9999

# 结果：使用 9999（覆盖标签的 8888）
```

#### 场景 3：不同环境不同端口

```bash
# 开发环境 - deploy-dev.conf
[user-api]
external_port=8888

# 生产环境 - deploy-prod.conf
[user-api]
external_port=9999

# 部署时指定配置文件
./deploy/script/user-api-test.sh ./deploy-prod.conf
```

---

## 🔧 配置说明

### deploy.conf 格式

```ini
# 服务配置段：[服务名-服务类型]
[user-api]
external_port=8888        # 外部访问端口
replicas=2                # 副本数（预留）
health_check=/health      # 健康检查路径（预留）

[user-rpc]
external_port=8090
replicas=3
```

### 端口映射策略

```ini
[port_strategy]
green_blue_offset=1           # 红绿部署偏移量
auto_calculate=true           # 是否自动计算
external_port_start=8080      # 外部端口范围起始
external_port_end=9090        # 外部端口范围结束
```

---

## 📝 使用示例

### 示例 1：标准部署

```bash
# 1. 构建镜像（自动添加标签）
make build SVR=user MOD=api ENV=dev

# 2. 部署（自动读取标签）
./deploy/script/user-api-test.sh
```

### 示例 2：自定义端口覆盖

```bash
# 1. 编辑 deploy.conf
[user-api]
external_port=9999

# 2. 部署（使用配置文件的端口）
./deploy/script/user-api-test.sh
```

### 示例 3：多环境部署

```bash
# 开发环境
./deploy/script/user-api-test.sh ./deploy-dev.conf

# 测试环境
./deploy/script/user-api-test.sh ./deploy-test.conf

# 生产环境
./deploy/script/user-api-test.sh ./deploy-prod.conf
```

---

## 🔍 调试技巧

### 查看镜像标签

```bash
docker inspect --format='{{json .Config.Labels}}' <image_name>:<tag>

# 输出示例：
# {
#   "service.name": "user",
#   "service.type": "api",
#   "service.port": "8888",
#   "build.time": "2026-04-27T10:30:00+0800"
# }
```

### 测试端口获取

```bash
# 单独测试端口获取脚本
./deploy/script/get_deploy_port.sh \
  crpi-xxx/user-api-dev latest \
  ./deploy/script/deploy.conf
```

### 查看容器环境变量

```bash
docker exec <container_name> env | grep PORT

# 输出：
# PORT=8888
```

---

## 🚀 扩展到新服务

### 步骤 1：确保 yaml 配置正确

```yaml
# apps/newservice/api/etc/newservice.yaml
Name: newservice
Host: 0.0.0.0
Port: 8890  # ← 确保有此配置
```

### 步骤 2：构建时自动添加标签

```bash
make build SVR=newservice MOD=api ENV=dev
# 自动读取 Port: 8890 并添加标签
```

### 步骤 3：添加到 deploy.conf

```ini
[newservice-api]
external_port=8890
replicas=2
```

### 步骤 4：创建部署脚本

```bash
# 复制模板并修改
cp deploy/script/user-api-test.sh deploy/script/newservice-api-test.sh
# 修改镜像名和容器名
```

---

## ⚠️ 注意事项

1. **标签不可变** - 镜像一旦构建，标签中的端口无法修改
2. **配置优先级** - deploy.conf 始终优先于标签
3. **端口冲突** - 确保外部端口不与其他服务冲突
4. **Go 环境** - 部署服务器需要安装 Go（用于 port_getter.go，仅构建阶段）
5. **Docker 版本** - 需要支持 label 的 Docker 版本（17.05+）

---

## 🎉 总结

| 特性 | 实现方式 |
|------|---------|
| 构建端口读取 | port_getter.go 解析 yaml |
| 端口信息携带 | Docker 镜像标签 |
| 部署端口获取 | get_deploy_port.sh 读取标签 |
| 端口覆盖机制 | deploy.conf 配置文件 |
| 健康检查 | Dockerfile ENV PORT |
| 红绿部署 | 自动计算偏移端口 |

这个方案实现了：
- ✅ 构建与部署完全隔离
- ✅ 端口配置单一来源（标签）
- ✅ 灵活的环境覆盖（配置文件）
- ✅ 向后兼容现有流程
