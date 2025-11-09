package middleware

import (
	"context"
	"fmt"
	"log"

	"github.com/wordflowlab/agentsdk/pkg/backends"
)

const (
	// AGENT_MEMORY_FILE_PATH Agent 记忆文件的默认路径
	AGENT_MEMORY_FILE_PATH = "/agent.md"
)

// AgentMemoryMiddleware Agent 记忆中间件
// 功能:
// 1. 从 backend 加载 agent.md 文件内容
// 2. 将内容注入到 system prompt 开头
// 3. 提供长期记忆使用指南
type AgentMemoryMiddleware struct {
	*BaseMiddleware
	backend              backends.BackendProtocol
	memoryPath           string
	systemPromptTemplate string
	memoryLoaded         bool
	memoryContent        string
}

// AgentMemoryMiddlewareConfig 配置
type AgentMemoryMiddlewareConfig struct {
	Backend              backends.BackendProtocol // 存储后端
	MemoryPath           string                   // 长期记忆路径前缀,如 "/memories/"
	SystemPromptTemplate string                   // 可选,自定义模板
}

// NewAgentMemoryMiddleware 创建中间件
func NewAgentMemoryMiddleware(config *AgentMemoryMiddlewareConfig) (*AgentMemoryMiddleware, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Backend == nil {
		return nil, fmt.Errorf("Backend is required")
	}

	if config.MemoryPath == "" {
		config.MemoryPath = "/memories/"
	}

	if config.SystemPromptTemplate == "" {
		config.SystemPromptTemplate = "<agent_memory>\n%s\n</agent_memory>"
	}

	m := &AgentMemoryMiddleware{
		BaseMiddleware:       NewBaseMiddleware("agent_memory", 5), // 高优先级,早期执行
		backend:              config.Backend,
		memoryPath:           config.MemoryPath,
		systemPromptTemplate: config.SystemPromptTemplate,
		memoryLoaded:         false,
		memoryContent:        "",
	}

	log.Printf("[AgentMemoryMiddleware] Initialized (memory_path: %s)", config.MemoryPath)
	return m, nil
}

// OnAgentStart Agent 启动时加载记忆
func (m *AgentMemoryMiddleware) OnAgentStart(ctx context.Context, agentID string) error {
	// 仅在首次加载
	if m.memoryLoaded {
		return nil
	}

	log.Printf("[AgentMemoryMiddleware] Loading agent memory from %s", AGENT_MEMORY_FILE_PATH)

	// 从 backend 读取 agent.md
	content, err := m.backend.Read(ctx, AGENT_MEMORY_FILE_PATH, 0, 0)
	if err != nil {
		// 文件不存在时,记录警告但不返回错误
		log.Printf("[AgentMemoryMiddleware] Failed to load agent memory: %v (will use empty memory)", err)
		m.memoryContent = ""
	} else {
		m.memoryContent = content
		log.Printf("[AgentMemoryMiddleware] Agent memory loaded (%d chars)", len(content))
	}

	m.memoryLoaded = true
	return nil
}

// WrapModelCall 包装模型调用,注入记忆到 system prompt
func (m *AgentMemoryMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	// 如果还没加载记忆,先加载
	if !m.memoryLoaded {
		err := m.OnAgentStart(ctx, "default")
		if err != nil {
			log.Printf("[AgentMemoryMiddleware] Failed to load memory: %v", err)
		}
	}

	// 保存原始 system prompt
	originalSystemPrompt := req.SystemPrompt

	// 构建记忆部分
	memorySection := fmt.Sprintf(m.systemPromptTemplate, m.memoryContent)

	// 注入到 system prompt 开头
	if originalSystemPrompt != "" {
		req.SystemPrompt = memorySection + "\n\n" + originalSystemPrompt
	} else {
		req.SystemPrompt = memorySection
	}

	// 追加长期记忆使用文档
	longTermMemoryPrompt := m.buildLongTermMemoryPrompt()
	req.SystemPrompt = req.SystemPrompt + "\n\n" + longTermMemoryPrompt

	log.Printf("[AgentMemoryMiddleware] Injected agent memory into system prompt (%d chars total)",
		len(req.SystemPrompt))

	// 调用处理器
	resp, err := handler(ctx, req)

	// 恢复原始 system prompt,避免在重用请求对象时累积
	req.SystemPrompt = originalSystemPrompt

	return resp, err
}

// buildLongTermMemoryPrompt 构建长期记忆使用指南
func (m *AgentMemoryMiddleware) buildLongTermMemoryPrompt() string {
	exampleBlock := fmt.Sprintf(`# List available memory files
ls %s

# Read specific memory
read_file '%sagent.md'

# Update core agent memory
edit_file('%sagent.md', old_content, new_content)

# Create project-specific memory
write_file('%sproject_context.md', content)`,
		m.memoryPath,
		m.memoryPath,
		AGENT_MEMORY_FILE_PATH,
		m.memoryPath,
	)

	return fmt.Sprintf(`## Long-term Memory

You have access to a long-term memory system that allows you to remember information across sessions.

### When to Check Memory

- At the start of a new session
- Before answering questions about previous work
- When the user asks about past conversations or preferences

### Memory-First Response Pattern

When a user asks a question:
1. Check if there's relevant memory by listing files in %s
2. Read relevant memory files if they exist
3. Base your response on the memory when available
4. Fall back to general knowledge only if no relevant memory exists

### When to Update Memory

Update memory IMMEDIATELY when:
- The user describes who they are or how they want you to behave
- The user gives feedback on your work quality or style
- The user explicitly asks you to remember something
- You notice patterns in the user's preferences or workflow

### Learning from Feedback

Every correction is an opportunity for permanent improvement:
- Capture the "why" behind corrections, not just the "what"
- Extract underlying principles from specific examples
- Update memory to prevent repeating the same mistake

### Memory Storage

- %s: Core instructions and behavioral patterns about the agent
- Other files in %s: Project-specific context, preferences, and learnings

### Usage Examples

%s

Remember: Memory enables continuity and personalization across sessions. Use it proactively to provide a better experience.`,
		m.memoryPath,
		AGENT_MEMORY_FILE_PATH,
		m.memoryPath,
		exampleBlock,
	)
}

// GetMemoryContent 获取当前加载的记忆内容
func (m *AgentMemoryMiddleware) GetMemoryContent() string {
	return m.memoryContent
}

// IsMemoryLoaded 检查记忆是否已加载
func (m *AgentMemoryMiddleware) IsMemoryLoaded() bool {
	return m.memoryLoaded
}

// ReloadMemory 重新加载记忆(用于动态更新)
func (m *AgentMemoryMiddleware) ReloadMemory(ctx context.Context) error {
	m.memoryLoaded = false
	return m.OnAgentStart(ctx, "reload")
}

// GetConfig 获取配置信息
func (m *AgentMemoryMiddleware) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"memory_path":     m.memoryPath,
		"memory_loaded":   m.memoryLoaded,
		"memory_size":     len(m.memoryContent),
		"memory_file":     AGENT_MEMORY_FILE_PATH,
	}
}
