// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/HeRedBo/easy-chat/apps/im/rpc/imclient"
	"github.com/HeRedBo/easy-chat/apps/social/api/internal/config"
	"github.com/HeRedBo/easy-chat/apps/social/rpc/socialclient"
	"github.com/HeRedBo/easy-chat/apps/user/rpc/userclient"
	"github.com/HeRedBo/easy-chat/pkg/interceptor"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

var retryPolicy = `{
	"methodConfig" : [{
		"name": [{
			"service": "user.User"
		}],
		"waitForReady": true,
		"retryPolicy": {
			"maxAttempts": 4,
			"initialBackoff": "0.001s",
			"maxBackoff": "0.002s",
			"backoffMultiplier": 1.0,
			"retryableStatusCodes": ["UNKNOWN"]
		}
	}]
}`

type ServiceContext struct {
	Config config.Config
	*redis.Redis
	userclient.User
	socialclient.Social
	imclient.Im
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		Redis:  redis.MustNewRedis(c.Redisx),
		User:   userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
		Social: socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc,
			zrpc.WithUnaryClientInterceptor(interceptor.DefaultIdempotentClient),
			zrpc.WithDialOption(grpc.WithDefaultServiceConfig(retryPolicy)),
		)),
		Im: imclient.NewIm(zrpc.MustNewClient(c.Imrpc)),
	}
}
