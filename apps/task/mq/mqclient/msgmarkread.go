package mqclient

import (
	"context"
	"encoding/json"

	"github.com/HeRedBo/easy-chat/apps/task/mq/mq"
	"github.com/zeromicro/go-queue/kq"
)

type MsgReadChatTransferClient interface {
	Push(msg *mq.MsgMarkRead) error
}

type msgReadChatTransferClient struct {
	pusher *kq.Pusher
}

func NewMsgReadChatTransferClient(addr []string, topic string, opts ...kq.PushOption) MsgReadChatTransferClient {
	return &msgReadChatTransferClient{
		pusher: kq.NewPusher(addr, topic),
	}
}

func (c *msgReadChatTransferClient) Push(msg *mq.MsgMarkRead) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 使用 ConversationId 作为 key，保证同一会话的已读回执路由到同一 partition
	// 避免同一会话的已读消息乱序处理
	return c.pusher.PushWithKey(context.Background(), msg.ConversationId, string(body))
}
