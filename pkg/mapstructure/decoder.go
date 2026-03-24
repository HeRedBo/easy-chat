package mapstructure

import (
	"unicode"

	"github.com/mitchellh/mapstructure"
)

// Decoder 结构体用于存储解码器配置
type Decoder struct {
	config *mapstructure.DecoderConfig
}

// NewDecoder 创建一个新的解码器
func NewDecoder(result interface{}) *Decoder {
	config := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           result,
		TagName:          "mapstructure",
		MatchName: func(mapKey, structField string) bool {
			return mapKey == structField || toSnakeCase(mapKey) == structField
		},
	}
	return &Decoder{config: config}
}

// Decode 解码数据到目标结构体
func (d *Decoder) Decode(input interface{}) error {
	decoder, err := mapstructure.NewDecoder(d.config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}

// Helper function: convert camelCase to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// Decode 简化版解码函数
func Decode(input interface{}, result interface{}) error {
	decoder := NewDecoder(result)
	return decoder.Decode(input)
}
