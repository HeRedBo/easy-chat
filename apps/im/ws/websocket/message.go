package websocket

// Message 客户端对服务请求结构体
type Message struct {
	Method string      `json:"method,omitempty"`
	UserId string      `json:"user_id,omitempty"`
	FormId string      `json:"form_id,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

func NewMessage(fid string, data interface{}) *Message {

	return &Message{
		FormId: fid,
		Data:   data,
	}
}
