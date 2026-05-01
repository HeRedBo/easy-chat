// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/HeRedBo/easy-chat/pkg/zrpcx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf

	ImRpc     zrpc.RpcClientConf
	SocialRpc zrpc.RpcClientConf
	UserRpc   zrpc.RpcClientConf

	RpcRetry map[string]zrpcx.RetryPolicy `json:",optional"`

	JwtAuth struct {
		AccessSecret string
		AccessExpire int64
	}
}
