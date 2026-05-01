package mqclient

import (
	"context"
	"encoding/json"

	"github.com/HeRedBo/easy-chat/apps/task/mq/mq"
	"github.com/zeromicro/go-queue/kq"
)

type MsgChatTransferClient interface {
	Push(msg *mq.MsgChatTransfer) error
}

type msgChatTransferClient struct {
	pusher *kq.Pusher
}

func NewMsgChatTransferClient(addr []string, topic string, opts ...kq.PushOption) MsgChatTransferClient {
	return &msgChatTransferClient{
		pusher: kq.NewPusher(addr, topic),
	}
}

func (c *msgChatTransferClient) Push(msg *mq.MsgChatTransfer) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 使用 ConversationId 作为 key，保证同一会话消息路由到同一 partition
	// 这样同一会话的消息由同一个消费者处理，保证会话内消息有序
	// 即使 Consumers > 1，同一会话的消息也不会乱序
	return c.pusher.PushWithKey(context.Background(), msg.ConversationId, string(body))
}
