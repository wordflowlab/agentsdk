package builtin

import (
	"fmt"
)

// NewClaudeErrorResponse 创建Claude兼容的错误响应
func NewClaudeErrorResponse(err error, recommendations ...string) map[string]interface{} {
	return map[string]interface{}{
		"ok":             false,
		"error":          err.Error(),
		"recommendations": recommendations,
	}
}

// ValidateRequired 验证必需参数的通用函数
func ValidateRequired(input map[string]interface{}, required []string) error {
	for _, key := range required {
		if _, exists := input[key]; !exists {
			return fmt.Errorf("missing required parameter: %s", key)
		}
	}
	return nil
}

// GetStringParam 获取字符串参数的通用函数
func GetStringParam(input map[string]interface{}, key string, defaultValue string) string {
	if value, exists := input[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetIntParam 获取整数参数的通用函数
func GetIntParam(input map[string]interface{}, key string, defaultValue int) int {
	if value, exists := input[key]; exists {
		if num, ok := value.(float64); ok {
			return int(num)
		}
	}
	return defaultValue
}

// GetBoolParam 获取布尔参数的通用函数
func GetBoolParam(input map[string]interface{}, key string, defaultValue bool) bool {
	if value, exists := input[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetStringSliceParam 获取字符串数组参数的通用函数
func GetStringSliceParam(input map[string]interface{}, key string) []string {
	if value, exists := input[key]; exists {
		if slice, ok := value.([]interface{}); ok {
			result := make([]string, len(slice))
			for i, item := range slice {
				if str, ok := item.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}
	return []string{}
}