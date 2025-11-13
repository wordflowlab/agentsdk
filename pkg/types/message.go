package types

// Role 定义消息角色
type Role string

const (
	// RoleUser 用户角色
	RoleUser Role = "user"

	// RoleAssistant AI助手角色
	RoleAssistant Role = "assistant"

	// RoleSystem 系统角色
	RoleSystem Role = "system"

	// RoleTool 工具角色
	RoleTool Role = "tool"
)

// Message 表示一条消息
type Message struct {
	// Role 消息角色
	Role Role `json:"role"`

	// Content 消息内容
	Content string `json:"content"`

	// Name 可选的名称字段（用于function/tool角色）
	Name string `json:"name,omitempty"`

	// ToolCalls 工具调用列表（仅assistant角色）
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// ToolCallID 工具调用ID（仅tool角色）
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ToolCall 表示一个工具调用
type ToolCall struct {
	// ID 工具调用的唯一标识符
	ID string `json:"id"`

	// Type 工具类型，通常为 "function"
	Type string `json:"type,omitempty"`

	// Name 工具名称
	Name string `json:"name"`

	// Arguments 工具参数（JSON对象）
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolResult 表示工具执行结果
type ToolResult struct {
	// ToolCallID 关联的工具调用ID
	ToolCallID string `json:"tool_call_id"`

	// Content 工具执行结果
	Content string `json:"content"`

	// Error 错误信息（如果有）
	Error string `json:"error,omitempty"`
}

// Bookmark 表示事件流的书签位置
type Bookmark struct {
	// Cursor 游标位置
	Cursor int64 `json:"cursor"`

	// Timestamp 时间戳
	Timestamp int64 `json:"timestamp,omitempty"`
}

// ToolCallSnapshot 工具调用快照
type ToolCallSnapshot struct {
	// ID 工具调用ID
	ID string `json:"id"`

	// Name 工具名称
	Name string `json:"name"`

	// Arguments 工具参数
	Arguments map[string]interface{} `json:"arguments,omitempty"`

	// Result 工具执行结果
	Result interface{} `json:"result,omitempty"`

	// Error 错误信息
	Error string `json:"error,omitempty"`
}

// AgentRuntimeState Agent运行时状态
type AgentRuntimeState string

const (
	// StateIdle Agent空闲
	StateIdle AgentRuntimeState = "idle"

	// StateRunning Agent运行中
	StateRunning AgentRuntimeState = "running"

	// StatePaused Agent暂停
	StatePaused AgentRuntimeState = "paused"

	// StateCompleted Agent完成
	StateCompleted AgentRuntimeState = "completed"

	// StateFailed Agent失败
	StateFailed AgentRuntimeState = "failed"
)

// BreakpointState 断点状态
type BreakpointState struct {
	// Enabled 是否启用
	Enabled bool `json:"enabled"`

	// Condition 断点条件
	Condition string `json:"condition,omitempty"`

	// HitCount 命中次数
	HitCount int `json:"hit_count,omitempty"`
}
