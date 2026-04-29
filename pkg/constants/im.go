package constants

type MType int

const (
	TextMType MType = iota
)

type ChatType int

const (
	GroupChatType ChatType = iota + 1
	SingleChatType
)

type ContentType int

const (
	ContentChatMsg ContentType = iota
	ContentMakeRead
)

type MsgKind int

const (
	MsgKindChat MsgKind = iota
	MsgKindReadAck
	MsgKindRevoke
	MsgKindSystem
)
