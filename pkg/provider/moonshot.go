package provider

import (
	"github.com/wordflowlab/agentsdk/pkg/types"
)

const (
	// MoonshotAPIBaseURL Moonshot API 基础 URL
	MoonshotAPIBaseURL = "https://api.moonshot.cn/v1"
)

// MoonshotProvider Moonshot（月之暗面 Kimi）提供商
// Moonshot AI 的 Kimi 模型以超长上下文（128K-200K）闻名
type MoonshotProvider struct {
	*OpenAICompatibleProvider
}

// NewMoonshotProvider 创建 Moonshot 提供商
func NewMoonshotProvider(config *types.ModelConfig) (Provider, error) {
	// Moonshot 配置选项
	options := &OpenAICompatibleOptions{
		RequireAPIKey:      true,
		DefaultModel:       "moonshot-v1-128k", // 默认使用 128K 上下文版本
		SupportReasoning:   false,
		SupportPromptCache: false,
		SupportVision:      false, // Moonshot 目前主要专注文本
		SupportAudio:       false,
	}

	// 创建 OpenAI 兼容 Provider
	baseProvider, err := NewOpenAICompatibleProvider(
		config,
		MoonshotAPIBaseURL,
		"Moonshot",
		options,
	)
	if err != nil {
		return nil, err
	}

	return &MoonshotProvider{
		OpenAICompatibleProvider: baseProvider,
	}, nil
}

// Capabilities 返回 Moonshot 的能力
func (p *MoonshotProvider) Capabilities() ProviderCapabilities {
	caps := ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       false,
		SupportAudio:        false,
		SupportReasoning:    false,
		SupportPromptCache:  false,
		SupportJSONMode:     true,
		SupportFunctionCall: true,
		MaxTokens:           128000, // 默认 128K
		ToolCallingFormat:   "openai",
	}

	// 根据模型调整 MaxTokens
	model := p.Config().Model
	if model == "moonshot-v1-32k" {
		caps.MaxTokens = 32000
	} else if model == "moonshot-v1-128k" {
		caps.MaxTokens = 128000
	}

	return caps
}

// MoonshotFactory Moonshot 工厂
type MoonshotFactory struct{}

// Create 创建 Moonshot 提供商
func (f *MoonshotFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewMoonshotProvider(config)
}
