package websocket

import (
	"time"

	"github.com/HeRedBo/easy-chat/apps/im/ws/types"
)

// FrameType 从 types 包引入，统一管理帧类型常量
// 本地定义类型别名，保持 websocket 包内的使用方式不变
type FrameType = types.FrameType

const (
	FrameData  = types.FrameData
	FramePing  = types.FramePing
	FrameAck   = types.FrameAck
	FrameNoAck = types.FrameNoAck
	FrameCAck  = types.FrameCAck
	FrameErr   = types.FrameErr
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

// GetFrameType 实现 validator.ValidatableMessage 接口
func (m *Message) GetFrameType() uint8 { return uint8(m.FrameType) }

// GetId 实现 validator.ValidatableMessage 接口
func (m *Message) GetId() string { return m.Id }

// GetMethod 实现 validator.ValidatableMessage 接口
func (m *Message) GetMethod() string { return m.Method }

// GetData 实现 validator.ValidatableMessage 接口
func (m *Message) GetData() interface{} { return m.Data }
