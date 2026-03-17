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

	return c.pusher.Push(context.Background(), string(body))
}
