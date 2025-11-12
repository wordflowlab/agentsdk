package provider

import (
	"context"

	"github.com/wordflowlab/agentsdk/pkg/types"
)

// StreamChunk 流式响应块
type StreamChunk struct {
	Type  string      // "content_block_start", "content_block_delta", "content_block_stop", "message_delta"
	Index int         // 内容块索引
	Delta interface{} // 增量数据
	Usage *TokenUsage // Token使用情况
}

// TokenUsage Token使用统计
type TokenUsage struct {
	InputTokens  int64
	OutputTokens int64
}

// StreamOptions 流式请求选项
type StreamOptions struct {
	Tools       []ToolSchema
	MaxTokens   int
	Temperature float64
	System      string
}

// CompleteResponse 完整响应
type CompleteResponse struct {
	Message types.Message
	Usage   *TokenUsage
}

// ToolSchema 工具Schema
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ProviderCapabilities 模型能力
type ProviderCapabilities struct {
	// 模型能力
	SupportToolCalling  bool // 是否支持工具调用
	SupportSystemPrompt bool // 是否支持独立 system prompt
	SupportStreaming    bool // 是否支持流式输出
	SupportVision       bool // 是否支持视觉

	// 限制
	MaxTokens       int // 最大 token 数
	MaxToolsPerCall int // 单次最多调用工具数

	// Tool Calling 格式
	ToolCallingFormat string // "anthropic" | "openai" | "qwen" | "custom"
}

// Provider 模型提供商接口
type Provider interface {
	// Stream 流式对话
	Stream(ctx context.Context, messages []types.Message, opts *StreamOptions) (<-chan StreamChunk, error)

	// Complete 非流式对话(阻塞式,返回完整响应)
	Complete(ctx context.Context, messages []types.Message, opts *StreamOptions) (*CompleteResponse, error)

	// Config 返回配置
	Config() *types.ModelConfig

	// Capabilities 返回模型能力
	Capabilities() ProviderCapabilities

	// SetSystemPrompt 设置系统提示词
	SetSystemPrompt(prompt string) error

	// GetSystemPrompt 获取系统提示词
	GetSystemPrompt() string

	// Close 关闭连接
	Close() error
}

// Factory 模型提供商工厂
type Factory interface {
	Create(config *types.ModelConfig) (Provider, error)
}
