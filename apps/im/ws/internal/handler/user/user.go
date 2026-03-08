package user

import (
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/svc"
	websocketx "github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
	"github.com/gorilla/websocket"
)

func OnLine(svc *svc.ServiceContext) websocketx.HandlerFunc {
	return func(srv *websocketx.Server, conn *websocket.Conn, msg *websocketx.Message) {
		//uids := srv.GetUsers()
		u := srv.GetUsers(conn)
		err := srv.Send(websocketx.NewMessage(u[0], msg), conn)
		srv.Info("err ", err)
	}
}
