package zaploggerfilter

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// Mask 掩码字符串
var Mask = "***"

// SensitiveDataFilter 负责敏感数据的检测和过滤
type SensitiveDataFilter struct {
	sensitiveFields map[string]bool
}

// NewSensitiveDataFilter 创建一个新的敏感数据过滤器
// fields: 需要被视为敏感的字段名称列表
func NewSensitiveDataFilter(fields []string) *SensitiveDataFilter {

	sensitiveMap := make(map[string]bool, len(fields)+5) // 额外空间给信用卡相关字段

	// 将所有字段转换为小写并存储
	for _, field := range fields {
		lowerField := strings.ToLower(field)
		sensitiveMap[lowerField] = true
	}

	return &SensitiveDataFilter{
		sensitiveFields: sensitiveMap,
	}
}

// IsSensitiveField 检查给定字段名是否为敏感字段
// fieldName: 要检查的字段名
// 返回: 如果是敏感字段则返回true
func (f *SensitiveDataFilter) IsSensitiveField(fieldName string) bool {
	if fieldName == "" {
		return false
	}
	// 转换为小写以实现大小写不敏感的比较
	lowerField := strings.ToLower(fieldName)
	// 检查是否在敏感字段列表中
	return f.sensitiveFields[lowerField]
}

// MaskSensitiveData 递归地对map中的敏感数据进行掩码处理
// data: 要处理的数据（如果为nil则返回nil）
// 返回: 处理后的数据，敏感字段值被替换为掩码
func (f *SensitiveDataFilter) MaskSensitiveData(data map[string]interface{}) map[string]interface{} {
	// 处理nil输入
	if data == nil {
		return nil
	}

	result := make(map[string]interface{}, len(data))

	for key, value := range data {
		// 检查键是否为敏感字段
		lowerKey := strings.ToLower(key)
		if f.IsSensitiveField(lowerKey) {
			result[key] = Mask
			continue
		}

		// 递归处理嵌套结构
		switch v := value.(type) {
		case map[string]interface{}:
			// 递归处理嵌套的map
			result[key] = f.MaskSensitiveData(v)
		case []interface{}:
			// 处理切片类型
			result[key] = f.maskSliceData(v)
		default:
			// 保留原始值，不检查内容
			result[key] = v
		}
	}

	return result
}

// maskSliceData 处理切片中的敏感数据
// slice: 要处理的切片（如果为nil则返回nil）
// 返回: 处理后的切片
func (f *SensitiveDataFilter) maskSliceData(slice []interface{}) []interface{} {
	// 处理nil输入
	if slice == nil {
		return nil
	}

	result := make([]interface{}, len(slice))

	for i, item := range slice {
		switch v := item.(type) {
		case map[string]interface{}:
			// 递归处理嵌套的map
			result[i] = f.MaskSensitiveData(v)
		case []interface{}:
			// 递归处理嵌套的切片
			result[i] = f.maskSliceData(v)
		default:
			// 保留原始值，不检查内容
			result[i] = v
		}
	}

	return result
}

// SensitiveDataMarshaler 自定义JSON序列化器，用于在序列化过程中过滤敏感数据
type SensitiveDataMarshaler struct {
	Data   interface{}
	Filter *SensitiveDataFilter
}

// MarshalJSON 实现json.Marshaler接口
func (m *SensitiveDataMarshaler) MarshalJSON() ([]byte, error) {
	// 处理nil过滤器
	if m.Filter == nil {
		return json.Marshal(m.Data)
	}

	// 处理不同类型的数据
	switch v := m.Data.(type) {
	case map[string]interface{}:
		// 对于map类型，直接处理
		maskedData := m.Filter.MaskSensitiveData(v)
		return json.Marshal(maskedData)
	case []interface{}:
		// 对于数组类型，直接处理
		maskedSlice := m.Filter.maskSliceData(v)
		return json.Marshal(maskedSlice)
	default:
		// 对于其他类型，先序列化为JSON，然后解析为map进行处理
		jsonData, err := json.Marshal(m.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}

		// 尝试解析为map
		var dataMap map[string]interface{}
		err = json.Unmarshal(jsonData, &dataMap)
		if err == nil {
			// 掩码敏感字段
			maskedData := m.Filter.MaskSensitiveData(dataMap)
			// 重新序列化为JSON
			result, marshalErr := json.Marshal(maskedData)
			if marshalErr != nil {
				return nil, fmt.Errorf("failed to marshal masked data: %w", marshalErr)
			}
			return result, nil
		}

		// 尝试解析为数组
		var dataArray []interface{}
		err = json.Unmarshal(jsonData, &dataArray)
		if err == nil {
			// 掩码数组中的敏感字段
			maskedArray := m.Filter.maskSliceData(dataArray)
			// 重新序列化为JSON
			result, marshalErr := json.Marshal(maskedArray)
			if marshalErr != nil {
				return nil, fmt.Errorf("failed to marshal masked array: %w", marshalErr)
			}
			return result, nil
		}

		// 如果不是对象或数组类型，直接返回原始数据
		return jsonData, nil
	}
}

// SensitiveDataEncoder 集成了敏感数据过滤功能的zap编码器
type SensitiveDataEncoder struct {
	zapcore.Encoder
	Filter *SensitiveDataFilter
}

// EncodeEntry 重写编码方法，在编码过程中过滤敏感字段
func (e *SensitiveDataEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	// 处理nil过滤器
	if e.Filter == nil {
		return e.Encoder.EncodeEntry(ent, fields)
	}

	// 处理空字段列表
	if len(fields) == 0 {
		return e.Encoder.EncodeEntry(ent, fields)
	}

	// 预分配过滤后的字段列表，容量至少为原始字段数
	filteredFields := make([]zapcore.Field, 0, len(fields))

	// 检查并替换敏感字段
	for _, field := range fields {
		// 转换键为小写进行比较
		lowerKey := strings.ToLower(field.Key)

		// 检查字段名是否为敏感字段
		if e.Filter.IsSensitiveField(lowerKey) {
			// 敏感字段直接替换为掩码字符串
			filteredFields = append(filteredFields, zap.String(field.Key, Mask))
		} else if (field.Type == zapcore.ReflectType || field.Type == zapcore.ObjectMarshalerType) && field.Interface != nil {
			// 对于复杂类型，使用自定义序列化器处理
			marshaler := &SensitiveDataMarshaler{
				Data:   field.Interface,
				Filter: e.Filter,
			}
			filteredFields = append(filteredFields, zap.Any(field.Key, marshaler))
		} else {
			// 其他字段保持不变
			filteredFields = append(filteredFields, field)
		}
	}

	// 使用原始编码器进行编码
	return e.Encoder.EncodeEntry(ent, filteredFields)
}
