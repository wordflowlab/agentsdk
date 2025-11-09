package middleware

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/wordflowlab/agentsdk/pkg/provider"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// MiddlewareFactory 中间件工厂函数
// config参数可用于传递Provider等依赖
type MiddlewareFactory func(config *MiddlewareFactoryConfig) (Middleware, error)

// MiddlewareFactoryConfig 工厂配置
type MiddlewareFactoryConfig struct {
	Provider     provider.Provider
	AgentID      string
	Metadata     map[string]interface{}
	CustomConfig map[string]interface{} // 自定义配置
}

// Registry 中间件注册表
type Registry struct {
	mu        sync.RWMutex
	factories map[string]MiddlewareFactory
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	r := &Registry{
		factories: make(map[string]MiddlewareFactory),
	}
	// 注册内置中间件
	r.registerBuiltin()
	return r
}

// Register 注册中间件工厂
func (r *Registry) Register(name string, factory MiddlewareFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
	log.Printf("[MiddlewareRegistry] Registered: %s", name)
}

// Create 创建中间件实例
func (r *Registry) Create(name string, config *MiddlewareFactoryConfig) (Middleware, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("middleware not found: %s", name)
	}

	if config == nil {
		config = &MiddlewareFactoryConfig{}
	}

	return factory(config)
}

// List 列出所有已注册的中间件名称
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// registerBuiltin 注册内置中间件
func (r *Registry) registerBuiltin() {
	// Summarization Middleware
	r.Register("summarization", func(config *MiddlewareFactoryConfig) (Middleware, error) {
		if config.Provider == nil {
			return nil, fmt.Errorf("summarization middleware requires provider")
		}

		// 自定义配置(可选)
		maxTokens := 170000
		messagesToKeep := 6
		if config.CustomConfig != nil {
			if mt, ok := config.CustomConfig["max_tokens"].(int); ok {
				maxTokens = mt
			}
			if mk, ok := config.CustomConfig["messages_to_keep"].(int); ok {
				messagesToKeep = mk
			}
		}

		// 创建 summarizer 函数(使用Provider)
		summarizer := func(ctx context.Context, messages []types.Message) (string, error) {
			// 调用Provider生成总结
			// 为简化,使用默认总结器
			return defaultSummarizer(ctx, messages)
		}

		return NewSummarizationMiddleware(&SummarizationMiddlewareConfig{
			MaxTokensBeforeSummary: maxTokens,
			MessagesToKeep:         messagesToKeep,
			SummaryPrefix:          "## Previous conversation summary:",
			TokenCounter:           defaultTokenCounter,
			Summarizer:             summarizer,
		})
	})

	log.Printf("[MiddlewareRegistry] Built-in middlewares registered: %v", r.List())
}

// DefaultRegistry 全局默认注册表
var DefaultRegistry = NewRegistry()
