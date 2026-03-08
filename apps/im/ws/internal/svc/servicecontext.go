package svc

import (
	"github.com/HeRedBo/easy-chat/apps/im/immodels"
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/config"
	"github.com/HeRedBo/easy-chat/apps/task/mq/mqclient"
)

type ServiceContext struct {
	Config config.Config
	immodels.ChatLogModel

	MsgChatTransfer mqclient.MsgChatTransferClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:          c,
		ChatLogModel:    immodels.NewChatLogModel(c.Mongo.Url, c.Mongo.Db, "chatlog"),
		MsgChatTransfer: mqclient.NewMsgChatTransferClient(c.MsgChatTransfer.Addrs, c.MsgChatTransfer.Topic),
	}
}
