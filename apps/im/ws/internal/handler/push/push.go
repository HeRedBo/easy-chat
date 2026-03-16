package push

import (
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
	"github.com/HeRedBo/easy-chat/apps/im/ws/ws"
	"github.com/HeRedBo/easy-chat/pkg/constants"
	"github.com/mitchellh/mapstructure"
)

func Push(svc *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {
		var data ws.Push
		if err := mapstructure.Decode(msg.Data, &data); err != nil {
			_ = srv.Send(websocket.NewErrMessage(err))
			return
		}

		// 发送的目标
		rconn := srv.GetConn(data.RecvId)
		if rconn == nil {
			// todo: 目标离线
			return
		}

		srv.Infof("push msg %v", data)
		switch data.ChatType {
		case constants.SingleChatType:
			err := single(srv, &data, data.RecvId)
			if err != nil {
				return
			}
		case constants.GroupChatType:
			err := group(srv, &data)
			if err != nil {
				return
			}
		}
	}
}

func single(srv *websocket.Server, data *ws.Push, recvId string) error {
	rconn := srv.GetConn(data.RecvId)
	if rconn == nil {
		// todo: 目标离线
		return nil
	}
	srv.Infof("push msg %v", data)
	return srv.Send(websocket.NewMessage(data.SendId, &ws.Chat{
		ConversationId: data.ConversationId,
		ChatType:       data.ChatType,
		SendTime:       data.SendTime,
		Msg: ws.Msg{
			MType:   data.MType,
			Content: data.Content,
		},
	}), rconn)
}

// 基于并发发送
func group(srv *websocket.Server, data *ws.Push) error {
	for _, id := range data.RecvIds {
		func(id string) {
			srv.Schedule(func() {
				err := single(srv, data, id)
				if err != nil {
					return
				}
			})
		}(id)
	}
	return nil
}
