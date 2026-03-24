package ws

import (
	"github.com/HeRedBo/easy-chat/pkg/constants"
	_ "github.com/mitchellh/mapstructure"
)

type (
	Msg struct {
		constants.MType `mapstructure:"m_type" json:"m_type"`
		Content         string            `mapstructure:"content" json:"content,omitempty"`
		MsgId           string            `mapstructure:"msg_id"`
		ReadRecords     map[string]string `mapstructure:"read_records"` // 已读记录
	}

	Chat struct {
		ConversationId     string `mapstructure:"conversation_id" json:"conversation_id,omitempty"`
		constants.ChatType `mapstructure:"chat_type" json:"chat_type,omitempty"`
		SendId             string `mapstructure:"send_id" json:"send_id,omitempty"`
		RecvId             string `mapstructure:"recv_id" json:"recv_id,omitempty"`
		SendTime           int64  `mapstructure:"send_time" json:"send_time,omitempty"`
		Msg                `mapstructure:"msg" json:"msg,omitempty"`
	}

	Push struct {
		// 消息类型，1.私聊、2.群聊
		ChatType constants.ChatType `mapstructure:"chat_type"`
		// 会话ID
		ConversationId string `mapstructure:"conversation_id"`
		// 发送者
		SendId string `mapstructure:"send_id"`
		// 接收者
		RecvId  string   `mapstructure:"recv_id"`
		RecvIds []string `mapstructure:"recv_ids"`
		// 发送时间
		SendTime int64 `mapstructure:"send_time"`
		// 消息内容类型
		MType       constants.MType       `mapstructure:"m_type"`
		MsgId       string                `mapstructure:"msg_id"`
		ReadRecords map[string]string     `mapstructure:"read_records"` // 已读记录
		ContentType constants.ContentType `mapstructure:"content_type"`
		Content     string                `mapstructure:"content"`
	}

	MarkRead struct {
		constants.ChatType `mapstructure:"chat_type"`
		RecvId             string `mapstructure:"recv_id"`
		// 会话id
		ConversationId string   `mapstructure:"conversation_id"`
		MsgIds         []string `mapstructure:"msg_ids"`
	}
)
