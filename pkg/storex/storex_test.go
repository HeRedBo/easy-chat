package storex

import (
	"context"
	"testing"
)

func TestRequestIDGeneration(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()
	
	if id1 == id2 {
		t.Errorf("Expected different request IDs, got %s and %s", id1, id2)
	}
	
	t.Logf("Generated request IDs: %s, %s", id1, id2)
}

func TestStorexWithRequestID(t *testing.T) {
	// 生成 Request ID
	reqID := GenerateRequestID()
	
	// 创建带有 Request ID 的 context
	ctx := context.WithValue(context.Background(), RequestIDKey{}, reqID)
	
	// 测试 Set 和 Get
	testKey := "test_key"
	testValue := "test_value"
	
	Set(ctx, testKey, testValue)
	
	value, ok := Get(ctx, testKey)
	if !ok {
		t.Error("Expected to get value, but got false")
		return
	}
	
	if value != testValue {
		t.Errorf("Expected value %s, got %v", testValue, value)
	}
	
	// 测试 Clean
	Clean(ctx)
	
	_, ok = Get(ctx, testKey)
	if ok {
		t.Error("Expected value to be cleaned, but still exists")
	}
}

func TestStorexWithoutRequestID(t *testing.T) {
	// 不带 Request ID 的 context
	ctx := context.Background()
	
	// Set 应该不会存储
	Set(ctx, "key", "value")
	
	// Get 应该返回 false
	_, ok := Get(ctx, "key")
	if ok {
		t.Error("Expected false when no request ID in context")
	}
}

func TestCleanupByReqID(t *testing.T) {
	reqID := GenerateRequestID()
	ctx := context.WithValue(context.Background(), RequestIDKey{}, reqID)
	
	// 存储数据
	Set(ctx, "key1", "value1")
	Set(ctx, "key2", "value2")
	
	// 验证数据存在
	_, ok := Get(ctx, "key1")
	if !ok {
		t.Error("Expected key1 to exist")
	}
	
	// 使用 CleanupByReqID 清理
	CleanupByReqID(reqID)
	
	// 验证数据被清理
	_, ok = Get(ctx, "key1")
	if ok {
		t.Error("Expected data to be cleaned after CleanupByReqID")
	}
}
