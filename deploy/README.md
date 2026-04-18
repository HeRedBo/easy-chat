# 🚀 EasyChat 构建部署指南

> 🐳 基于阿里云容器镜像服务的自动化构建、推送与部署方案

---

## 📑 目录

- [🌍 环境规范](#-环境规范)
- [🏷️ 镜像命名规则](#️-镜像命名规则)
- [⚠️ 重要说明](#️-重要说明)
- [⚡ 快速开始](#-快速开始)
- [📖 命令详解](#-命令详解)
- [⚙️ 配置说明](#️-配置说明)
- [🔄 构建流程](#-构建流程)
- [📦 依赖清单](#-依赖清单)

---

## 🌍 环境规范

遵循行业标准的多环境管理策略：

| 环境标识 | 名称 | 说明 |
|:--------:|------|------|
| 🟢 `dev` | 开发环境 | 日常开发联调，快速迭代 |
| 🟡 `test` | 测试环境 | 集成测试，QA 验证 |
| 🟠 `prv` | 预发布环境 | 生产数据模拟，上线前最终验证 |
| 🔴 `prod` | 生产环境 | 对外服务的正式环境 |

---

## 🏷️ 镜像命名规则

```
{服务名}-{模块名}-{环境}
```

**示例：**

| 镜像名称 | 服务 | 模块 | 环境 |
|----------|:----:|:----:|:----:|
| `user-rpc-dev` | 👤 用户服务 | RPC | 🟢 开发 |
| `user-api-dev` | 👤 用户服务 | API | 🟢 开发 |
| `task-mq-prv` | ⚙️ 任务服务 | MQ | 🟠 预发布 |
| `im-rpc-prod` | 💬 IM服务 | RPC | 🔴 生产 |
| `social-api-test` | 👥 社交服务 | API | 🟡 测试 |

---

## ⚠️ 重要说明

> [!WARNING]
> **所有命令必须在项目根目录执行！**
>
> 否则会出现路径错误、编译失败、配置文件找不到等问题。

🔑 登录阿里云镜像仓库后才能推送镜像：

```bash
docker login --username=<你的阿里云账号> crpi-yw54l3gskjfve0c3.cn-hangzhou.personal.cr.aliyuncs.com
```

---

## ⚡ 快速开始

🎯 **5 秒上手：**

```bash
# 1️⃣ 查看当前构建配置
make -f deploy/make/build.mk info

# 2️⃣ 构建镜像（默认：user-rpc-dev）
make -f deploy/make/build.mk build

# 3️⃣ 构建并推送到阿里云
make -f deploy/make/build.mk release
```

---

## 📖 命令详解

### 🔍 `info` — 查看构建配置

展示当前构建的所有配置信息，包括服务名、模块、环境、端口、镜像地址等：

```bash
# 默认配置：user rpc dev
make -f deploy/make/build.mk info
```

**输出示例：**

```
========================================
Service SVR: user
Module MOD:   rpc
Env ENV:      dev
Version:      latest
Port:         8081
Entry Point:  apps/user/rpc/user.go
Config File:  apps/user/rpc/etc/dev/user.yaml
Binary:       bin/user-rpc
Image:        crpi-xxx.../user-rpc-dev:latest
Dockerfile:   deploy/dockerfile/Dockerfile_local
========================================
```

---

### 🔨 `build` — 构建本地镜像

编译 Go 二进制文件 → 构建 Docker 镜像，一键完成：

```bash
# 🏗️ 默认构建（user-rpc-dev）
make -f deploy/make/build.mk build

# 🎯 指定服务 + 模块
make -f deploy/make/build.mk build SVR=user MOD=api

# 🌐 指定环境
make -f deploy/make/build.mk build ENV=prv

# 🚀 完整自定义
make -f deploy/make/build.mk build SVR=im MOD=ws ENV=prod
```

**参数说明：**

| 参数 | 说明 | 可选值 |
|:----:|------|--------|
| 📦 `SVR` | 服务名 | `user` / `im` / `social` / `task` |
| 🧩 `MOD` | 模块名 | `api` / `rpc` / `mq` |
| 🌍 `ENV` | 环境 | `dev` / `test` / `prv` / `prod` |
| 🏷️ `VERSION` | 版本号 | 默认 `latest` |

---

### 🚢 `release` — 构建并推送镜像

执行 `build` + `push`，将镜像推送到阿里云容器镜像仓库：

```bash
# 构建并推送 user-api 到开发环境
make -f deploy/make/build.mk release SVR=user MOD=api

# 构建并推送 task-mq 到生产环境
make -f deploy/make/build.mk release SVR=task MOD=mq ENV=prod VERSION=v1.2.0
```

---

### ☁️ `push` — 仅推送已有镜像

```bash
make -f deploy/make/build.mk push SVR=user MOD=api ENV=dev
```

---

## ⚙️ 配置说明

所有配置项位于 `deploy/make/build.mk`，按需修改：

### 🌐 1. 阿里云镜像地址

修改 `IMAGE_BASE` 为你的阿里云镜像仓库地址：

```makefile
IMAGE_BASE := crpi-yw54l3gskjfve0c3.cn-hangzhou.personal.cr.aliyuncs.com/redbo-easy-chat
```

### ✅ 2. 服务白名单

控制允许构建的服务列表，不在白名单中的服务将被拒绝：

```makefile
ALLOWED_SVRS := user im social task
```

### ✅ 3. 模块白名单

控制允许构建的模块类型：

```makefile
ALLOWED_MODS := rpc api mq
```

### ✅ 4. 环境白名单

控制允许部署的环境标识：

```makefile
ALLOWED_ENVS := dev test prv prod
```

---

## 🔄 构建流程

```
  ┌─────────────┐     ┌──────────────┐     ┌───────────────┐     ┌──────────────┐
  │  ✅ 参数校验  │────▶│  🔌 获取端口  │────▶│  🔨 编译 Go    │────▶│  🐳 Docker    │
  │   check      │     │  get-port     │     │  go build     │     │ docker build  │
  └─────────────┘     └──────────────┘     └───────────────┘     └──────┬───────┘
                                                                           │
                                                                   ┌───────▼───────┐
                                                                   │  ☁️ 推送镜像   │
                                                                   │  docker push   │
                                                                   │  (release时)   │
                                                                   └───────────────┘
```

---

## 📦 依赖清单

| 依赖 | 版本要求 | 说明 |
|------|:--------:|------|
| 🐹 **Go** | 1.22+ | 编译服务二进制 |
| 🐳 **Docker** | 20.10+ | 构建和运行容器镜像 |
| 🔧 **Make** | GNU Make 4.0+ | 执行构建脚本 |