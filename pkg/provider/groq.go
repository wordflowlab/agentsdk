package provider

import (
	"github.com/wordflowlab/agentsdk/pkg/types"
)

const (
	// GroqAPIBaseURL Groq API 基础 URL
	GroqAPIBaseURL = "https://api.groq.com/openai/v1"
)

// GroqProvider Groq 提供商
// Groq 提供超快速的 LLM 推理服务，完全兼容 OpenAI API
type GroqProvider struct {
	*OpenAICompatibleProvider
}

// NewGroqProvider 创建 Groq 提供商
func NewGroqProvider(config *types.ModelConfig) (Provider, error) {
	// Groq 配置选项
	options := &OpenAICompatibleOptions{
		RequireAPIKey:      true,
		DefaultModel:       "llama-3.3-70b-versatile", // 默认使用最新的 Llama 模型
		SupportReasoning:   false,
		SupportPromptCache: false,
		SupportVision:      false, // Groq 目前不支持多模态
		SupportAudio:       false,
	}

	// 创建 OpenAI 兼容 Provider
	baseProvider, err := NewOpenAICompatibleProvider(
		config,
		GroqAPIBaseURL,
		"Groq",
		options,
	)
	if err != nil {
		return nil, err
	}

	return &GroqProvider{
		OpenAICompatibleProvider: baseProvider,
	}, nil
}

// Capabilities 返回 Groq 的能力
func (p *GroqProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       false,
		SupportAudio:        false,
		SupportReasoning:    false,
		SupportPromptCache:  false,
		SupportJSONMode:     true,
		SupportFunctionCall: true,
		MaxTokens:           32768, // Groq 支持 32K context
		ToolCallingFormat:   "openai",
	}
}

// GroqFactory Groq 工厂
type GroqFactory struct{}

// Create 创建 Groq 提供商
func (f *GroqFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewGroqProvider(config)
}
