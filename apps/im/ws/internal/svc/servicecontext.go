package svc

import (
	"github.com/HeRedBo/easy-chat/apps/im/immodels"
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/config"
)

type ServiceContext struct {
	Config config.Config
	immodels.ChatLogModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		ChatLogModel: immodels.NewChatLogModel(c.Mongo.Url, c.Mongo.Db, "chatlog"),
	}
}
