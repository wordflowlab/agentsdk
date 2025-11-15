package context

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

// TokenCounter 提供 Token 计数功能
// 用于估算不同 LLM 模型的 Token 使用量
type TokenCounter interface {
	// Count 计算单个文本的 Token 数量
	Count(ctx context.Context, text string) (int, error)

	// CountBatch 批量计算多个文本的 Token 数量
	CountBatch(ctx context.Context, texts []string) ([]int, error)

	// EstimateMessages 估算消息列表的总 Token 数
	EstimateMessages(ctx context.Context, messages []Message) (int, error)

	// ModelName 返回此计数器对应的模型名称
	ModelName() string
}

// Message 表示一条消息
type Message struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// ModelConfig 定义模型的 Token 计算配置
type ModelConfig struct {
	Name              string  // 模型名称
	CharsPerToken     float64 // 平均每个 Token 的字符数
	BaseTokenOverhead int     // 每条消息的基础 Token 开销
	RoleTokenCost     int     // Role 标签的 Token 开销
}

// 预定义的模型配置
var (
	// === OpenAI 系列 ===
	GPT4oConfig = ModelConfig{
		Name:              "gpt-4o",
		CharsPerToken:     4.0,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	GPT4oMiniConfig = ModelConfig{
		Name:              "gpt-4o-mini",
		CharsPerToken:     4.0,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	GPT4TurboConfig = ModelConfig{
		Name:              "gpt-4-turbo",
		CharsPerToken:     4.0,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	GPT4Config = ModelConfig{
		Name:              "gpt-4",
		CharsPerToken:     4.0,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	GPT35TurboConfig = ModelConfig{
		Name:              "gpt-3.5-turbo",
		CharsPerToken:     4.0,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	// OpenAI 推理模型（o1/o3 系列）
	O1Config = ModelConfig{
		Name:              "o1",
		CharsPerToken:     4.0,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	O3Config = ModelConfig{
		Name:              "o3",
		CharsPerToken:     4.0,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	// === Anthropic/Claude 系列 ===
	ClaudeSonnet45Config = ModelConfig{
		Name:              "claude-sonnet-4-5",
		CharsPerToken:     3.5, // Claude 对中文更友好
		BaseTokenOverhead: 4,
		RoleTokenCost:     1,
	}

	Claude3OpusConfig = ModelConfig{
		Name:              "claude-3-opus",
		CharsPerToken:     3.5,
		BaseTokenOverhead: 4,
		RoleTokenCost:     1,
	}

	Claude3SonnetConfig = ModelConfig{
		Name:              "claude-3-sonnet",
		CharsPerToken:     3.5,
		BaseTokenOverhead: 4,
		RoleTokenCost:     1,
	}

	Claude3HaikuConfig = ModelConfig{
		Name:              "claude-3-haiku",
		CharsPerToken:     3.5,
		BaseTokenOverhead: 4,
		RoleTokenCost:     1,
	}

	// === Google Gemini 系列 ===
	Gemini2FlashConfig = ModelConfig{
		Name:              "gemini-2.0-flash-exp",
		CharsPerToken:     3.8,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	Gemini2ThinkingConfig = ModelConfig{
		Name:              "gemini-2.0-flash-thinking-exp",
		CharsPerToken:     3.8,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	// === 中国市场模型 ===
	// DeepSeek
	DeepSeekChatConfig = ModelConfig{
		Name:              "deepseek-chat",
		CharsPerToken:     3.2, // 对中文优化
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	// GLM/智谱
	GLM4Config = ModelConfig{
		Name:              "glm-4",
		CharsPerToken:     3.0, // 中文模型
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	GLM3TurboConfig = ModelConfig{
		Name:              "glm-3-turbo",
		CharsPerToken:     3.0,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	// Moonshot/Kimi
	Moonshot128KConfig = ModelConfig{
		Name:              "moonshot-v1-128k",
		CharsPerToken:     3.2,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	Moonshot32KConfig = ModelConfig{
		Name:              "moonshot-v1-32k",
		CharsPerToken:     3.2,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	// === 开源模型 ===
	// Llama 系列
	Llama3Config = ModelConfig{
		Name:              "llama-3",
		CharsPerToken:     4.0,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	Llama370BConfig = ModelConfig{
		Name:              "llama-3.3-70b",
		CharsPerToken:     4.0,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	// Qwen 系列
	Qwen2Config = ModelConfig{
		Name:              "qwen-2",
		CharsPerToken:     3.0, // 中文优化
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}

	// === 通用配置（保守估算）===
	DefaultConfig = ModelConfig{
		Name:              "default",
		CharsPerToken:     4.0,
		BaseTokenOverhead: 3,
		RoleTokenCost:     1,
	}
)

// SimpleTokenCounter 简单的基于字符的 Token 计数器
// 使用启发式规则估算 Token 数量，无需调用外部 API
type SimpleTokenCounter struct {
	config ModelConfig
}

// NewSimpleTokenCounter 创建一个简单的 Token 计数器
func NewSimpleTokenCounter(config ModelConfig) *SimpleTokenCounter {
	return &SimpleTokenCounter{
		config: config,
	}
}

// NewGPT4Counter 创建 GPT-4 Token 计数器
func NewGPT4Counter() *SimpleTokenCounter {
	return NewSimpleTokenCounter(GPT4Config)
}

// NewClaudeCounter 创建 Claude Token 计数器
func NewClaudeCounter() *SimpleTokenCounter {
	return NewSimpleTokenCounter(ClaudeSonnet45Config)
}

// Count 实现 TokenCounter 接口
func (c *SimpleTokenCounter) Count(ctx context.Context, text string) (int, error) {
	if text == "" {
		return 0, nil
	}

	// 基于字符数估算 Token
	charCount := utf8.RuneCountInString(text)
	tokenCount := int(float64(charCount) / c.config.CharsPerToken)

	// 至少为 1 个 Token
	if tokenCount == 0 && charCount > 0 {
		tokenCount = 1
	}

	return tokenCount, nil
}

// CountBatch 实现 TokenCounter 接口
func (c *SimpleTokenCounter) CountBatch(ctx context.Context, texts []string) ([]int, error) {
	if len(texts) == 0 {
		return []int{}, nil
	}

	counts := make([]int, len(texts))
	for i, text := range texts {
		count, err := c.Count(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to count text at index %d: %w", i, err)
		}
		counts[i] = count
	}

	return counts, nil
}

// EstimateMessages 实现 TokenCounter 接口
func (c *SimpleTokenCounter) EstimateMessages(ctx context.Context, messages []Message) (int, error) {
	if len(messages) == 0 {
		return 0, nil
	}

	totalTokens := 0

	for _, msg := range messages {
		// 计算内容的 Token 数
		contentTokens, err := c.Count(ctx, msg.Content)
		if err != nil {
			return 0, fmt.Errorf("failed to count message content: %w", err)
		}

		// 添加消息开销
		messageTokens := contentTokens + c.config.BaseTokenOverhead + c.config.RoleTokenCost

		totalTokens += messageTokens
	}

	return totalTokens, nil
}

// ModelName 实现 TokenCounter 接口
func (c *SimpleTokenCounter) ModelName() string {
	return c.config.Name
}

// TikTokenCounter 使用 tiktoken 库的精确计数器（可选）
// 注意: 这需要外部依赖，暂时作为接口定义
type TikTokenCounter struct {
	modelName string
	// encoder tiktoken.Encoder // 需要引入 tiktoken 库
}

// TokenEstimate 包含详细的 Token 估算信息
type TokenEstimate struct {
	TotalTokens   int            // 总 Token 数
	MessageTokens []int          // 每条消息的 Token 数
	Breakdown     map[string]int // 详细分解（content, overhead, role 等）
}

// DetailedTokenCounter 提供详细 Token 估算的计数器
type DetailedTokenCounter struct {
	baseCounter TokenCounter
}

// NewDetailedTokenCounter 创建详细计数器
func NewDetailedTokenCounter(baseCounter TokenCounter) *DetailedTokenCounter {
	return &DetailedTokenCounter{
		baseCounter: baseCounter,
	}
}

// EstimateMessagesDetailed 返回详细的 Token 估算
func (c *DetailedTokenCounter) EstimateMessagesDetailed(ctx context.Context, messages []Message) (*TokenEstimate, error) {
	if len(messages) == 0 {
		return &TokenEstimate{
			TotalTokens:   0,
			MessageTokens: []int{},
			Breakdown:     map[string]int{},
		}, nil
	}

	estimate := &TokenEstimate{
		MessageTokens: make([]int, len(messages)),
		Breakdown:     make(map[string]int),
	}

	for i, msg := range messages {
		contentTokens, err := c.baseCounter.Count(ctx, msg.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to count message %d: %w", i, err)
		}

		// 记录分解信息
		estimate.Breakdown["content"] += contentTokens
		estimate.MessageTokens[i] = contentTokens
	}

	// 计算总数
	estimate.TotalTokens, _ = c.baseCounter.EstimateMessages(ctx, messages)

	// 计算开销
	estimate.Breakdown["overhead"] = estimate.TotalTokens - estimate.Breakdown["content"]

	return estimate, nil
}

// MultiModelTokenCounter 支持多模型的 Token 计数器
type MultiModelTokenCounter struct {
	counters map[string]TokenCounter
	default_ TokenCounter
}

// NewMultiModelTokenCounter 创建多模型计数器
func NewMultiModelTokenCounter() *MultiModelTokenCounter {
	return &MultiModelTokenCounter{
		counters: make(map[string]TokenCounter),
		default_: NewSimpleTokenCounter(DefaultConfig),
	}
}

// RegisterCounter 注册一个模型的计数器
func (m *MultiModelTokenCounter) RegisterCounter(modelName string, counter TokenCounter) {
	m.counters[modelName] = counter
}

// GetCounter 获取指定模型的计数器
func (m *MultiModelTokenCounter) GetCounter(modelName string) TokenCounter {
	if counter, ok := m.counters[modelName]; ok {
		return counter
	}

	// 尝试模糊匹配
	lowerModelName := strings.ToLower(modelName)
	for name, counter := range m.counters {
		if strings.Contains(lowerModelName, strings.ToLower(name)) {
			return counter
		}
	}

	return m.default_
}

// CountForModel 为指定模型计算 Token 数
func (m *MultiModelTokenCounter) CountForModel(ctx context.Context, modelName string, text string) (int, error) {
	counter := m.GetCounter(modelName)
	return counter.Count(ctx, text)
}

// EstimateMessagesForModel 为指定模型估算消息 Token 数
func (m *MultiModelTokenCounter) EstimateMessagesForModel(ctx context.Context, modelName string, messages []Message) (int, error) {
	counter := m.GetCounter(modelName)
	return counter.EstimateMessages(ctx, messages)
}

// TokenBudget 表示 Token 预算配置
type TokenBudget struct {
	MaxTokens     int // 最大 Token 数
	ReservedTokens int // 预留 Token 数（用于输出）
	WarningThreshold float64 // 警告阈值（0.0-1.0）
}

// DefaultTokenBudget 返回默认的 Token 预算
func DefaultTokenBudget() TokenBudget {
	return TokenBudget{
		MaxTokens:        128000, // GPT-4 Turbo 的上下文窗口
		ReservedTokens:   4096,   // 预留 4K 用于输出
		WarningThreshold: 0.8,    // 80% 时发出警告
	}
}

// AvailableTokens 返回可用的 Token 数
func (b TokenBudget) AvailableTokens() int {
	return b.MaxTokens - b.ReservedTokens
}

// IsWithinBudget 检查是否在预算内
func (b TokenBudget) IsWithinBudget(usedTokens int) bool {
	return usedTokens <= b.AvailableTokens()
}

// ShouldWarn 检查是否应该发出警告
func (b TokenBudget) ShouldWarn(usedTokens int) bool {
	threshold := int(float64(b.AvailableTokens()) * b.WarningThreshold)
	return usedTokens >= threshold
}

// RemainingTokens 返回剩余可用 Token 数
func (b TokenBudget) RemainingTokens(usedTokens int) int {
	remaining := b.AvailableTokens() - usedTokens
	if remaining < 0 {
		return 0
	}
	return remaining
}

// UsagePercentage 返回使用百分比
func (b TokenBudget) UsagePercentage(usedTokens int) float64 {
	if b.AvailableTokens() == 0 {
		return 100.0
	}
	percentage := float64(usedTokens) / float64(b.AvailableTokens()) * 100.0
	if percentage > 100.0 {
		return 100.0
	}
	return percentage
}
