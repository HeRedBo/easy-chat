package msgTransfer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/HeRedBo/easy-chat/apps/im/ws/ws"
	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/task/mq/mq"
	"github.com/HeRedBo/easy-chat/pkg/bitmap"
	"github.com/HeRedBo/easy-chat/pkg/constants"
)

type MsgReadTransfer struct {
	*BaseMsgTransfer
}

func NewMsgReadTransfer(svc *svc.ServiceContext) *MsgReadTransfer {
	return &MsgReadTransfer{
		BaseMsgTransfer: NewBaseMsgTransfer(svc),
	}
}

func (m *MsgReadTransfer) Consume(ctx context.Context, key, value string) error {

	m.Info("MsgReadTransfer ", value)

	var data mq.MsgMarkRead

	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return nil
	}

	// 更新消息聊天记录中的已读状态
	ReadRecords, err := m.UpdateChatLogRead(ctx, &data)
	if err != nil {
		return err
	}

	fmt.Println("MsgReadTransfer ", ReadRecords)

	// 将已读消息发送给用户
	return m.Transfer(ctx, &ws.Push{
		ConversationId: data.ConversationId,
		ChatType:       data.ChatType,
		SendId:         data.SendId,
		RecvId:         data.RecvId,
		ContentType:    constants.ContentMakeRead,
		ReadRecords:    ReadRecords,
	})
}

func (m *MsgReadTransfer) UpdateChatLogRead(ctx context.Context, data *mq.MsgMarkRead) (map[string]string, error) {
	res := make(map[string]string)
	chatlogs, err := m.svcCtx.ChatLogModel.ListByMsgIds(ctx, data.MsgIds)
	if err != nil {
		return res, err
	}
	m.Infof("chatlogs: %v", chatlogs)

	for _, chatlog := range chatlogs {
		switch chatlog.ChatType {
		case constants.GroupChatType:
			readRecords := bitmap.Load(chatlog.ReadRecords)
			readRecords.Set(data.SendId)
			chatlog.ReadRecords = readRecords.Export()
		case constants.SingleChatType:
			chatlog.ReadRecords = []byte{1}
		}

		res[chatlog.ID.Hex()] = base64.StdEncoding.EncodeToString(chatlog.ReadRecords)

		err := m.svcCtx.ChatLogModel.UpdateMakeRead(ctx, chatlog.ID, chatlog.ReadRecords)
		if err != nil {
			m.Errorf("update make read err: %v", err)
		}
	}
	return res, nil
}
