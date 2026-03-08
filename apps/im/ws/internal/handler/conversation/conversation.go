package conversation

import (
	"context"
	"time"

	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/logic"
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
	"github.com/HeRedBo/easy-chat/apps/im/ws/ws"
	"github.com/HeRedBo/easy-chat/pkg/constants"
	"github.com/mitchellh/mapstructure"
)

// Chat 针对用户处理的消息
func Chat(srvCtx *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {
		var data ws.Chat
		if err := mapstructure.Decode(msg.Data, &data); err != nil {
			_ = srv.Send(websocket.NewErrMessage(err), conn)
			return
		}
		// 处理私聊信息
		switch data.ChatType {
		case constants.SingleChatType:
			err := logic.NewConversation(context.Background(), srv, srvCtx).SingleChat(&data, conn.Uid)
			if err != nil {
				err = srv.Send(websocket.NewErrMessage(err), conn)
				return
			}

			err = srv.SendByUserId(websocket.NewMessage(conn.Uid, ws.Chat{
				ConversationId: data.ConversationId,
				ChatType:       data.ChatType,
				SendId:         conn.Uid,
				RecvId:         data.RecvId,
				SendTime:       time.Now().UnixMilli(),
				Msg:            data.Msg,
			}), data.RecvId)
			if err != nil {
				return
			}
		}

	}
}
