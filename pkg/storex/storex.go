package storex

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

var (
	reqData    sync.Map
	reqIDCounter int64
)

// RequestIDKey 是 context 中存储 Request ID 的 key
type RequestIDKey struct{}

// GenerateRequestID 生成唯一的 Request ID
func GenerateRequestID() string {
	id := atomic.AddInt64(&reqIDCounter, 1)
	return fmt.Sprintf("req_%d", id)
}

// GetRequestID 从 context 中获取 Request ID
func GetRequestID(ctx context.Context) (string, bool) {
	reqID, ok := ctx.Value(RequestIDKey{}).(string)
	return reqID, ok
}

// Set 存储数据（使用 Request ID 作为 key）
func Set(ctx context.Context, key any, val any) {
	reqID, ok := GetRequestID(ctx)
	if !ok {
		return // 如果没有 Request ID，不存储
	}

	if m, ok := reqData.Load(reqID); ok {
		m.(*sync.Map).Store(key, val)
	} else {
		var m sync.Map
		m.Store(key, val)
		reqData.Store(reqID, &m)
	}
}

// Get 获取数据
func Get(ctx context.Context, key any) (any, bool) {
	reqID, ok := GetRequestID(ctx)
	if !ok {
		return nil, false
	}
	if m, ok := reqData.Load(reqID); ok {
		return m.(*sync.Map).Load(key)
	}
	return nil, false
}

// Clean 清空当前请求数据
func Clean(ctx context.Context) {
	reqID, ok := GetRequestID(ctx)
	if !ok {
		return
	}
	reqData.Delete(reqID)
}

// CleanupByReqID 根据 Request ID 清理（用于 middleware）
func CleanupByReqID(reqID string) {
	reqData.Delete(reqID)
}
