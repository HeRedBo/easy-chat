package msgTransfer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"github.com/HeRedBo/easy-chat/apps/im/ws/ws"
	"github.com/HeRedBo/easy-chat/apps/task/mq/internal/svc"
	"github.com/HeRedBo/easy-chat/apps/task/mq/mq"
	"github.com/HeRedBo/easy-chat/pkg/bitmap"
	"github.com/HeRedBo/easy-chat/pkg/constants"
	"github.com/zeromicro/go-queue/kq"
)

var (
	GroupMsgReadRecordDelayTime  = time.Second
	GroupMsgReadRecordDelayCount = 10
)

const (
	GroupMsgReadHandlerAtTransfer = iota
	GroupMsgReadHandlerDelayTransfer
)

type MsgReadTransfer struct {
	*BaseMsgTransfer

	mu        sync.Mutex
	push      chan *ws.Push
	groupMses map[string]*groupMsgRead
}

func NewMsgReadTransfer(svc *svc.ServiceContext) kq.ConsumeHandler {
	m := &MsgReadTransfer{
		BaseMsgTransfer: NewBaseMsgTransfer(svc),
		groupMses:       make(map[string]*groupMsgRead, 1),
		push:            make(chan *ws.Push, 1),
	}

	if svc.Config.MsgReadHandler.GroupMsgReadHandler != GroupMsgReadHandlerAtTransfer {
		if svc.Config.MsgReadHandler.GroupMsgReadRecordDelayCount > 0 {
			GroupMsgReadRecordDelayCount = svc.Config.MsgReadHandler.GroupMsgReadRecordDelayCount
		}
		if svc.Config.MsgReadHandler.GroupMsgReadRecordDelayTime > 0 {
			GroupMsgReadRecordDelayTime = time.Duration(svc.Config.MsgReadHandler.GroupMsgReadRecordDelayTime) * time.Second
		}
	}
	// 开启协程处理已读消息发送
	go m.transfer()
	return m
}

func (m *MsgReadTransfer) Consume(ctx context.Context, key, value string) error {

	m.Info("MsgReadTransfer ", value)

	var data mq.MsgMarkRead

	if err := json.Unmarshal([]byte(value), &data); err != nil {
		// 消息格式错误，无法解析，属于致命错误，不重试
		m.Errorf("Failed to unmarshal message: %v", err)
		return nil // 返回 nil，避免无限重试
	}

	// 更新消息聊天记录中的已读状态（必须成功，属于致命错误）
	ReadRecords, err := m.UpdateChatLogRead(ctx, &data)
	if err != nil {
		// 数据库更新失败，返回 error，消息会重新消费
		m.Errorf("Failed to update chat log read status: %v", err)
		return err
	}

	push := &ws.Push{
		ChatType:       data.ChatType,
		ConversationId: data.ConversationId,
		SendId:         data.SendId,
		RecvId:         data.RecvId,
		ContentType:    constants.ContentMakeRead,
		MsgKind:        constants.MsgKindReadAck,
		ReadRecords:    ReadRecords,
	}

	// 推送已读回执（非致命错误，失败不影响已读状态持久化）
	switch data.ChatType {
	case constants.SingleChatType:
		// 直接推送
		m.push <- push

	case constants.GroupChatType:
		// 判断是否开启合并消息的处理
		if m.svcCtx.Config.MsgReadHandler.GroupMsgReadHandler == GroupMsgReadHandlerAtTransfer {
			m.push <- push
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		push.SendId = ""

		if _, ok := m.groupMses[push.ConversationId]; ok {
			m.Infof("merge push %v", push.ConversationId)
			// 合并请求
			m.groupMses[push.ConversationId].mergePush(push)
		} else {
			m.Infof("neGroupMsgRead %v", push.ConversationId)
			m.groupMses[push.ConversationId] = newGroupMsgRead(push, m.push)
		}
	}

	return nil

	//fmt.Println("MsgReadTransfer ", ReadRecords)
	//
	//// 将已读消息发送给用户
	//return m.Transfer(ctx, &ws.Push{
	//	ConversationId: data.ConversationId,
	//	ChatType:       data.ChatType,
	//	SendId:         data.SendId,
	//	RecvId:         data.RecvId,
	//	ContentType:    constants.ContentMakeRead,
	//	ReadRecords:    ReadRecords,
	//})
}

func (m *MsgReadTransfer) transfer() {
	for push := range m.push {

		if push.RecvId != "" || len(push.RecvIds) > 0 {
			if err := m.Transfer(context.Background(), push); err != nil {
				m.Errorf("msgTransfer push %v err %v", push, err)
			}
		}

		if push.ChatType == constants.SingleChatType {
			continue
		}

		if m.svcCtx.Config.MsgReadHandler.GroupMsgReadHandler == GroupMsgReadHandlerAtTransfer {
			continue
		}
		// 清空数据
		m.mu.Lock()
		if _, ok := m.groupMses[push.ConversationId]; ok && m.groupMses[push.ConversationId].isIdle() {
			m.groupMses[push.ConversationId].clear()
			delete(m.groupMses, push.ConversationId)
		}
		m.mu.Unlock()
	}
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
