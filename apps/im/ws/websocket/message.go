package websocket

type FrameType uint8

const (
	FrameData FrameType = 0x0
	FramePing FrameType = 0x1
)

// Message 客户端对服务请求结构体
type Message struct {
	FrameType `json:"frame_type"`
	Method    string      `json:"method,omitempty"`
	UserId    string      `json:"user_id,omitempty"`
	FormId    string      `json:"form_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

func NewMessage(srv *Server, conn *Conn, data interface{}) *Message {
	fid := srv.GetUsers(conn)[0]
	return &Message{
		FrameType: FrameData,
		FormId:    fid,
		Data:      data,
	}
}
