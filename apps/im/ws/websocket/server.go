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

	connToUser map[*Conn]string
	userToConn map[string]*Conn

	upgrader websocket.Upgrader
	logx.Logger
}

func NewServer(addr string, opts ...ServerOptions) *Server {
	opt := NewServerOption(opts...)
	return &Server{
		addr:           addr,
		authentication: opt.Authentication,
		connToUser:     make(map[*Conn]string),
		userToConn:     make(map[string]*Conn),
		routes:         make(map[string]HandlerFunc),
		upgrader:       websocket.Upgrader{},
		Logger:         logx.WithContext(context.Background()),
	}
}

func (s *Server) ServerWs(w http.ResponseWriter, r *http.Request) {

	defer func() {
		if err := recover(); err != nil {
			s.Errorf("server handler ws recover err : %v", err)
		}
	}()

	// 添加鉴权
	if !s.authentication.Auth(w, r) {
		s.Infof("Websocket server authentication failed")
		_, err := w.Write([]byte("authentication failed"))
		if err != nil {
			s.Errorf("server handler ws write err : %v", err)
			return
		}
		return
	}

	//conn, err := s.upgrader.Upgrade(w, r, nil)
	//if err != nil {
	//	s.Error("upgrader.Upgrade err: %v", err)
	//}
	conn := NewConn(s, w, r)
	if conn == nil {
		return
	}

	// 添加连接记录, 会有并发问题
	s.addConn(conn, r)
	//  读取信息，完成请求，还需建立连接
	go s.handleConn(conn)
}

func (s *Server) addConn(conn *Conn, req *http.Request) {
	// 此处是map的写操作，在操作上会存在并发的可能问题
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

// 根据连接对象执行任务
func (s *Server) handleConn(conn *Conn) {
	uids := s.GetUsers(conn)
	conn.Uid = uids[0]
	// 5. 循环读取客户端消息
	for {
		// 读取消息（类型：文本/二进制/Ping/Pong/Close）
		_, msg, err := conn.ReadMessage()
		if err != nil {
			s.Error("websocket.读取消息失败 err: %v, userId %s", err, "")
			// 关闭并删除连接
			s.Close(conn)
			return
		}
		// 请求信息
		var message Message
		err = json.Unmarshal(msg, &message)
		if err != nil {
			s.Errorf("json unmarshal err %v, msg %v", err, string(msg))
			_ = s.Send(NewErrMessage(err), conn)
			continue
		}
		// 依据消息进行处理
		switch message.FrameType {
		case FramePing:
			// ping 回复
			err := s.Send(&Message{FrameType: FramePing}, conn)
			if err != nil {
				return
			}
		case FrameData:
			if handler, ok := s.routes[message.Method]; ok {
				handler(s, conn, &message)
			} else {
				//err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("不存在请求方法 %v 请仔细检查", message.Method)))
				err := s.Send(&Message{
					FrameType: FrameData,
					Data:      fmt.Sprintf("不存在请求方法 %v 请仔细检查", message.Method),
				}, conn)
				if err != nil {
					s.Errorf(" conn.WriteMessage err %v, msg %v", err, fmt.Sprintf("不存在请求方法 %v 请仔细检查", message.Method))
					return
				}
			}
		}
	}
}

func (s *Server) GetConn(uid string) *Conn {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	return s.userToConn[uid]
}

func (s *Server) GetUser(conn *Conn) string {
	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	return s.connToUser[conn]
}

func (s *Server) GetCons(uids ...string) []*Conn {
	if len(uids) == 0 {
		return nil
	}

	s.RWMutex.RLock()
	defer s.RWMutex.RUnlock()

	res := make([]*Conn, 0, len(uids))
	for _, uid := range uids {
		res = append(res, s.userToConn[uid])
	}
	return res
}

func (s *Server) GetUsers(cones ...*Conn) []string {

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

func (s *Server) Close(conn *Conn) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()
	uid := s.connToUser[conn]
	if uid == "" {
		// 已经被关闭了连接
		return
	}

	fmt.Printf("关闭与%s的链接\n", uid)
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

func (s *Server) Send(msg interface{}, cones ...*Conn) error {
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
