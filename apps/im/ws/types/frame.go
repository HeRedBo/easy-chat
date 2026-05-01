package types

// FrameType 帧类型
type FrameType uint8

const (
	FrameData  FrameType = 0x0 // 数据帧
	FramePing  FrameType = 0x1 // 心跳帧
	FrameAck   FrameType = 0x2 // 确认帧
	FrameNoAck FrameType = 0x4 // 无需确认帧
	FrameCAck  FrameType = 0x5 // 客户端确认帧
	FrameErr   FrameType = 0x9 // 错误帧
)
