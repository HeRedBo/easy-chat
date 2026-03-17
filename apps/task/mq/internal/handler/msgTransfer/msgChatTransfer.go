package msgTransfer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/HeRedBo/easy-chat/apps/im/immodels"
	"github.com/HeRedBo/easy-chat/apps/im/ws/ws"
	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/task/mq/mq"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type MsgChatTransfer struct {
	*BaseMsgTransfer
}

func NewMsgChatTransfer(svc *svc.ServiceContext) *MsgChatTransfer {
	return &MsgChatTransfer{
		NewBaseMsgTransfer(svc),
	}
}

func (m *MsgChatTransfer) Consume(ctx context.Context, key, value string) error {
	// 使用带上下文的日志，便于追踪
	// m.WithContext(ctx).Infof("PaymentSuccess key :%s , val :%s", key, value)
	// fmt.Printf("=> %s\n", value)
	// dump.P("key :", key, "Value :", value)
	fmt.Println("key : ", key, " value : ", value)

	var (
		data  mq.MsgChatTransfer
		MsgId = bson.NewObjectID()
	)
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return err
	}

	// 记录数据
	if err := m.addChatLog(ctx, MsgId, &data); err != nil {
		return err
	}

	// 推送消息
	return m.Transfer(ctx, &ws.Push{
		ConversationId: data.ConversationId,
		ChatType:       data.ChatType,
		SendId:         data.SendId,
		RecvIds:        data.RecvIds,
		SendTime:       data.SendTime,
		MType:          data.MType,
		MsgId:          MsgId.Hex(),
		Content:        data.Content,
	})
}

func (m *MsgChatTransfer) addChatLog(ctx context.Context, MsgId bson.ObjectID, data *mq.MsgChatTransfer) error {
	// 记录消息
	chatLog := immodels.ChatLog{
		ID:             MsgId,
		ConversationId: data.ConversationId,
		SendId:         data.SendId,
		RecvId:         data.RecvId,
		ChatType:       data.ChatType,
		MsgFrom:        0,
		MsgType:        data.MType,
		MsgContent:     data.Content,
		SendTime:       data.SendTime,
	}
	err := m.svcCtx.ChatLogModel.Insert(ctx, &chatLog)
	if err != nil {
		return err
	}

	return m.svcCtx.ConversationModel.UpdateMsg(ctx, &chatLog)
}

//func (m *MsgChatTransfer) single(ctx context.Context, data *mq.MsgChatTransfer) error {
//	return m.svcCtx.WsClient.Send(websocket.Message{
//		FrameType: websocket.FrameNoAck,
//		Method:    "push",
//		FormId:    constants.SYSTEM_ROOT_UID,
//		Data:      data,
//	})
//}
//
//func (m *MsgChatTransfer) group(ctx context.Context, data *mq.MsgChatTransfer) error {
//
//	res, err := m.svcCtx.Social.GroupUsers(ctx, &socialclient.GroupUsersReq{
//		GroupId: data.RecvId,
//	})
//	if err != nil {
//		return err
//	}
//
//	data.RecvIds = make([]string, 0, len(res.List))
//	for _, member := range res.List {
//		if member.UserId == data.RecvId {
//			continue
//		}
//		data.RecvIds = append(data.RecvIds, member.UserId)
//	}
//
//	return m.svcCtx.WsClient.Send(websocket.Message{
//		FrameType: websocket.FrameNoAck,
//		Method:    "push",
//		FormId:    constants.SYSTEM_ROOT_UID,
//		Data:      data,
//	})
//}
