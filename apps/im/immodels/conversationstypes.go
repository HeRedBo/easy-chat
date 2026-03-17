package immodels

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Conversations struct {
	ID bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`

	// TODO: Fill your own fields
	UserId           string                   `bson:"user_id" json:"user_id"`
	ConversationList map[string]*Conversation `bson:"conversation_list,omitempty" json:"conversation_list,omitempty"`

	UpdateAt time.Time `bson:"update_at,omitempty" json:"update_at,omitempty"`
	CreateAt time.Time `bson:"create_at,omitempty" json:"create_at,omitempty"`
}
