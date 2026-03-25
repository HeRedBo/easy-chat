package msgTransfer

import (
	"sync"
	"time"

	"github.com/HeRedBo/easy-chat/apps/im/ws/ws"
	"github.com/HeRedBo/easy-chat/pkg/constants"
	"github.com/zeromicro/go-zero/core/logx"
)

type groupMsgRead struct {
	mu             sync.Mutex
	conversationId string
	push           *ws.Push
	pushCh         chan *ws.Push
	count          int

	// 上次推送时间
	pushTime time.Time
	done     chan struct{}
}

func newGroupMsgRead(push *ws.Push, pushCh chan *ws.Push) *groupMsgRead {
	m := &groupMsgRead{
		conversationId: push.ConversationId,
		push:           push,
		pushCh:         pushCh,
		count:          1,
		pushTime:       time.Now(),
		done:           make(chan struct{}),
	}

	// 开启协程处理 合并消息逻辑
	go m.transfer()
	return m
}

// 合并消息
func (m *groupMsgRead) mergePush(push *ws.Push) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.count++
	for msgId, read := range push.ReadRecords {
		m.push.ReadRecords[msgId] = read
	}
}

func (m *groupMsgRead) transfer() {
	// 1、超时控制
	// 2、超量发送

	// 计算 ticker 间隔，确保至少 50ms，最多 500ms
	tickerInterval := GroupMsgReadRecordDelayTime / 10
	if tickerInterval < 50*time.Millisecond {
		tickerInterval = 50 * time.Millisecond
	}
	if tickerInterval > 500*time.Millisecond {
		tickerInterval = 500 * time.Millisecond
	}

	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()
	defer func() {
		// 处理退出前的剩余消息
		m.mu.Lock()
		if m.push != nil {
			// 推送剩余消息
			logx.Infof("退出前推送剩余消息 %v", m.push)
			m.pushCh <- m.push
			m.push = nil
			m.count = 0
		}
		m.mu.Unlock()
	}()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			// 检查是否达到数量阈值
			if m.count >= GroupMsgReadRecordDelayCount && m.push != nil {
				push := m.push
				m.count = 0
				m.push = nil
				m.pushTime = time.Now()
				m.mu.Unlock()
				// 推送消息
				logx.Infof("达到数量阈值推送 %v", push)
				m.pushCh <- push
				continue
			}
			// 检查是否超时
			if time.Since(m.pushTime) >= GroupMsgReadRecordDelayTime && m.push != nil {
				push := m.push
				m.count = 0
				m.push = nil
				m.pushTime = time.Now()
				m.mu.Unlock()
				// 推送消息
				logx.Infof("超时推送 %v", push)
				m.pushCh <- push
				continue
			}

			// 检查是否空闲
			if m.isIdle() {
				m.mu.Unlock()
				// 使得 MsgReadTransfer 释放
				m.pushCh <- &ws.Push{
					ChatType: constants.GroupChatType,
					Content:  m.conversationId,
				}
				continue
			}
			m.mu.Unlock()
		case <-m.done:
			return
		}
	}
}

//func (m *groupMsgRead) transfer_old() {
//	// 1、超时控制
//	// 2、超量发送
//
//	timer := time.NewTimer(GroupMsgReadRecordDelayTime / 2)
//	defer timer.Stop()
//
//	for {
//		select {
//		case <-timer.C:
//			m.mu.Lock()
//
//			pushTime := m.pushTime
//			val := GroupMsgReadRecordDelayTime - time.Since(pushTime)
//			push := m.push
//			logx.Infof("timer C %v val %v", time.Now(), val)
//
//			if val > 0 && m.count < GroupMsgReadRecordDelayCount || push == nil {
//				if val > 0 {
//					timer.Reset(val)
//				}
//				// 未达标
//				m.mu.Unlock()
//				continue
//			}
//			m.pushTime = time.Now()
//			m.count = 0
//			m.push = nil
//			timer.Reset(GroupMsgReadRecordDelayTime / 2)
//			m.mu.Unlock()
//			// 推送消息
//			logx.Infof("超过 合并的条件推送 %v", push)
//			m.pushCh <- push
//		case <-m.done:
//			return
//		default:
//			m.mu.Lock()
//
//			logx.Infof("groupMsgRead count %v push %v", m.count, m.push)
//
//			if m.count >= GroupMsgReadRecordDelayCount {
//				logx.Infof("groupMsgRead transfer ConversationId %v push %v", m.push.ConversationId, m.push)
//				dump.P("groupMsgRead transfer ConversationId %v push %v", m.push.ConversationId, m.push)
//				push := m.push
//				m.count = 0
//				m.push = nil
//				m.mu.Unlock()
//
//				logx.Infof("default 达到推送量推送 %v", push)
//				m.pushCh <- push
//				continue
//			}
//
//			if m.IsIdle() {
//				m.mu.Unlock()
//				// 使得 MsgReadTransfer 释放
//				m.pushCh <- &ws.Push{
//					ChatType: constants.GroupChatType,
//					Content:  m.conversationId,
//				}
//				continue
//			}
//			m.mu.Unlock()
//
//			tempDelay := GroupMsgReadRecordDelayTime / 4
//			if tempDelay > time.Second {
//				tempDelay = time.Second
//			}
//			time.Sleep(tempDelay)
//		}
//
//	}
//}

func (m *groupMsgRead) IsIdle() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isIdle()
}
func (m *groupMsgRead) isIdle() bool {
	pushTime := m.pushTime
	val := GroupMsgReadRecordDelayTime*2 - time.Since(pushTime)

	if val <= 0 && m.push == nil && m.count == 0 {
		return true
	}

	return false
}

func (m *groupMsgRead) clear() {
	select {
	case <-m.done:
	default:
		close(m.done)
	}
	m.push = nil
}
