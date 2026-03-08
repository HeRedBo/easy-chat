package ws

import (
	"github.com/HeRedBo/easy-chat/pkg/constants"
	_ "github.com/mitchellh/mapstructure"
)

type (
	Msg struct {
		constants.MType `mapstructure:"m_type"`
		Content         string `mapstructure:"content"`
	}

	Chat struct {
		ConversationId     string `mapstructure:"conversation_id"`
		constants.ChatType `mapstructure:"chat_type"`
		SendId             string `mapstructure:"send_id"`
		RecvId             string `mapstructure:"recv_id"`
		SendTime           int64  `mapstructure:"send_time"`
		Msg                `mapstructure:"msg"`
	}

	Push struct {
		// 消息类型，1.私聊、2.群聊
		constants.ChatType `mapstructure:"chat_type"`
		// 会话ID
		ConversationId string `mapstructure:"conversation_id"`
		// 发送者
		SendId string `mapstructure:"send_id"`
		// 接受者
		RecvId string `mapstructure:"recv_id"`
		// 发送时间
		SendTime int64 `mapstructure:"send_time"`
		// 消息内容类型
		constants.MType `mapstructure:"m_type"`
		Content         string `mapstructure:"content"`
	}
)
