package middleware

import (
	"context"
	"sort"

	"github.com/wordflowlab/agentsdk/pkg/tools"
)

// Stack 中间件栈
// 管理多个中间件,构建洋葱模型的调用链
type Stack struct {
	middlewares []Middleware
}

// NewStack 创建中间件栈
func NewStack(middlewares []Middleware) *Stack {
	// 按优先级排序(优先级数值越小越先执行)
	sorted := make([]Middleware, len(middlewares))
	copy(sorted, middlewares)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority() < sorted[j].Priority()
	})

	return &Stack{
		middlewares: sorted,
	}
}

// Tools 收集所有中间件提供的工具
func (s *Stack) Tools() []tools.Tool {
	var allTools []tools.Tool
	for _, m := range s.middlewares {
		if t := m.Tools(); t != nil {
			allTools = append(allTools, t...)
		}
	}
	return allTools
}

// ExecuteModelCall 执行模型调用(通过中间件栈)
func (s *Stack) ExecuteModelCall(
	ctx context.Context,
	req *ModelRequest,
	finalHandler ModelCallHandler,
) (*ModelResponse, error) {
	// 构建中间件链
	handler := finalHandler
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		m := s.middlewares[i]
		currentHandler := handler
		handler = func(ctx context.Context, req *ModelRequest) (*ModelResponse, error) {
			return m.WrapModelCall(ctx, req, currentHandler)
		}
	}

	return handler(ctx, req)
}

// ExecuteToolCall 执行工具调用(通过中间件栈)
func (s *Stack) ExecuteToolCall(
	ctx context.Context,
	req *ToolCallRequest,
	finalHandler ToolCallHandler,
) (*ToolCallResponse, error) {
	// 构建中间件链
	handler := finalHandler
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		m := s.middlewares[i]
		currentHandler := handler
		handler = func(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
			return m.WrapToolCall(ctx, req, currentHandler)
		}
	}

	return handler(ctx, req)
}

// OnAgentStart 通知所有中间件 Agent 启动
func (s *Stack) OnAgentStart(ctx context.Context, agentID string) error {
	for _, m := range s.middlewares {
		if err := m.OnAgentStart(ctx, agentID); err != nil {
			return err
		}
	}
	return nil
}

// OnAgentStop 通知所有中间件 Agent 停止
func (s *Stack) OnAgentStop(ctx context.Context, agentID string) error {
	// 逆序通知(LIFO)
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		if err := s.middlewares[i].OnAgentStop(ctx, agentID); err != nil {
			return err
		}
	}
	return nil
}

// Middlewares 返回中间件列表
func (s *Stack) Middlewares() []Middleware {
	return s.middlewares
}
