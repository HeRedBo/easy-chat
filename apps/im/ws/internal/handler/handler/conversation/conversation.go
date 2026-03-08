package conversation

import (
	"time"

	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
	"github.com/HeRedBo/easy-chat/apps/im/ws/ws"
	"github.com/HeRedBo/easy-chat/apps/task/mq/mq"
	"github.com/HeRedBo/easy-chat/pkg/constants"
	"github.com/HeRedBo/easy-chat/pkg/wuid"
	"github.com/mitchellh/mapstructure"
)

func Chat(srvCtx *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {
		var data ws.Chat
		if err := mapstructure.Decode(msg.Data, &data); err != nil {
			err := srv.Send(websocket.NewErrMessage(err), conn)
			if err != nil {
				return
			}
			return
		}

		if data.ConversationId == "" {
			switch data.ChatType {
			case constants.SingleChatType:
				userId := srv.GetUser(conn)
				data.ConversationId = wuid.CombineId(data.RecvId, userId)
			case constants.GroupChatType:
				data.ConversationId = data.RecvId
			}
		}

		err := srvCtx.MsgChatTransfer.Push(&mq.MsgChatTransfer{
			ChatType:       data.ChatType,
			ConversationId: data.ConversationId,
			SendId:         data.SendId,
			RecvId:         data.RecvId,
			MType:          data.MType,
			Content:        data.Content,
			SendTime:       time.Now().UnixNano(),
		})

		if err != nil {
			err := srv.Send(websocket.NewErrMessage(err), conn)
			if err != nil {
				return
			}
		}

		//
		//l := logic.NewUserLogic(context.Background(), srv, srvCtx)
		//
		//if err := l.Chat(&data, srv.GetUser(conn)); err != nil {
		//	err := srv.Send(websocketx.NewErrMessage(err), conn)
		//	if err != nil {
		//		return
		//	}
		//}
	}
}
