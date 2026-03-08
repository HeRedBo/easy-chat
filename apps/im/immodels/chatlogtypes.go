package immodels

import (
	"time"

	"github.com/HeRedBo/easy-chat/pkg/constants"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatLog struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`

	ConversationId string             `bson:"conversation_id"`
	SendId         string             `bson:"send_id"`
	RecvId         string             `bson:"recv_id"`
	MsgFrom        int                `bson:"msg_from"`
	ChatType       constants.ChatType `bson:"chat_type"`
	MsgType        constants.MType    `bson:"msg_type"`
	MsgContent     string             `bson:"msg_content"`
	SendTime       int64              `bson:"send_time"`
	Status         int                `bson:"status"`
	// TODO: Fill your own fields
	UpdateAt time.Time `bson:"update_at,omitempty" json:"update_at,omitempty"`
	CreateAt time.Time `bson:"create_at,omitempty" json:"create_at,omitempty"`
}
