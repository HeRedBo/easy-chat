package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/cache"
)

type Config struct {
	service.ServiceConf

	ListenOn string

	MsgChatTransfer kq.KqConf

	Mysql struct {
		DataSource string
	}

	Cache cache.CacheConf
}
