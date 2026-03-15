package immodels

import (
	"time"

	"github.com/HeRedBo/easy-chat/pkg/constants"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Conversation struct {
	ID bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	// TODO: Fill your own fields
	ConversationId string             `bson:"conversation_id,omitempty"`
	ChatType       constants.ChatType `bson:"chat_type,omitempty"`
	//TargetId       string             `bson:"targetId,omitempty"`
	IsShow bool     `bson:"is_show,omitempty"`
	Total  int      `bson:"total,omitempty"`
	Seq    int64    `bson:"seq"`
	Msg    *ChatLog `bson:"msg,omitempty"`

	UpdateAt time.Time `bson:"update_at,omitempty" json:"update_at,omitempty"`
	CreateAt time.Time `bson:"create_at,omitempty" json:"create_at,omitempty"`
}
