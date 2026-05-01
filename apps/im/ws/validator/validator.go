package validator

import (
	"fmt"

	"github.com/HeRedBo/easy-chat/apps/im/ws/types"
)

// validFrameTypes 合法的帧类型列表
var validFrameTypes = []uint8{
	uint8(types.FrameData), uint8(types.FramePing), uint8(types.FrameAck),
	uint8(types.FrameNoAck), uint8(types.FrameCAck), uint8(types.FrameErr),
}

// frameTypesRequiringId 必须携带 Id 的帧类型列表
var frameTypesRequiringId = []uint8{
	uint8(types.FrameData), uint8(types.FrameAck), uint8(types.FrameNoAck),
}

// frameTypesRequiringMethod 必须携带 Method 的帧类型列表
var frameTypesRequiringMethod = []uint8{
	uint8(types.FrameData),
}

// ValidatableMessage 验证消息接口
// websocket.Message 通过实现该接口来满足验证器的入参要求
// 避免 validator 包直接依赖 websocket 包，防止循环导入
type ValidatableMessage interface {
	GetFrameType() uint8
	GetId() string
	GetMethod() string
	GetData() interface{}
}

// DataValidator Data 层验证接口
// 每个 Method 对应一个实现，用于验证 Data 字段内部的具体业务数据
type DataValidator interface {
	Validate(data interface{}) error
}

// globalHandlers 全局 Data 验证器注册表
// 各 Method 的验证器通过 init() 在 validator_data.go 中自注册
var globalHandlers map[string]DataValidator

func init() {
	globalHandlers = make(map[string]DataValidator)
}

// RegisterDataValidator 全局注册 Method 对应的 Data 验证器
// 在 validator_data.go 的 init() 中调用，实现自注册
func RegisterDataValidator(method string, handler DataValidator) {
	globalHandlers[method] = handler
}

// Validator 消息验证器
// 负责消息外层字段校验和按 Method 路由的 Data 内层校验
type Validator struct {
	handlers map[string]DataValidator // 按 Method 注册的 Data 验证器
}

// NewValidator 创建验证器
// 自动加载通过 RegisterDataValidator 全局注册的所有 Data 验证器
func NewValidator() *Validator {
	// 复制全局注册表到实例，避免并发问题
	handlers := make(map[string]DataValidator, len(globalHandlers))
	for method, handler := range globalHandlers {
		handlers[method] = handler
	}

	return &Validator{
		handlers: handlers,
	}
}

// Register 注册 Method 对应的 Data 验证器（实例级别）
// 用于需要动态添加验证器的场景，一般推荐使用 RegisterDataValidator 自注册
func (v *Validator) Register(method string, handler DataValidator) {
	v.handlers[method] = handler
}

// Validate 统一验证入口
// 第一步：验证消息外层字段（FrameType、Id 等）
// 第二步：根据 Method 查找 Data 验证器，验证 Data 内层
func (v *Validator) Validate(msg ValidatableMessage) error {
	// 1. 外层校验：验证消息固定字段
	if err := v.validateOuter(msg); err != nil {
		return err
	}

	// 2. 内层校验：根据 Method 找到对应的 Data 验证器
	method := msg.GetMethod()
	if method == "" {
		// 非 Data 帧（如 Ping、Ack）不需要内层校验
		return nil
	}

	handler, ok := v.handlers[method]
	if !ok {
		// 未注册验证器的 Method，跳过内层校验
		return nil
	}

	if err := handler.Validate(msg.GetData()); err != nil {
		return fmt.Errorf("method %s data validate failed: %w", method, err)
	}

	return nil
}

// validateOuter 验证消息外层字段
func (v *Validator) validateOuter(msg ValidatableMessage) error {
	ft := msg.GetFrameType()

	// 校验 FrameType：必须是合法的枚举值
	if !contains(validFrameTypes, ft) {
		return fmt.Errorf("invalid frame_type: %d", ft)
	}

	// 校验 Id：特定帧类型必须有 Id
	if contains(frameTypesRequiringId, ft) && msg.GetId() == "" {
		return fmt.Errorf("id is required for frame_type %d", ft)
	}

	// 校验 Method：特定帧类型必须有 Method
	if contains(frameTypesRequiringMethod, ft) && msg.GetMethod() == "" {
		return fmt.Errorf("method is required for frame_type %d", ft)
	}

	return nil
}

// contains 判断切片中是否包含指定值
func contains(slice []uint8, val uint8) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
