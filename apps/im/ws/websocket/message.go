package websocket

import "time"

type FrameType uint8

const (
	FrameData  FrameType = 0x0
	FramePing  FrameType = 0x1
	FrameAck   FrameType = 0x2
	FrameNoAck FrameType = 0x4
	FrameCAck  FrameType = 0x5
	FrameErr   FrameType = 0x9
)

// Message 客户端对服务请求结构体
type Message struct {
	FrameType `json:"frame_type"`
	Id        string      `json:"id"`
	AckSeq    int         `json:"ack_seq,omitempty"`
	AckTime   time.Time   `json:"ack_time,omitempty"`
	ErrCount  int         `json:"err_count,omitempty"`
	Method    string      `json:"method,omitempty"`
	UserId    string      `json:"user_id,omitempty"`
	FormId    string      `json:"form_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

func NewMessage(formId string, data interface{}) *Message {
	return &Message{
		FrameType: FrameData,
		FormId:    formId,
		Data:      data,
	}
}

//func NewMessage(srv *Server, conn *Conn, data interface{}) *Message {
//	fid := srv.GetUsers(conn)[0]
//	return &Message{
//		FrameType: FrameData,
//		FormId:    fid,
//		Data:      data,
//	}
//}

func NewErrMessage(err error) *Message {
	return &Message{
		FrameType: FrameErr,
		Data:      err.Error(),
	}
}
