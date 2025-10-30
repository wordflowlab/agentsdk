package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/wordflowlab/agentsdk/pkg/types"
)

const (
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	defaultAnthropicVersion = "2023-06-01"
)

// AnthropicProvider Anthropic模型提供商
type AnthropicProvider struct {
	config       *types.ModelConfig
	client       *http.Client
	baseURL      string
	version      string
	systemPrompt string // 系统提示词
}

// NewAnthropicProvider 创建Anthropic提供商
func NewAnthropicProvider(config *types.ModelConfig) (*AnthropicProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("anthropic api key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}

	return &AnthropicProvider{
		config:  config,
		client:  &http.Client{},
		baseURL: baseURL,
		version: defaultAnthropicVersion,
	}, nil
}

// Stream 流式对话
func (ap *AnthropicProvider) Stream(ctx context.Context, messages []types.Message, opts *StreamOptions) (<-chan StreamChunk, error) {
	// 构建请求体
	reqBody := ap.buildRequest(messages, opts)

	// 序列化
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", ap.baseURL+"/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", ap.config.APIKey)
	req.Header.Set("anthropic-version", ap.version)

	// 发送请求
	resp, err := ap.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic api error: %d - %s", resp.StatusCode, string(body))
	}

	// 创建流式响应channel
	chunkCh := make(chan StreamChunk, 10)

	go ap.processStream(resp.Body, chunkCh)

	return chunkCh, nil
}

// buildRequest 构建请求体
func (ap *AnthropicProvider) buildRequest(messages []types.Message, opts *StreamOptions) map[string]interface{} {
	req := map[string]interface{}{
		"model":    ap.config.Model,
		"messages": ap.convertMessages(messages),
		"stream":   true,
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			req["max_tokens"] = opts.MaxTokens
		} else {
			req["max_tokens"] = 4096 // 默认值
		}

		if opts.Temperature > 0 {
			req["temperature"] = opts.Temperature
		}

		if opts.System != "" {
			req["system"] = opts.System
		} else if ap.systemPrompt != "" {
			// 如果 opts 没有 system，使用存储的 systemPrompt
			req["system"] = ap.systemPrompt
		}

		if len(opts.Tools) > 0 {
			req["tools"] = opts.Tools
		}
	} else {
		req["max_tokens"] = 4096
		if ap.systemPrompt != "" {
			req["system"] = ap.systemPrompt
		}
	}

	return req
}

// convertMessages 转换消息格式
func (ap *AnthropicProvider) convertMessages(messages []types.Message) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))

	for _, msg := range messages {
		// 跳过system消息(system在opts中单独传递)
		if msg.Role == types.MessageRoleSystem {
			continue
		}

		content := make([]interface{}, 0, len(msg.Content))
		for _, block := range msg.Content {
			switch b := block.(type) {
			case *types.TextBlock:
				content = append(content, map[string]interface{}{
					"type": "text",
					"text": b.Text,
				})
			case *types.ToolUseBlock:
				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    b.ID,
					"name":  b.Name,
					"input": b.Input,
				})
			case *types.ToolResultBlock:
				content = append(content, map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": b.ToolUseID,
					"content":     b.Content,
					"is_error":    b.IsError,
				})
			}
		}

		result = append(result, map[string]interface{}{
			"role":    string(msg.Role),
			"content": content,
		})
	}

	return result
}

// processStream 处理流式响应
func (ap *AnthropicProvider) processStream(body io.ReadCloser, chunkCh chan<- StreamChunk) {
	defer close(chunkCh)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()

		// SSE格式: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// 忽略特殊标记
		if data == "[DONE]" {
			break
		}

		// 解析JSON
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		chunk := ap.parseStreamEvent(event)
		if chunk != nil {
			chunkCh <- *chunk
		}
	}
}

// parseStreamEvent 解析流式事件
func (ap *AnthropicProvider) parseStreamEvent(event map[string]interface{}) *StreamChunk {
	eventType, _ := event["type"].(string)

	chunk := &StreamChunk{
		Type: eventType,
	}

	switch eventType {
	case "content_block_start":
		if index, ok := event["index"].(float64); ok {
			chunk.Index = int(index)
		}
		if contentBlock, ok := event["content_block"].(map[string]interface{}); ok {
			chunk.Delta = contentBlock
		}

	case "content_block_delta":
		if index, ok := event["index"].(float64); ok {
			chunk.Index = int(index)
		}
		if delta, ok := event["delta"].(map[string]interface{}); ok {
			chunk.Delta = delta
		}

	case "content_block_stop":
		if index, ok := event["index"].(float64); ok {
			chunk.Index = int(index)
		}

	case "message_delta":
		if delta, ok := event["delta"].(map[string]interface{}); ok {
			chunk.Delta = delta
		}
		if usage, ok := event["usage"].(map[string]interface{}); ok {
			chunk.Usage = &TokenUsage{
				InputTokens:  int64(usage["input_tokens"].(float64)),
				OutputTokens: int64(usage["output_tokens"].(float64)),
			}
		}
	}

	return chunk
}

// Config 返回配置
func (ap *AnthropicProvider) Config() *types.ModelConfig {
	return ap.config
}

// Capabilities 返回模型能力
func (ap *AnthropicProvider) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		SupportToolCalling:  true,
		SupportSystemPrompt: true,
		SupportStreaming:    true,
		SupportVision:       false, // 根据模型决定
		MaxTokens:           200000,
		MaxToolsPerCall:     0, // 无限制
		ToolCallingFormat:   "anthropic",
	}
}

// SetSystemPrompt 设置系统提示词
func (ap *AnthropicProvider) SetSystemPrompt(prompt string) error {
	ap.systemPrompt = prompt
	return nil
}

// GetSystemPrompt 获取系统提示词
func (ap *AnthropicProvider) GetSystemPrompt() string {
	return ap.systemPrompt
}

// Close 关闭连接
func (ap *AnthropicProvider) Close() error {
	// HTTP客户端不需要显式关闭
	return nil
}

// AnthropicFactory Anthropic工厂
type AnthropicFactory struct{}

// Create 创建Anthropic提供商
func (f *AnthropicFactory) Create(config *types.ModelConfig) (Provider, error) {
	return NewAnthropicProvider(config)
}
