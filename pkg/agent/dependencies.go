package agent

import (
	"github.com/wordflowlab/agentsdk/pkg/provider"
	"github.com/wordflowlab/agentsdk/pkg/router"
	"github.com/wordflowlab/agentsdk/pkg/sandbox"
	"github.com/wordflowlab/agentsdk/pkg/store"
	"github.com/wordflowlab/agentsdk/pkg/tools"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// Dependencies Agent依赖
type Dependencies struct {
	Store           store.Store
	SandboxFactory  *sandbox.Factory
	ToolRegistry    *tools.Registry
	ProviderFactory provider.Factory
	// Router 为可选依赖，如果为 nil，则沿用旧的静态 ModelConfig 行为。
	Router          router.Router
	TemplateRegistry *TemplateRegistry
}

// TemplateRegistry 模板注册表
type TemplateRegistry struct {
	templates map[string]*types.AgentTemplateDefinition
}

// NewTemplateRegistry 创建模板注册表
func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{
		templates: make(map[string]*types.AgentTemplateDefinition),
	}
}

// Register 注册模板
func (tr *TemplateRegistry) Register(template *types.AgentTemplateDefinition) {
	tr.templates[template.ID] = template
}

// Get 获取模板
func (tr *TemplateRegistry) Get(id string) (*types.AgentTemplateDefinition, error) {
	template, ok := tr.templates[id]
	if !ok {
		return nil, &TemplateNotFoundError{ID: id}
	}
	return template, nil
}

// List 列出所有模板
func (tr *TemplateRegistry) List() []*types.AgentTemplateDefinition {
	templates := make([]*types.AgentTemplateDefinition, 0, len(tr.templates))
	for _, t := range tr.templates {
		templates = append(templates, t)
	}
	return templates
}

// TemplateNotFoundError 模板未找到错误
type TemplateNotFoundError struct {
	ID string
}

func (e *TemplateNotFoundError) Error() string {
	return "template not found: " + e.ID
}
