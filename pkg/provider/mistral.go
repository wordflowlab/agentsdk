package provider

import (
	"github.com/wordflowlab/agentsdk/pkg/types"
)

const (
	// MistralAPIBaseURL Mistral API 基础 URL
	MistralAPIBaseURL = "https://api.mistral.ai/v1"
)

// MistralProvider Mistral 提供商
// Mistral AI 是欧洲领先的 AI 公司，提供高质量的开源和商业模型
type MistralProvider struct {
	*OpenAICompatibleProvider
}

// NewMistralProvider 创建 Mistral 提供商
func NewMistralProvider(config *types.ModelConfig) (Provider, error) {
	// Mistral 配置选项
	options := &OpenAICompatibleOptions{
		RequireAPIKey:      true,
		DefaultModel:       "mistral-large-latest", // 默认使用最新的 large 模型
		SupportReasoning:   true,                   // 支持推理模式
		SupportPromptCache: false,                  // Mistral 暂不支持 Prompt Caching
		SupportVision:      true,                   // Pixtral 模型支持视觉
		SupportAudio:       false,
	}

	// 创建 OpenAI 兼容 Provider
	baseProvider, err := NewOpenAICompatibleProvider(
		config,
		MistralAPIBaseURL,
		"Mistral",
		options,
	)
	if err != nil {
		return nil, err
	}

	return &MistralProvider{
		OpenAICompatibleProvider: baseProvider,
	}, nil
}

// Capabilities 返回 Mistral 的能力
func (p *MistralProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       true,  // Pixtral 模型
		SupportAudio:        false,
		SupportReasoning:    true,  // 原生推理支持
		SupportPromptCache:  false,
		SupportJSONMode:     true, // response_format: json
		SupportFunctionCall: true,
		MaxTokens:           128000, // 128K context
		ToolCallingFormat:   "openai",
	}
}

// MistralFactory Mistral 工厂
type MistralFactory struct{}

// Create 创建 Mistral 提供商
func (f *MistralFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewMistralProvider(config)
}
