package msgTransfer

import (
	"context"
	"fmt"

	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/svc"
	"github.com/gookit/goutil/dump"
	"github.com/zeromicro/go-zero/core/logx"
)

type MsgChatTransfer struct {
	logx.Logger
	svc *svc.ServiceContext
}

func NewMsgChatTransfer(svc *svc.ServiceContext) *MsgChatTransfer {
	return &MsgChatTransfer{
		Logger: logx.WithContext(context.Background()),
		svc:    svc,
	}
}

func (m *MsgChatTransfer) Consume(ctx context.Context, key, value string) error {
	// 使用带上下文的日志，便于追踪
	m.WithContext(ctx).Infof("PaymentSuccess key :%s , val :%s", key, value)
	fmt.Printf("=> %s\n", value)
	dump.P("key :", key, "Value :", value)
	return nil
}
