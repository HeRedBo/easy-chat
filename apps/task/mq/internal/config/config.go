package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	service.ServiceConf

	SocialRpc zrpc.RpcClientConf

	ListenOn string

	MsgChatTransfer        kq.KqConf
	MsgReadTransferHandler kq.KqConf

	MsgReadHandler struct {
		GroupMsgReadHandler          int
		GroupMsgReadRecordDelayTime  int64
		GroupMsgReadRecordDelayCount int
	}

	Redisx redis.RedisConf

	Mongo struct {
		Url string
		Db  string
	}

	Mysql struct {
		DataSource string
	}

	Cache cache.CacheConf

	Ws struct {
		Host string
	}
}
