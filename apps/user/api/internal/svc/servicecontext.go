// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/HeRedBo/easy-chat/apps/user/api/internal/config"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/userclient"
	"github.com/HeRedBo/easy-chat/pkg/zrpcx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	*redis.Redis
	userclient.User
}

func NewServiceContext(c config.Config) *ServiceContext {
	retryOpt := zrpcx.BuildGlobalRetryOption(c.RpcRetry)
	return &ServiceContext{
		Config: c,
		Redis:  redis.MustNewRedis(c.Redisx),
		User:   userclient.NewUser(zrpc.MustNewClient(c.UserRpc, retryOpt)),
	}
}
