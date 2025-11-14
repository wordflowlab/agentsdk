package tools

import (
	"context"

	"github.com/wordflowlab/agentsdk/pkg/sandbox"
)

// ToolContext 工具执行上下文
type ToolContext struct {
	AgentID    string
	Sandbox    sandbox.Sandbox
	Signal     context.Context
	Emit       func(eventType string, data interface{})
	Services   map[string]interface{}
	ThreadID   string // Working Memory 会话 ID
	ResourceID string // Working Memory 资源 ID
}

// Tool 工具接口
type Tool interface {
	// Name 工具名称
	Name() string

	// Description 工具描述
	Description() string

	// InputSchema JSON Schema定义
	InputSchema() map[string]interface{}

	// Execute 执行工具
	Execute(ctx context.Context, input map[string]interface{}, tc *ToolContext) (interface{}, error)

	// Prompt 工具使用说明(可选)
	Prompt() string
}

// ToolDescriptor 工具描述符(用于持久化)
type ToolDescriptor struct {
	Name       string                 `json:"name"`
	RegistryID string                 `json:"registry_id,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
}

// ToolFactory 工具工厂函数
type ToolFactory func(config map[string]interface{}) (Tool, error)

// Registry 工具注册表
type Registry struct {
	factories map[string]ToolFactory
}

// NewRegistry 创建工具注册表
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]ToolFactory),
	}
}

// Register 注册工具
func (r *Registry) Register(name string, factory ToolFactory) {
	r.factories[name] = factory
}

// Create 创建工具实例
func (r *Registry) Create(name string, config map[string]interface{}) (Tool, error) {
	factory, ok := r.factories[name]
	if !ok {
		return nil, &ToolNotFoundError{Name: name}
	}

	return factory(config)
}

// List 列出所有已注册的工具
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// Has 检查工具是否已注册
func (r *Registry) Has(name string) bool {
	_, ok := r.factories[name]
	return ok
}

// ToolNotFoundError 工具未找到错误
type ToolNotFoundError struct {
	Name string
}

func (e *ToolNotFoundError) Error() string {
	return "tool not found: " + e.Name
}
