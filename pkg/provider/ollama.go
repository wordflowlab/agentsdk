package provider

import (
	"github.com/wordflowlab/agentsdk/pkg/types"
)

const (
	// OllamaDefaultBaseURL Ollama 默认基础 URL
	OllamaDefaultBaseURL = "http://localhost:11434/v1"
)

// OllamaProvider Ollama 提供商
// Ollama 是本地 LLM 部署的首选方案，支持多种开源模型
type OllamaProvider struct {
	*OpenAICompatibleProvider
}

// NewOllamaProvider 创建 Ollama 提供商
func NewOllamaProvider(config *types.ModelConfig) (Provider, error) {
	// 使用配置中的 BaseURL，或使用默认值
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = OllamaDefaultBaseURL
	}

	// Ollama 配置选项
	options := &OpenAICompatibleOptions{
		RequireAPIKey:      false, // Ollama 不需要 API Key
		DefaultModel:       "llama3.2",
		SupportReasoning:   false,
		SupportPromptCache: false,
		SupportVision:      true,  // Ollama 支持 vision 模型
		SupportAudio:       false,
	}

	// 创建 OpenAI 兼容 Provider
	baseProvider, err := NewOpenAICompatibleProvider(
		config,
		baseURL,
		"Ollama",
		options,
	)
	if err != nil {
		return nil, err
	}

	return &OllamaProvider{
		OpenAICompatibleProvider: baseProvider,
	}, nil
}

// Capabilities 返回 Ollama 的能力
func (p *OllamaProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       true, // 部分模型支持
		SupportAudio:        false,
		SupportReasoning:    false,
		SupportPromptCache:  false,
		SupportJSONMode:     true,
		SupportFunctionCall: true,
		MaxTokens:           128000, // 取决于具体模型
		ToolCallingFormat:   "openai",
	}
}

// OllamaFactory Ollama 工厂
type OllamaFactory struct{}

// Create 创建 Ollama 提供商
func (f *OllamaFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewOllamaProvider(config)
}
