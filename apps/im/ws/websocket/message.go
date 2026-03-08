package websocket

// Message 客户端对服务请求结构体
type Message struct {
	Method string      `json:"method,omitempty"`
	UserId string      `json:"UserId,omitempty"`
	FormId string      `json:"FormId,omitempty"`
	Data   interface{} `json:"Data,omitempty"`
}

func NewMessage(fid string, data interface{}) *Message {

	return &Message{
		FormId: fid,
		Data:   data,
	}
}
