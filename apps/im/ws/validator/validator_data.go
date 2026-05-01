package validator

import (
	"fmt"

	"github.com/HeRedBo/easy-chat/pkg/mapstructure"
	"github.com/go-playground/validator/v10"
)

func init() {
	RegisterDataValidator("conversation.chat", ChatSendValidator{})
	RegisterDataValidator("conversation.markChat", MarkReadValidator{})
}

// ==================== conversation.chat ====================

// ChatSendReq conversation.chat 方法的 Data 请求结构体
type ChatSendReq struct {
	ConversationId string `validate:"omitempty"`
	ChatType       int    `validate:"required,oneof=1 2"` // 1:群聊 2:私聊
	RecvId         string `validate:"required"`           // 接收者ID
	MType          int    `validate:"required,oneof=0"`   // 消息类型：0=文本
	Content        string `validate:"required,max=500"`   // 消息内容，最长500
}

// ChatSendValidator conversation.chat 的 Data 验证器
type ChatSendValidator struct{}

func (ChatSendValidator) Validate(data interface{}) error {
	req := &ChatSendReq{}
	if err := mapstructure.Decode(data, req); err != nil {
		return fmt.Errorf("data decode failed: %w", err)
	}
	return validator.New().Struct(req)
}

// ==================== conversation.markChat ====================

// MarkReadReq conversation.markChat 方法的 Data 请求结构体
type MarkReadReq struct {
	ChatType       int      `validate:"required,oneof=1 2"` // 1:群聊 2:私聊
	RecvId         string   `validate:"required"`           // 接收者ID
	ConversationId string   `validate:"omitempty"`
	MsgIds         []string `validate:"required,min=1,dive,required"` // 消息ID列表，至少1个
}

// MarkReadValidator conversation.markChat 的 Data 验证器
type MarkReadValidator struct{}

func (MarkReadValidator) Validate(data interface{}) error {
	req := &MarkReadReq{}
	if err := mapstructure.Decode(data, req); err != nil {
		return fmt.Errorf("data decode failed: %w", err)
	}
	return validator.New().Struct(req)
}
