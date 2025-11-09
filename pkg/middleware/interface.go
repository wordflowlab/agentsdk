package middleware

import (
	"context"

	"github.com/wordflowlab/agentsdk/pkg/tools"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// ModelRequest 模型请求
type ModelRequest struct {
	Messages     []types.Message
	SystemPrompt string
	Tools        []tools.Tool
	Metadata     map[string]interface{}
}

// ModelResponse 模型响应
type ModelResponse struct {
	Message  types.Message
	Metadata map[string]interface{}
}

// ToolCallRequest 工具调用请求
type ToolCallRequest struct {
	ToolCallID   string
	ToolName     string
	ToolInput    map[string]interface{}
	Tool         tools.Tool
	Context      *tools.ToolContext
	Metadata     map[string]interface{}
}

// ToolCallResponse 工具调用响应
type ToolCallResponse struct {
	Result   interface{}
	Metadata map[string]interface{}
}

// ModelCallHandler 模型调用处理器
type ModelCallHandler func(ctx context.Context, req *ModelRequest) (*ModelResponse, error)

// ToolCallHandler 工具调用处理器
type ToolCallHandler func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error)

// Middleware 中间件接口
// 中间件采用洋葱模型,支持请求和响应的拦截处理
type Middleware interface {
	// Name 返回中间件名称
	Name() string

	// Priority 返回优先级 (数值越小优先级越高,越早执行)
	// 建议范围: 0-1000
	// 0-100: 系统核心中间件
	// 100-500: 功能中间件
	// 500-1000: 用户自定义中间件
	Priority() int

	// Tools 注入工具列表
	// 返回中间件提供的工具,会被合并到 Agent 的工具集中
	Tools() []tools.Tool

	// WrapModelCall 包装模型调用
	// 在模型调用前后执行自定义逻辑
	// handler: 下一层处理器(可能是下一个中间件或最终的模型调用)
	WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error)

	// WrapToolCall 包装工具调用
	// 在工具调用前后执行自定义逻辑
	// handler: 下一层处理器(可能是下一个中间件或最终的工具执行)
	WrapToolCall(ctx context.Context, req *ToolCallRequest, handler ToolCallHandler) (*ToolCallResponse, error)

	// OnAgentStart Agent 启动时回调
	OnAgentStart(ctx context.Context, agentID string) error

	// OnAgentStop Agent 停止时回调
	OnAgentStop(ctx context.Context, agentID string) error
}

// BaseMiddleware 基础中间件实现
// 提供默认的空实现,子类只需覆盖需要的方法
type BaseMiddleware struct {
	name     string
	priority int
}

// NewBaseMiddleware 创建基础中间件
func NewBaseMiddleware(name string, priority int) *BaseMiddleware {
	return &BaseMiddleware{
		name:     name,
		priority: priority,
	}
}

func (m *BaseMiddleware) Name() string {
	return m.name
}

func (m *BaseMiddleware) Priority() int {
	return m.priority
}

func (m *BaseMiddleware) Tools() []tools.Tool {
	return nil
}

func (m *BaseMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	// 默认直接调用下一层
	return handler(ctx, req)
}

func (m *BaseMiddleware) WrapToolCall(ctx context.Context, req *ToolCallRequest, handler ToolCallHandler) (*ToolCallResponse, error) {
	// 默认直接调用下一层
	return handler(ctx, req)
}

func (m *BaseMiddleware) OnAgentStart(ctx context.Context, agentID string) error {
	return nil
}

func (m *BaseMiddleware) OnAgentStop(ctx context.Context, agentID string) error {
	return nil
}
