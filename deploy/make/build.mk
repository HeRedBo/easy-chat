# ==============================================
# Universal Build Script
# Usage:
#   make -f deploy/make/build.mk info
#   make -f deploy/make/build.mk build
#   make -f deploy/make/build.mk release SVR=user MOD=rpc
#   make -f deploy/make/build.mk release SVR=user MOD=api
#   make -f deploy/make/build.mk release SVR=task MOD=mq
# ==============================================

# 白名单配置
ALLOWED_SVRS := user # im task
ALLOWED_MODS := rpc api #  mq
ALLOWED_ENVS := dev test prv prod

# 外部传入参数
SVR ?= user
MOD ?= rpc
VERSION ?= latest
ENV     ?= dev
# 自动计算路径
APP_BIN       := bin/$(SVR)-$(MOD)
APP_MAIN_FILE := apps/$(SVR)/$(MOD)/$(SVR).go
APP_YAML      := apps/$(SVR)/$(MOD)/etc/dev/$(SVR).yaml

# 镜像相关（请修改为你的阿里云地址）
IMAGE_BASE    := crpi-yw54l3gskjfve0c3.cn-hangzhou.personal.cr.aliyuncs.com/redbo-easy-chat
IMAGE_NAME    := $(IMAGE_BASE)/$(SVR)-$(MOD)-$(ENV)
IMAGE_TAG     := $(IMAGE_NAME):$(VERSION)

# 通用 Dockerfile
DOCKERFILE    := deploy/dockerfile/Dockerfile_local

# ==============================================
# 1. 参数校验
# ==============================================
check:
ifndef SVR
	@echo "[ERROR] Please set SVR=$(strip $(subst $(space),|,$(ALLOWED_SVRS)))"
	@exit 1
endif
ifndef MOD
	@echo "[ERROR] Please set MOD=$(strip $(subst $(space),|,$(ALLOWED_MODS)))"
	@exit 1
endif
ifndef ENV
	@echo "[ERROR] Please set ENV=$(strip $(subst $(space),|,$(ALLOWED_ENVS)))"
	@exit 1
endif

ifneq ($(filter $(SVR),$(ALLOWED_SVRS)),$(SVR))
	@echo "[ERROR] no supported service: $(SVR)"
	@echo "[ALLOWED] $(ALLOWED_SVRS)"
	@exit 1
endif

ifneq ($(filter $(MOD),$(ALLOWED_MODS)),$(MOD))
	@echo "[ERROR] no supported module: $(MOD)"
	@echo "[ALLOWED] $(ALLOWED_MODS)"
	@exit 1
endif

# ==============================================
# 2. 自动从 yaml 读取端口 调用外部 Go 脚本获取端口
# ==============================================
get-port:
	$(eval PORT=$(shell go run ./deploy/make/port_getter.go $(APP_YAML) $(MOD)))
	@echo -n "$(PORT)" > .port

PORT := $(shell cat .port 2>/dev/null)

# ==============================================
# 3. 查看信息（新增）
# ==============================================
info: check get-port
	@echo "========================================"
	@echo "Service SVR: $(SVR)"
	@echo "Module MOD: $(MOD)"
	@echo "Env ENV: $(ENV)"
	@echo "Version VERSION:  $(VERSION)"
	@echo "Port PORT: $(PORT)"
	@echo "Entry Point: $(APP_MAIN_FILE)"
	@echo "Config File:  $(APP_YAML)"
	@echo "Binary: $(APP_BIN)"
	@echo "Image: $(IMAGE_TAG)"
	@echo "Dockerfile:  $(DOCKERFILE)"
	@echo "========================================"
	@rm -f .port

# ==============================================
# 4. 本地构建
# ==============================================
build: check get-port
	@echo "[GO] Compile binary..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o $(APP_BIN) $(APP_MAIN_FILE)

	@echo "[Docker] Build image..."
	docker build . \
		-f $(DOCKERFILE) \
		--build-arg SERVER_NAME=$(SVR) \
		--build-arg SERVER_TYPE=$(MOD) \
		--build-arg PORT=$(PORT) \
		-t $(IMAGE_TAG)

	@rm -f .port
	@echo "[SUCCESS] Build complete: $(IMAGE_TAG)"

# ==============================================
# 5. 推送 & 发布
# ==============================================
push: check
	@echo "[Docker] Push image..."
	docker push $(IMAGE_TAG)

release: build push

.PHONY: check get-port info build push release