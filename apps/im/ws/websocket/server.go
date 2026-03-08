package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

type Server struct {
	sync.RWMutex

	authentication Authentication
	routes         map[string]HandlerFunc
	addr           string

	connToUser map[*websocket.Conn]string
	userToConn map[string]*websocket.Conn

	upgrader websocket.Upgrader
	logx.Logger
}

func NewServer(addr string, opts ...ServerOptions) *Server {
	opt := NewServerOption(opts...)
	return &Server{
		addr:           addr,
		authentication: opt.Authentication,
		connToUser:     make(map[*websocket.Conn]string),
		userToConn:     make(map[string]*websocket.Conn),
		routes:         make(map[string]HandlerFunc),
		upgrader:       websocket.Upgrader{},
		Logger:         logx.WithContext(context.Background()),
	}
}

func (s *Server) ServerWs(w http.ResponseWriter, r *http.Request) {

	defer func() {
		if err := recover(); err != nil {
			s.Errorf("Websocket server panic: %v", err)
		}
	}()

	// 添加鉴权
	if !s.authentication.Auth(w, r) {
		s.Infof("Websocket server authentication failed")
		return
	}
	// 1. 将HTTP连接升级为WebSocket连接
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.Error("upgrader.Upgrade err: %v", err)
	}

	// 添加连接记录, 会有并发问题
	s.addConn(conn, r)
	//  读取信息，完成请求，还需建立连接
	go s.handleConn(conn)
}

func (s *Server) addConn(conn *websocket.Conn, req *http.Request) {

	uid := s.authentication.UserId(req)
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	// 验证用户是否之前登入过
	if c := s.userToConn[uid]; c != nil {
		// 关闭之前的连接
		err := c.Close()
		if err != nil {
			return
		}
	}
	s.connToUser[conn] = uid
	s.userToConn[uid] = conn
}

func (s *Server) handleConn(conn *websocket.Conn) {

	// 5. 循环读取客户端消息
	for {
		// 读取消息（类型：文本/二进制/Ping/Pong/Close）
		_, msg, err := conn.ReadMessage()
		if err != nil {
			s.Error("websocket.读取消息失败 err: %v, userId %s", err, "")
			// 关闭并删除连接
			return
		}
		// 请求信息
		var message Message
		err = json.Unmarshal(msg, &message)
		if err != nil {
			s.Errorf("json unmarshal err %v, msg %v", err, string(msg))
			return
		}
		// 依据消息进行处理
		if handler, ok := s.routes[message.Method]; ok {
			handler(s, conn, &message)
		} else {
			err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("不存在请求方法 %v 请仔细检查", message.Method)))
			if err != nil {
				s.Errorf(" conn.WriteMessage err %v, msg %v", err, fmt.Sprintf("不存在请求方法 %v 请仔细检查", message.Method))
				return
			}
		}

		// 6. 回复客户端（echo功能）
	}
}

func (s *Server) GetConn(uid string) *websocket.Conn {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	return s.userToConn[uid]
}

func (s *Server) GetCons(uids ...string) []*websocket.Conn {
	if len(uids) == 0 {
		return nil
	}

	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	res := make([]*websocket.Conn, 0, len(uids))
	for _, uid := range uids {
		res = append(res, s.userToConn[uid])
	}
	return res
}

func (s *Server) GetUsers(cones ...*websocket.Conn) []string {

	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()
	var res []string
	if len(cones) == 0 {
		// 获取全部
		res = make([]string, 0, len(s.connToUser))
		for _, uid := range s.connToUser {
			res = append(res, uid)
		}
	} else {
		// 获取部分
		res = make([]string, 0, len(cones))
		for _, conn := range cones {
			res = append(res, s.connToUser[conn])
		}
	}

	return res
}

func (s *Server) Close(conn *websocket.Conn) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()
	uid := s.connToUser[conn]
	if uid == "" {
		// 已经被关闭
		return
	}

	delete(s.connToUser, conn)
	delete(s.userToConn, uid)
	err := conn.Close()
	if err != nil {
		s.Errorf("close Conn err %v, userId %s", err, uid)
		return
	}
}

// region 消息发送
func (s *Server) SendByUserId(msg interface{}, sendIds ...string) error {
	if len(sendIds) == 0 {
		return nil
	}

	return s.Send(msg, s.GetCons(sendIds...)...)
}

func (s *Server) Send(msg interface{}, cones ...*websocket.Conn) error {
	if len(cones) == 0 {
		return nil
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	for _, conn := range cones {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return err
		}
	}

	return nil
}

//  endregion

func (s *Server) AddRoutes(rs []Route) {
	for _, r := range rs {
		s.routes[r.Method] = r.Handler
	}
}
func (s *Server) Start() {
	http.HandleFunc("/ws", s.ServerWs)
	err := http.ListenAndServe(s.addr, nil)
	if err != nil {
		return
	}
}

func (s *Server) Stop() {
	fmt.Println("stop server")
}
