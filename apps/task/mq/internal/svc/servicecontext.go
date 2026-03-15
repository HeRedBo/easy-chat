package svc

import (
	"net/http"

	"github.com/HeRedBo/easy-chat/apps/im/immodels"
	"github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/config"
	"github.com/HeRedBo/easy-chat/pkg/constants"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	config.Config
	WsClient websocket.Client
	*redis.Redis

	immodels.ChatLogModel
	immodels.ConversationModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	svc := &ServiceContext{
		Config:            c,
		Redis:             redis.MustNewRedis(c.Redisx),
		ChatLogModel:      immodels.NewChatLogModel(c.Mongo.Url, c.Mongo.Db, "chat_log"),
		ConversationModel: immodels.NewConversationModel(c.Mongo.Url, c.Mongo.Db, "conversation"),
	}

	token, err := svc.GetSystemToken()
	if err != nil {
		panic(err)
	}
	header := http.Header{}
	header.Set("Authorization", token)
	svc.WsClient = websocket.NewClient(c.Ws.Host, websocket.WithClientHeader(header))
	return svc
}

func (svc *ServiceContext) GetSystemToken() (string, error) {
	return svc.Redis.Get(constants.REDIS_SYSTEM_ROOT_TOKEN)
}
