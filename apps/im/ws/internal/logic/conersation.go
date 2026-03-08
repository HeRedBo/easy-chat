package logic

import (
	"context"
	"time"

	"github.com/HeRedBo/easy-chat/apps/im/immodels"
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/svc"
	websocketx "github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
	"github.com/HeRedBo/easy-chat/apps/im/ws/ws"
	"github.com/HeRedBo/easy-chat/pkg/wuid"
)

type ChatLogSLg interface {
	SingleChatLog(data *ws.Chat, userId string) error
}

type Conversation struct {
	ctx context.Context
	srv *websocketx.Server
	svc *svc.ServiceContext
}

func NewConversation(ctx context.Context, srv *websocketx.Server, svcCtx *svc.ServiceContext) *Conversation {
	return &Conversation{
		ctx: ctx,
		srv: srv,
		svc: svcCtx,
	}
}

func (l *Conversation) SingleChat(data *ws.Chat, userId string) error {
	if data.ConversationId == "" {
		data.ConversationId = wuid.CombineId(userId, data.RecvId)
	}
	// 记录消息
	chatLog := immodels.ChatLog{
		ConversationId: data.ConversationId,
		SendId:         userId,
		RecvId:         data.RecvId,
		ChatType:       data.ChatType,
		MsgFrom:        0,
		MsgType:        data.MType,
		MsgContent:     data.Content,
		SendTime:       time.Now().UnixNano(),
	}
	err := l.svc.ChatLogModel.Insert(l.ctx, &chatLog)
	return err
}
