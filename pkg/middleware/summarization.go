package middleware

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/wordflowlab/agentsdk/pkg/types"
)

// SummarizationMiddleware 自动总结对话历史以管理上下文窗口
// 功能:
// 1. 监控消息历史的 token 数量
// 2. 当超过阈值时,自动总结旧消息
// 3. 保留最近的 N 条消息
// 4. 用总结消息替换旧的历史记录
type SummarizationMiddleware struct {
	*BaseMiddleware
	maxTokensBeforeSummary int
	messagesToKeep         int
	summaryPrefix          string
	tokenCounter           TokenCounterFunc
	summarizer             SummarizerFunc
	summarizationCount     int // 统计总结触发次数
}

// TokenCounterFunc 自定义 token 计数函数类型
type TokenCounterFunc func(messages []types.Message) int

// SummarizerFunc 总结生成函数类型
// 接收要总结的消息列表,返回总结内容字符串
type SummarizerFunc func(ctx context.Context, messages []types.Message) (string, error)

// SummarizationMiddlewareConfig 配置
type SummarizationMiddlewareConfig struct {
	Summarizer             SummarizerFunc   // 用于生成总结的函数
	MaxTokensBeforeSummary int              // 触发总结的 token 阈值
	MessagesToKeep         int              // 总结后保留的最近消息数量
	SummaryPrefix          string           // 总结消息的前缀
	TokenCounter           TokenCounterFunc // 自定义 token 计数器
}

// NewSummarizationMiddleware 创建中间件
func NewSummarizationMiddleware(config *SummarizationMiddlewareConfig) (*SummarizationMiddleware, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.MaxTokensBeforeSummary <= 0 {
		config.MaxTokensBeforeSummary = 170000
	}

	if config.MessagesToKeep <= 0 {
		config.MessagesToKeep = 6
	}

	if config.SummaryPrefix == "" {
		config.SummaryPrefix = "## Previous conversation summary:"
	}

	if config.TokenCounter == nil {
		config.TokenCounter = defaultTokenCounter
	}

	if config.Summarizer == nil {
		config.Summarizer = defaultSummarizer
	}

	m := &SummarizationMiddleware{
		BaseMiddleware:         NewBaseMiddleware("summarization", 40),
		maxTokensBeforeSummary: config.MaxTokensBeforeSummary,
		messagesToKeep:         config.MessagesToKeep,
		summaryPrefix:          config.SummaryPrefix,
		tokenCounter:           config.TokenCounter,
		summarizer:             config.Summarizer,
		summarizationCount:     0,
	}

	log.Printf("[SummarizationMiddleware] Initialized (max_tokens: %d, keep_messages: %d)",
		config.MaxTokensBeforeSummary, config.MessagesToKeep)
	return m, nil
}

// WrapModelCall 包装模型调用,在调用前检查是否需要总结
func (m *SummarizationMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	messages := req.Messages
	if len(messages) == 0 {
		return handler(ctx, req)
	}

	// 计算当前消息的 token 数
	totalTokens := m.tokenCounter(messages)

	log.Printf("[SummarizationMiddleware] Current tokens: %d (threshold: %d)",
		totalTokens, m.maxTokensBeforeSummary)

	// 如果未超过阈值,直接返回
	if totalTokens <= m.maxTokensBeforeSummary {
		return handler(ctx, req)
	}

	log.Printf("[SummarizationMiddleware] Token threshold exceeded, triggering summarization...")

	// 分离 system messages 和其他消息
	var systemMessages []types.Message
	var regularMessages []types.Message

	for _, msg := range messages {
		if msg.Role == types.MessageRoleSystem {
			systemMessages = append(systemMessages, msg)
		} else {
			regularMessages = append(regularMessages, msg)
		}
	}

	// 如果常规消息少于或等于要保留的数量,不进行总结
	if len(regularMessages) <= m.messagesToKeep {
		log.Printf("[SummarizationMiddleware] Not enough messages to summarize (have %d, keep %d)",
			len(regularMessages), m.messagesToKeep)
		return handler(ctx, req)
	}

	// 计算要总结的消息
	numToSummarize := len(regularMessages) - m.messagesToKeep
	messagesToSummarize := regularMessages[:numToSummarize]
	messagesToKeep := regularMessages[numToSummarize:]

	log.Printf("[SummarizationMiddleware] Summarizing %d messages, keeping %d recent messages",
		numToSummarize, m.messagesToKeep)

	// 生成总结
	summary, err := m.summarizer(ctx, messagesToSummarize)
	if err != nil {
		log.Printf("[SummarizationMiddleware] Failed to generate summary: %v, keeping original messages", err)
		return handler(ctx, req) // 失败时保留原始消息
	}

	log.Printf("[SummarizationMiddleware] Summary generated (%d chars)", len(summary))

	// 构建新的消息列表: system messages + 总结消息 + 保留的最近消息
	newMessages := make([]types.Message, 0, len(systemMessages)+1+len(messagesToKeep))
	newMessages = append(newMessages, systemMessages...)
	newMessages = append(newMessages, types.Message{
		Role: types.MessageRoleSystem,
		ContentBlocks: []types.ContentBlock{
			&types.TextBlock{
				Text: fmt.Sprintf("%s\n\n%s", m.summaryPrefix, summary),
			},
		},
	})
	newMessages = append(newMessages, messagesToKeep...)

	// 更新请求的消息
	req.Messages = newMessages
	m.summarizationCount++

	log.Printf("[SummarizationMiddleware] Summarization complete. Messages: %d -> %d (total summarizations: %d)",
		len(messages), len(newMessages), m.summarizationCount)

	return handler(ctx, req)
}

// defaultSummarizer 默认的总结生成器
// 这是一个简单的实现,实际使用时应该提供自定义的 summarizer 函数调用真实的 LLM
func defaultSummarizer(ctx context.Context, messages []types.Message) (string, error) {
	// 默认实现: 提取关键信息并格式化
	var summary strings.Builder
	summary.WriteString("Summary of previous conversation:\n\n")

	// 统计消息数量
	summary.WriteString(fmt.Sprintf("- Total messages: %d\n", len(messages)))

	// 提取关键内容 (前 3 条和后 3 条)
	numToShow := 3
	if len(messages) <= numToShow*2 {
		numToShow = len(messages) / 2
	}

	if numToShow > 0 {
		summary.WriteString("\nEarly conversation:\n")
		for i := 0; i < numToShow && i < len(messages); i++ {
			msg := messages[i]
			content := extractMessageContent(msg)
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			summary.WriteString(fmt.Sprintf("- [%s]: %s\n", msg.Role, content))
		}

		if len(messages) > numToShow*2 {
			summary.WriteString(fmt.Sprintf("\n... (%d messages omitted) ...\n", len(messages)-numToShow*2))
		}

		if len(messages) > numToShow {
			summary.WriteString("\nRecent conversation:\n")
			start := len(messages) - numToShow
			for i := start; i < len(messages); i++ {
				msg := messages[i]
				content := extractMessageContent(msg)
				if len(content) > 100 {
					content = content[:100] + "..."
				}
				summary.WriteString(fmt.Sprintf("- [%s]: %s\n", msg.Role, content))
			}
		}
	}

	return summary.String(), nil
}

// defaultTokenCounter 默认的 token 计数器(基于字符数估算)
// 粗略估算: 4 个字符约等于 1 个 token
func defaultTokenCounter(messages []types.Message) int {
	totalChars := 0
	for _, msg := range messages {
		// 计算 role 的字符数
		totalChars += len(string(msg.Role))

		// 计算内容块的字符数
		for _, block := range msg.ContentBlocks {
			switch b := block.(type) {
			case *types.TextBlock:
				totalChars += len(b.Text)
			case *types.ToolUseBlock:
				totalChars += len(b.Name)
				// 估算 input 的大小
				totalChars += len(fmt.Sprintf("%v", b.Input))
			case *types.ToolResultBlock:
				totalChars += len(fmt.Sprintf("%v", b.Content))
			}
		}
	}
	// 4 字符 ≈ 1 token
	return totalChars / 4
}

// extractMessageContent 提取消息的文本内容
func extractMessageContent(msg types.Message) string {
	var parts []string
	for _, block := range msg.ContentBlocks {
		switch b := block.(type) {
		case *types.TextBlock:
			parts = append(parts, b.Text)
		case *types.ToolUseBlock:
			parts = append(parts, fmt.Sprintf("[Tool: %s]", b.Name))
		case *types.ToolResultBlock:
			parts = append(parts, fmt.Sprintf("[ToolResult: %v]", b.Content))
		}
	}
	return strings.Join(parts, " ")
}

// GetSummarizationCount 获取总结触发次数
func (m *SummarizationMiddleware) GetSummarizationCount() int {
	return m.summarizationCount
}

// ResetSummarizationCount 重置计数器
func (m *SummarizationMiddleware) ResetSummarizationCount() {
	m.summarizationCount = 0
	log.Printf("[SummarizationMiddleware] Summarization count reset")
}

// estimateTokens 估算消息的 token 数量(调试用)
func (m *SummarizationMiddleware) estimateTokens(messages []types.Message) int {
	return m.tokenCounter(messages)
}

// GetConfig 获取当前配置
func (m *SummarizationMiddleware) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"max_tokens_before_summary": m.maxTokensBeforeSummary,
		"messages_to_keep":          m.messagesToKeep,
		"summary_prefix":            m.summaryPrefix,
		"summarization_count":       m.summarizationCount,
	}
}

// UpdateConfig 动态更新配置
func (m *SummarizationMiddleware) UpdateConfig(maxTokens, messagesToKeep int) {
	if maxTokens > 0 {
		m.maxTokensBeforeSummary = maxTokens
	}
	if messagesToKeep > 0 {
		m.messagesToKeep = messagesToKeep
	}
	log.Printf("[SummarizationMiddleware] Config updated (max_tokens: %d, keep_messages: %d)",
		m.maxTokensBeforeSummary, m.messagesToKeep)
}
