# EasyChat 通用构建部署脚本

## 一、环境规范（行业标准）
- dev    开发环境
- test   测试环境
- prv    预发布环境
- prod   生产环境

## 二、镜像命名规则
{服务名}-{模块名}-{环境}

示例：
- user-rpc-dev
- user-api-dev
- task-mq-prv
- im-rpc-prod

## 三、重要说明
**所有命令必须在项目根目录执行**，否则会出现路径错误、编译失败、配置文件找不到等问题！
相关的服务镜像需要本地登录阿里云镜像仓库，否则会报错。


## 四、使用命令
### 查看配置信息（默认：user rpc dev）
make -f deploy/make/build.mk info

### 构建本地镜像（默认：user rpc dev）
make -f deploy/make/build.mk build

### 构建 + 推送镜像
make -f deploy/make/build.mk release

### 指定服务 & 模块
make -f deploy/make/build.mk build SVR=user MOD=api

### 指定环境（dev/test/prv/prod）
make -f deploy/make/build.mk build ENV=prv

### 完整自定义（服务+模块+环境）
make -f deploy/make/build.mk build SVR=task MOD=mq ENV=prod

## 五、配置说明
### 1. 阿里云镜像地址
请在 `deploy/make/build.mk` 中修改：
IMAGE_BASE

### 2. 服务白名单
ALLOWED_SVRS := user im task

### 3. 模块白名单
ALLOWED_MODS := rpc api mq

### 4. 环境白名单
ALLOWED_ENVS := dev test prv prod

## 六、依赖
- Docker
- Go
- Make