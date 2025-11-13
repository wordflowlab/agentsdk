package provider

import (
	"github.com/wordflowlab/agentsdk/pkg/types"
)

const (
	// OpenRouterAPIBaseURL OpenRouter API 基础 URL
	OpenRouterAPIBaseURL = "https://openrouter.ai/api/v1"
)

// OpenRouterProvider OpenRouter 提供商
// OpenRouter 是聚合平台，支持数百个模型（OpenAI, Anthropic, Google, Meta, 等）
type OpenRouterProvider struct {
	*OpenAICompatibleProvider

	// OpenRouter 特定配置
	providerPreference string   // 优先使用的 provider
	providerOrder      []string // provider 优先级顺序
	allowFallbacks     bool     // 是否允许 fallback
}

// OpenRouterConfig OpenRouter 特定配置
type OpenRouterConfig struct {
	// 优先使用的 provider (如 "OpenAI", "Anthropic")
	ProviderPreference string

	// Provider 优先级顺序
	ProviderOrder []string

	// 是否允许 fallback 到其他 provider
	AllowFallbacks bool

	// 应用名称（用于统计）
	AppName string

	// 站点 URL（用于统计）
	SiteURL string
}

// NewOpenRouterProvider 创建 OpenRouter 提供商
func NewOpenRouterProvider(config *types.ModelConfig, orConfig *OpenRouterConfig) (Provider, error) {
	if orConfig == nil {
		orConfig = &OpenRouterConfig{
			AllowFallbacks: true,
		}
	}

	// OpenRouter 配置选项
	options := &OpenAICompatibleOptions{
		RequireAPIKey:      true,
		DefaultModel:       "openai/gpt-4o", // 默认使用 GPT-4o
		SupportReasoning:   true,            // 支持推理模型
		SupportPromptCache: true,            // 支持 Prompt Caching
		SupportVision:      true,            // 支持多模态
		SupportAudio:       true,
		CustomHeaders:      make(map[string]string),
	}

	// 添加 OpenRouter 特定请求头
	if orConfig.AppName != "" {
		options.CustomHeaders["HTTP-Referer"] = orConfig.SiteURL
		options.CustomHeaders["X-Title"] = orConfig.AppName
	}

	// 创建 OpenAI 兼容 Provider
	baseProvider, err := NewOpenAICompatibleProvider(
		config,
		OpenRouterAPIBaseURL,
		"OpenRouter",
		options,
	)
	if err != nil {
		return nil, err
	}

	return &OpenRouterProvider{
		OpenAICompatibleProvider: baseProvider,
		providerPreference:       orConfig.ProviderPreference,
		providerOrder:            orConfig.ProviderOrder,
		allowFallbacks:           orConfig.AllowFallbacks,
	}, nil
}

// NewOpenRouterProviderSimple 创建 OpenRouter 提供商（简化版）
func NewOpenRouterProviderSimple(config *types.ModelConfig) (Provider, error) {
	return NewOpenRouterProvider(config, nil)
}

// Capabilities 返回 OpenRouter 的能力
// 注意：OpenRouter 聚合多个 provider，能力取决于所选模型
func (p *OpenRouterProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       true,  // 取决于所选模型
		SupportAudio:        true,  // 取决于所选模型
		SupportReasoning:    true,  // 支持 o1/o3 等推理模型
		SupportPromptCache:  true,  // 支持 Prompt Caching
		SupportJSONMode:     true,
		SupportFunctionCall: true,
		MaxTokens:           200000, // 取决于所选模型，最高可达 200K
		ToolCallingFormat:   "openai",
		CacheMinTokens:      1024,
	}
}

// OpenRouterFactory OpenRouter 工厂
type OpenRouterFactory struct {
	Config *OpenRouterConfig
}

// Create 创建 OpenRouter 提供商
func (f *OpenRouterFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewOpenRouterProvider(config, f.Config)
}
