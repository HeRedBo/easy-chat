package user

import (
	"github.com/HeRedBo/easy-chat/apps/im/ws/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
)

// OnLine 获取在线用户列表
func OnLine(srvCtx *svc.ServiceContext) websocket.HandlerFunc {
	return func(srv *websocket.Server, conn *websocket.Conn, msg *websocket.Message) {
		uids := srv.GetUsers()
		u := srv.GetUsers(conn)
		err := srv.Send(websocket.NewMessage(u[0], uids), conn)
		srv.Info("err ", err)
	}
}
