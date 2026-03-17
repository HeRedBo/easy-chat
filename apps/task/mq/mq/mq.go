package mq

import "github.com/HeRedBo/easy-chat/pkg/constants"

type MsgChatTransfer struct {
	// 会话类型 1、私聊 、2.群聊
	ChatType constants.ChatType `json:"chat_type"`
	// 会话ID
	ConversationId string `json:"conversation_id"`
	// 发送者
	SendId string `json:"send_id"`
	// 接收着
	RecvId  string   `json:"recv_id"`
	RecvIds []string `json:"recv_ids"`
	// 消息类型
	constants.MType `json:"msg_type,omitempty"`
	// 消息内容
	Content string `json:"content"`
	// 发送时间
	SendTime int64 `json:"send_time"`
}

type MsgMarkRead struct {
	ChatType constants.ChatType `json:"chat_type,omitempty"`
	// 会话ID
	ConversationId string `json:"conversation_id,omitempty"`
	// 发送着
	SendId string `json:"send_id,omitempty"`
	// 接收着
	RecvId string `json:"recv_id,omitempty"`
	// 已读消息集合
	MsgIds []string `json:"msg_ids,omitempty"`
}
