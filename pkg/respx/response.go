package respx

import "context"

type Response struct {
	Code    int         `json:"code"`    // 业务码
	Message string      `json:"message"` // 提示信息
	Data    interface{} `json:"data"`    // 数据
}

// Ok 成功返回
func Ok(ctx context.Context, data interface{}) *Response {
	return &Response{
		Code:    Success,
		Message: GetSuccessMsg(ctx),
		Data:    data,
	}
}

// Fail 失败返回
func Fail(code int, message string) *Response {
	return &Response{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}
