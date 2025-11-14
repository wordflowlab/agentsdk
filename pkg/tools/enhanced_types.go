package tools

import (
	"context"
)

// Context 工具执行的基础上下文接口
// 提供最小的上下文信息
type Context interface {
	// Context 返回 Go 标准 context
	Context() context.Context

	// Value 获取上下文值
	Value(key interface{}) interface{}
}

// BaseTool 基础工具实现（提供默认的空方法）
type BaseTool struct {
	ToolName        string // 导出字段
	ToolDescription string // 导出字段
}

// NewBaseTool 创建基础工具
func NewBaseTool(name, description string) *BaseTool {
	return &BaseTool{
		ToolName:        name,
		ToolDescription: description,
	}
}

func (t *BaseTool) Name() string {
	return t.ToolName
}

func (t *BaseTool) Description() string {
	return t.ToolDescription
}

func (t *BaseTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *BaseTool) Execute(ctx context.Context, input map[string]interface{}, tc *ToolContext) (interface{}, error) {
	return nil, nil
}

func (t *BaseTool) Prompt() string {
	return ""
}
