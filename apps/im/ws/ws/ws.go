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
)
