package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/HeRedBo/easy-chat/apps/im/ws/validator"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

// AckType 应答信号类型
type AckType int

const (
	// NoAck 不进行ack确认
	NoAck AckType = iota
	// OnlyAck 只回-两次通信
	OnlyAck
	// RigorAck 严格 - 三次通信
	RigorAck
)

func (t AckType) Tostring() string {
	switch t {
	case OnlyAck:
		return "OnlyAck"
	case RigorAck:
		return "RigorAck"
	default:
		return "NoAck"
	}
}

type Server struct {
	sync.RWMutex

	opt *serverOption
	*threading.TaskRunner

	authentication Authentication
	routes         map[string]HandlerFunc
	validator      *validator.Validator
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
		validator:      validator.NewValidator(),

		connToUser: make(map[*Conn]string),
		userToConn: make(map[string]*Conn),
		routes:     make(map[string]HandlerFunc),
		upgrader:   websocket.Upgrader{},
		Logger:     logx.WithContext(context.Background()),
		opt:        &opt,
		TaskRunner: threading.NewTaskRunner(opt.concurrency),
	}
}

func (s *Server) ServerWs(w http.ResponseWriter, r *http.Request) {

	defer func() {
		if err := recover(); err != nil {
			s.Errorf("server handler ws recover err : %v", err)
		}
	}()

	conn := NewConn(s, w, r)
	if conn == nil {
		return
	}
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

	// 添加连接记录, 会有并发问题
	s.addConn(conn, r)
	//  读取信息，完成请求，还需建立连接
	go s.handlerConn(conn)
}

func (s *Server) addConn(conn *Conn, req *http.Request) {

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
func (s *Server) handlerConn(conn *Conn) {

	conn.Uid = s.GetUser(conn)
	// 处理写
	go s.handlerWrite(conn)

	if s.opt.ack != NoAck {
		// 接受确认消息
		go s.readAck(conn)
	}

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

		// TODO 给客户端一个 ack
		// 请求信息
		var message Message
		err = json.Unmarshal(msg, &message)
		if err != nil {
			s.Errorf("json unmarshal err %v, msg %v", err, string(msg))
			_ = s.Send(NewErrMessage(err), conn)
			continue
		}

		// 数据验证
		if err := s.validator.Validate(&message); err != nil {
			s.Errorf("message validate err: %v", err)
			_ = s.Send(NewErrMessage(err), conn)
			continue
		}

		if s.opt.ack != NoAck && message.FrameType != FrameNoAck {
			// 将消息添加到队列中
			s.Infof("conn message wirte in msgMq %v", message)
			conn.appendMsgMq(&message)
		} else {
			conn.message <- &message
		}

		// 依据消息类型分类处理
		//switch message.FrameType {
		//case FramePing:
		//	// ping 消息回复
		//	err := s.Send(&Message{FrameType: FramePing}, conn)
		//	if err != nil {
		//		return
		//	}
		//case FrameData:
		//	if handler, ok := s.routes[message.Method]; ok {
		//		handler(s, conn, &message)
		//	} else {
		//		err := s.Send(&Message{
		//			FrameType: FrameData,
		//			Data:      fmt.Sprintf("不存在请求方法 %v 请仔细检查", message.Method),
		//		}, conn)
		//		if err != nil {
		//			s.Errorf(" conn.WriteMessage err %v, msg %v", err, fmt.Sprintf("不存在请求方法 %v 请仔细检查", message.Method))
		//			return
		//		}
		//	}
		//}

	}
}

func (s *Server) isAck(message *Message) bool {
	if message == nil {
		return s.opt.ack != NoAck
	}
	return s.opt.ack != NoAck && message.FrameType != FrameNoAck
}
func (s *Server) readAck(conn *Conn) {

	// 记录失败次数在处理
	send := func(msg *Message, conn *Conn) error {
		err := s.Send(msg, conn)
		if err == nil {
			return nil
		}

		s.Errorf("message ack OnlyAck err: %v message %v", err, msg)
		conn.readMessages[0].ErrCount++
		conn.messageMu.Unlock()

		tempDelay := time.Duration(200*conn.readMessages[0].ErrCount) * time.Millisecond
		if delay := 1 * time.Second; tempDelay > delay {
			tempDelay = delay
		}
		time.Sleep(tempDelay)
		return err
	}

	for {
		select {
		case <-conn.done:
			// 关闭了链接
			s.Infof("close messsage ack uid %v", conn.Uid)
			return
		default:
		}

		conn.messageMu.Lock()

		if len(conn.readMessages) == 0 {
			conn.messageMu.Unlock()
			// 没有消息可以睡 100 毫秒 -- 让程序缓一缓
			time.Sleep(time.Millisecond * 100)
			continue
		}

		// 取出队列中的第一个数据
		message := conn.readMessages[0]
		// 根据ack 的确认策略选择合适得处理方式
		switch s.opt.ack {
		case OnlyAck:
			err := send(&Message{
				FrameType: FrameAck,
				AckSeq:    message.AckSeq + 1,
				Id:        message.Id,
			}, conn)
			if err != nil {
				return
			}
			// 只回答 向客户端发送 ack
			conn.readMessages = conn.readMessages[1:]
			conn.messageMu.Unlock()
			conn.message <- message
			s.Infof("message ack onlyAck send success mid %v", message.Id)
		case RigorAck:
			if message.AckSeq == 0 {
				//还未发送过确认信息
				conn.readMessages[0].AckSeq++
				conn.readMessages[0].AckTime = time.Now()
				err := send(&Message{
					FrameType: FrameAck,
					AckSeq:    message.AckSeq,
					Id:        message.Id,
				}, conn)
				if err != nil {
					return
				}
				conn.messageMu.Unlock()
				s.Infof("message ack rigorAck send  mid %v seq %v, time %v", message.Id, message.AckSeq, message.AckTime.Unix())
				continue
			}

			// 1. 客户端返回结果，再一次确认
			// 2. 客户端没有确认，考虑是否超过了ack的确认时间
			// 2.1 未超过，重新发送
			// 2.2 超过结束确认

			// 已经发送过序号了 需要等待客户端返回确认
			msgSeq := conn.readMessageSeq[message.Id]
			if msgSeq.AckSeq > message.AckSeq {
				// 客户端确认成功， 可以处理业务了
				conn.readMessages = conn.readMessages[1:]
				conn.messageMu.Unlock()
				conn.message <- message
				s.Infof("message ack rigorAck success mid %v ", message.Id)
				continue
			}

			// 很显然没有处理成功，先看看有没有超时
			// 2. 客户端没有确认，考虑是否超过了ack的确认时间
			val := s.opt.actTimeout - time.Since(msgSeq.AckTime)
			if !message.AckTime.IsZero() && val <= 0 {
				// TODO 超时了 可以选择断开与客户端连接，实际具体细节仍需要自己结合业务完善，此选择放弃该消息
				s.Errorf("message ack rigorAck fail mid %v  time %v because timeout", message.Id, message.AckTime)
				// 2.2 超过结束确认
				delete(conn.readMessageSeq, message.Id)
				conn.readMessages = conn.readMessages[1:]
				conn.messageMu.Unlock()
				continue
			}
			// 2.1 未超过，重新发送
			conn.messageMu.Unlock()
			err := send(&Message{
				FrameType: FrameAck,
				AckSeq:    message.AckSeq,
				Id:        message.Id,
			}, conn)
			if err != nil {
				return
			}
			// 没有超时，让程序等等
			time.Sleep(3 * time.Second) // 调整为 3s
		}
	}

}

func (s *Server) handlerWrite(conn *Conn) {
	for {
		select {
		case <-conn.done:
			return
		case message := <-conn.message:
			// 依据请求消息类型分类处理
			switch message.FrameType {
			case FramePing:
				// ping 回复
				s.Send(&Message{FrameType: FramePing}, conn)
			case FrameData, FrameNoAck:
				if handler, ok := s.routes[message.Method]; ok {
					handler(s, conn, message)
				} else {
					s.Send(&Message{
						FrameType: FrameData,
						Data:      fmt.Sprintf("不存在请求方法 %v 请仔细检查", message.Method),
					}, conn)
				}
			}

			if s.isAck(message) {
				// 删除 消息ack的序号记录
				conn.messageMu.Lock()
				delete(conn.readMessageSeq, message.Id)
				conn.messageMu.Unlock()
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
