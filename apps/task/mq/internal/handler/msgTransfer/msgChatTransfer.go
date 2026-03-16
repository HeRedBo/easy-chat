package msgTransfer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/HeRedBo/easy-chat/apps/im/immodels"
	"github.com/HeRedBo/easy-chat/apps/im/ws/websocket"
	"github.com/HeRedBo/easy-chat/apps/social/rpc/socialclient"
	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/task/mq/mq"
	"github.com/HeRedBo/easy-chat/pkg/constants"
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
	// m.WithContext(ctx).Infof("PaymentSuccess key :%s , val :%s", key, value)
	// fmt.Printf("=> %s\n", value)
	// dump.P("key :", key, "Value :", value)
	fmt.Println("key : ", key, " value : ", value)

	var (
		data mq.MsgChatTransfer
	)
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return err
	}

	// 记录数据
	if err := m.addChatLog(ctx, &data); err != nil {
		return err
	}

	// 推送消息
	switch data.ChatType {
	case constants.SingleChatType:
		return m.sinle(ctx, &data)
	case constants.GroupChatType:
		return m.group(ctx, &data)
	}
	return nil
}

func (m *MsgChatTransfer) addChatLog(ctx context.Context, data *mq.MsgChatTransfer) error {
	// 记录消息
	chatLog := immodels.ChatLog{
		ConversationId: data.ConversationId,
		SendId:         data.SendId,
		RecvId:         data.RecvId,
		ChatType:       data.ChatType,
		MsgFrom:        0,
		MsgType:        data.MType,
		MsgContent:     data.Content,
		SendTime:       data.SendTime,
	}
	err := m.svc.ChatLogModel.Insert(ctx, &chatLog)
	if err != nil {
		return err
	}

	return m.svc.ConversationModel.UpdateMsg(ctx, &chatLog)
}

func (m *MsgChatTransfer) sinle(ctx context.Context, data *mq.MsgChatTransfer) error {
	return m.svc.WsClient.Send(websocket.Message{
		FrameType: websocket.FrameNoAck,
		Method:    "push",
		FormId:    constants.SYSTEM_ROOT_UID,
		Data:      data,
	})
}

func (m *MsgChatTransfer) group(ctx context.Context, data *mq.MsgChatTransfer) error {

	res, err := m.svc.Social.GroupUsers(ctx, &socialclient.GroupUsersReq{
		GroupId: data.RecvId,
	})
	if err != nil {
		return err
	}

	data.RecvIds = make([]string, 0, len(res.List))
	for _, member := range res.List {
		if member.UserId == data.RecvId {
			continue
		}
		data.RecvIds = append(data.RecvIds, member.UserId)
	}

	return m.svc.WsClient.Send(websocket.Message{
		FrameType: websocket.FrameNoAck,
		Method:    "push",
		FormId:    constants.SYSTEM_ROOT_UID,
		Data:      data,
	})
}
