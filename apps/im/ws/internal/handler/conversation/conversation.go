package conversation

import (
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
)

func Chat(srvCtx *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {

		err := srv.SendByUserId(websocket.NewMessage(srv, conn, msg.Data), msg.UserId)
		srv.Info("err ", err)
	}
}
