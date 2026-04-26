package websocket

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Conn struct {
	*websocket.Conn
	s                 *Server
	Uid               string
	idleMu            sync.Mutex // guard the following
	idle              time.Time
	maxConnectionIdle time.Duration

	// 消息队列
	messageMu      sync.Mutex
	readMessages   []*Message
	readMessageSeq map[string]*Message

	message chan *Message

	done chan struct{}
}

func NewConn(s *Server, w http.ResponseWriter, r *http.Request) *Conn {

	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.Errorf("upgrader err %v", err)
		return nil
	}

	conn := &Conn{
		Conn:              c,
		s:                 s,
		idle:              time.Now(),
		maxConnectionIdle: defaultMaxConnectionIdle,
		done:              make(chan struct{}),
		message:           make(chan *Message, 1),
		readMessageSeq:    make(map[string]*Message),
	}

	if s.opt.maxConnectionIdle > 0 {
		conn.maxConnectionIdle = s.opt.maxConnectionIdle
	}

	// 增加对客户端的安全检测
	go conn.keepalive()
	return conn
}

func (c *Conn) ReadMessage() (messageType int, p []byte, err error) {
	messageType, p, err = c.Conn.ReadMessage()
	c.idle = time.Time{}
	return messageType, p, err
}

func (c *Conn) WriteMessage(messageType int, p []byte) error {
	err := c.Conn.WriteMessage(messageType, p)
	c.idle = time.Time{}
	return err
}

func (c *Conn) Close() error {
	close(c.done)
	return c.Conn.Close()
}

// 将消息记录到队列中
func (c *Conn) appendMsgMq(msg *Message) {
	c.messageMu.Lock()
	defer c.messageMu.Unlock()

	// 读队列
	if m, ok := c.readMessageSeq[msg.Id]; ok {
		// 客户端可能重复发送了消息， 或者收到了ack消息
		if len(c.readMessages) == 0 {
			// 数据已经被处理，顾属于重复处理
			return
		}

		if m.Id != msg.Id || m.AckSeq >= msg.AckSeq {
			// 数据处理已经存在，顾属于重复处理
			// ack 的消息内容还是一致或最大接收的序号说明也是重复了
			return
		}

		// 等于最新的消息
		c.readMessageSeq[msg.Id] = msg
		return
	}

	// 因为意外发送ack消息 直接过滤
	if msg.FrameType == FrameAck {
		return
	}

	c.readMessages = append(c.readMessages, msg)
	c.readMessageSeq[msg.Id] = msg
}

//func (c *Conn) ReadMessage() (*Message, error) {
//
//}

func (c *Conn) keepalive() {
	idleTimer := time.NewTimer(c.maxConnectionIdle)
	defer func() {
		idleTimer.Stop()
	}()

	for {
		select {
		case <-idleTimer.C:
			c.idleMu.Lock()
			idle := c.idle
			if idle.IsZero() {
				c.idleMu.Unlock()
				idleTimer.Reset(c.maxConnectionIdle)
				continue
			}
			val := c.maxConnectionIdle - time.Since(idle)
			c.idleMu.Unlock()
			if val <= 0 {
				// The connection has been idle for a duration of keepalive.MaxConnectionIdle or more.
				// Gracefully close the connection.
				c.s.Close(c)
				return
			}
			idleTimer.Reset(val)
		case <-c.done:
			return
		}
	}
}
