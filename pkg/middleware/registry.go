package middleware

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/wordflowlab/agentsdk/pkg/backends"
	"github.com/wordflowlab/agentsdk/pkg/memory"
	"github.com/wordflowlab/agentsdk/pkg/provider"
	"github.com/wordflowlab/agentsdk/pkg/sandbox"
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
	Sandbox      sandbox.Sandbox        // 可选: 需要访问沙箱文件系统的中间件
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

	// Filesystem Middleware (默认使用 Sandbox 文件系统)
	r.Register("filesystem", func(config *MiddlewareFactoryConfig) (Middleware, error) {
		if config.Sandbox == nil {
			return nil, fmt.Errorf("filesystem middleware requires sandbox")
		}

		fsBackend := backends.NewFilesystemBackend(config.Sandbox.FS())

		return NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
			Backend: fsBackend,
		}), nil
	})

	// AgentMemory Middleware (默认使用 Sandbox 文件系统, /memories/ 作为记忆根目录)
	r.Register("agent_memory", func(config *MiddlewareFactoryConfig) (Middleware, error) {
		if config.Sandbox == nil {
			return nil, fmt.Errorf("agent_memory middleware requires sandbox")
		}

		fsBackend := backends.NewFilesystemBackend(config.Sandbox.FS())

		memoryPath := "/memories/"
		if config.CustomConfig != nil {
			if mp, ok := config.CustomConfig["memory_path"].(string); ok && mp != "" {
				memoryPath = mp
			}
		}

		// 基础命名空间: 如果 AgentConfig.Metadata 中提供了 user_id, 则自动使用 users/<user_id>
		baseNamespace := ""
		if config.Metadata != nil {
			if userID, ok := config.Metadata["user_id"].(string); ok && userID != "" {
				baseNamespace = fmt.Sprintf("users/%s", userID)
			}
		}

		return NewAgentMemoryMiddleware(&AgentMemoryMiddlewareConfig{
			Backend:       fsBackend,
			MemoryPath:    memoryPath,
			BaseNamespace: baseNamespace,
		})
	})

	// WorkingMemory Middleware (跨会话状态管理)
	r.Register("working_memory", func(config *MiddlewareFactoryConfig) (Middleware, error) {
		if config.Sandbox == nil {
			return nil, fmt.Errorf("working_memory middleware requires sandbox")
		}

		fsBackend := backends.NewFilesystemBackend(config.Sandbox.FS())

		// 默认配置
		basePath := "/working_memory/"
		scope := "thread" // "thread" | "resource"
		experimental := false

		// 从自定义配置读取
		if config.CustomConfig != nil {
			if bp, ok := config.CustomConfig["base_path"].(string); ok && bp != "" {
				basePath = bp
			}
			if s, ok := config.CustomConfig["scope"].(string); ok && s != "" {
				scope = s
			}
			if exp, ok := config.CustomConfig["experimental"].(bool); ok {
				experimental = exp
			}
		}

		// 解析 scope
		var wmScope memory.WorkingMemoryScope
		if scope == "resource" {
			wmScope = memory.ScopeResource
		} else {
			wmScope = memory.ScopeThread
		}

		return NewWorkingMemoryMiddleware(&WorkingMemoryMiddlewareConfig{
			Backend:      fsBackend,
			BasePath:     basePath,
			Scope:        wmScope,
			Experimental: experimental,
		})
	})

	log.Printf("[MiddlewareRegistry] Built-in middlewares registered: %v", r.List())
}

// DefaultRegistry 全局默认注册表
var DefaultRegistry = NewRegistry()
