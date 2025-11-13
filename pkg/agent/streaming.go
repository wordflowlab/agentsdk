package agent

import (
	"context"
	"fmt"
	"iter"
	"log"

	"github.com/wordflowlab/agentsdk/pkg/events"
	"github.com/wordflowlab/agentsdk/pkg/session"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// StreamingAgent 扩展接口 - 支持流式执行
// 参考 Google ADK-Go 的 Agent.Run() iter.Seq2 设计
//
// 使用示例:
//
//	for event, err := range agent.Stream(ctx, "Hello") {
//	    if err != nil {
//	        log.Printf("Error: %v", err)
//	        break
//	    }
//	    fmt.Printf("Event: %+v\n", event)
//	}
type StreamingAgent interface {
	// Stream 流式执行，返回事件迭代器
	// 相比 Chat 方法的优势：
	// 1. 内存高效 - 按需生成事件，无需完整加载到内存
	// 2. 背压控制 - 客户端可通过 yield 返回 false 中断执行
	// 3. 实时响应 - 可以立即处理每个事件，而不是等待所有事件
	Stream(ctx context.Context, message string, opts ...Option) iter.Seq2[*session.Event, error]
}

// Stream 实现流式执行接口
// 返回 Go 1.23 的 iter.Seq2 迭代器，支持：
// - 流式生成事件
// - 客户端控制的取消（yield 返回 false）
// - 与 LLM 流式 API 无缝集成
func (a *Agent) Stream(ctx context.Context, message string, opts ...Option) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		// 应用选项
		config := &streamConfig{}
		for _, opt := range opts {
			opt(config)
		}

		log.Printf("[Agent Stream] Starting stream for message: %s", truncate(message, 50))

		// 1. 前置验证
		if err := a.validateMessage(message); err != nil {
			yield(nil, fmt.Errorf("validate message: %w", err))
			return
		}

		// 2. 创建用户消息
		userMsg := types.Message{
			Role:    types.RoleUser,
			Content: message,
		}

		// 3. 检查 Slash Commands
		if a.commandExecutor != nil {
			if handled, err := a.handleSlashCommandForStream(ctx, &userMsg, yield); handled {
				if err != nil {
					yield(nil, fmt.Errorf("slash command: %w", err))
				}
				return
			}
		}

		// 4. 应用 Skills 增强
		if a.skillInjector != nil {
			if err := a.skillInjector.Inject(ctx, &userMsg); err != nil {
				log.Printf("[Agent Stream] Skill injection failed: %v", err)
			}
		}

		// 5. 入队消息
		a.mu.Lock()
		a.messages = append(a.messages, userMsg)
		a.mu.Unlock()

		// 6. 持久化消息
		if err := a.persistMessage(ctx, &userMsg); err != nil {
			log.Printf("[Agent Stream] Failed to persist message: %v", err)
		}

		// 7. 流式执行模型步骤
		for {
			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
			}

			// 执行一步模型推理
			event, done, err := a.runModelStepStreaming(ctx)
			if err != nil {
				yield(nil, fmt.Errorf("model step: %w", err))
				return
			}

			// Yield 事件，如果客户端返回 false 则中断
			if event != nil {
				if !yield(event, nil) {
					log.Printf("[Agent Stream] Client cancelled stream")
					return
				}
			}

			// 检查是否完成
			if done {
				log.Printf("[Agent Stream] Stream completed")
				return
			}
		}
	}
}

// StreamCollect 辅助函数 - 收集所有事件
// 用于向后兼容，将流式接口转换为批量结果
//
// 使用示例:
//
//	events, err := agent.StreamCollect(agent.Stream(ctx, "Hello"))
//	if err != nil {
//	    return err
//	}
//	for _, event := range events {
//	    fmt.Println(event)
//	}
func StreamCollect(stream iter.Seq2[*session.Event, error]) ([]*session.Event, error) {
	var events []*session.Event
	for event, err := range stream {
		if err != nil {
			return events, err
		}
		if event != nil {
			events = append(events, event)
		}
	}
	return events, nil
}

// StreamFirst 辅助函数 - 获取第一个事件
func StreamFirst(stream iter.Seq2[*session.Event, error]) (*session.Event, error) {
	for event, err := range stream {
		return event, err
	}
	return nil, fmt.Errorf("no events in stream")
}

// StreamLast 辅助函数 - 获取最后一个事件
func StreamLast(stream iter.Seq2[*session.Event, error]) (*session.Event, error) {
	var lastEvent *session.Event
	var lastErr error

	for event, err := range stream {
		if err != nil {
			lastErr = err
			continue
		}
		if event != nil {
			lastEvent = event
		}
	}

	if lastErr != nil {
		return lastEvent, lastErr
	}
	if lastEvent == nil {
		return nil, fmt.Errorf("no events in stream")
	}
	return lastEvent, nil
}

// StreamFilter 辅助函数 - 过滤事件
func StreamFilter(stream iter.Seq2[*session.Event, error], predicate func(*session.Event) bool) iter.Seq2[*session.Event, error] {
	return func(yield func(*session.Event, error) bool) {
		for event, err := range stream {
			if err != nil {
				yield(nil, err)
				return
			}
			if event != nil && predicate(event) {
				if !yield(event, nil) {
					return
				}
			}
		}
	}
}

// streamConfig 流式执行配置
type streamConfig struct {
	// 未来可扩展的配置项
}

// Option 流式执行选项
type Option func(*streamConfig)

// validateMessage 验证消息
func (a *Agent) validateMessage(message string) error {
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}
	return nil
}

// handleSlashCommandForStream 处理 Slash Command（流式版本）
func (a *Agent) handleSlashCommandForStream(ctx context.Context, msg *types.Message, yield func(*session.Event, error) bool) (bool, error) {
	// 检查是否为 slash command
	if !a.commandExecutor.IsSlashCommand(msg.Content) {
		return false, nil
	}

	// 执行 slash command
	result, err := a.commandExecutor.Execute(ctx, msg.Content)
	if err != nil {
		return true, err
	}

	// 生成响应事件
	event := &session.Event{
		ID:        generateEventID(),
		Timestamp: a.createdAt,
		AgentID:   a.id,
		Author:    "system",
		Content: types.Message{
			Role:    types.RoleAssistant,
			Content: result,
		},
	}

	// Yield 事件
	yield(event, nil)
	return true, nil
}

// runModelStepStreaming 流式执行模型步骤
// 返回: (event, done, error)
func (a *Agent) runModelStepStreaming(ctx context.Context) (*session.Event, bool, error) {
	// 1. 准备消息
	a.mu.RLock()
	messages := make([]types.Message, len(a.messages))
	copy(messages, a.messages)
	a.mu.RUnlock()

	// 2. 通过 Middleware 调用 LLM
	var resp *types.Response
	var err error

	if a.middlewareStack != nil {
		// 使用 Middleware Stack
		req := &types.Request{
			Messages: messages,
			Tools:    a.getToolsForProvider(),
			MaxTokens: a.getMaxTokens(),
		}

		resp, err = a.middlewareStack.Handle(ctx, req, a.provider.SendMessage)
	} else {
		// 直接调用 Provider
		resp, err = a.provider.SendMessage(ctx, &types.Request{
			Messages: messages,
			Tools:    a.getToolsForProvider(),
			MaxTokens: a.getMaxTokens(),
		})
	}

	if err != nil {
		return nil, false, err
	}

	// 3. 处理响应
	a.mu.Lock()
	a.messages = append(a.messages, resp.Message)
	a.mu.Unlock()

	// 4. 检查是否有工具调用
	if len(resp.Message.ToolCalls) > 0 {
		// 执行工具调用
		if err := a.executeToolCalls(ctx, resp.Message.ToolCalls); err != nil {
			return nil, false, err
		}
		// 继续循环
		return nil, false, nil
	}

	// 5. 生成事件
	event := &session.Event{
		ID:        generateEventID(),
		Timestamp: a.createdAt,
		AgentID:   a.id,
		Author:    "assistant",
		Content:   resp.Message,
		Actions:   session.EventActions{},
	}

	// 6. 持久化事件
	if err := a.persistEvent(ctx, event); err != nil {
		log.Printf("[Agent Stream] Failed to persist event: %v", err)
	}

	// 7. 发布事件到 EventBus
	if a.eventBus != nil {
		a.eventBus.Publish(events.Event{
			Type:      events.EventTypeMessageReceived,
			AgentID:   a.id,
			Timestamp: event.Timestamp,
			Data:      event,
		})
	}

	return event, true, nil
}

// executeToolCalls 执行工具调用
func (a *Agent) executeToolCalls(ctx context.Context, toolCalls []types.ToolCall) error {
	results := make([]types.Message, len(toolCalls))

	for i, call := range toolCalls {
		tool, ok := a.toolMap[call.Name]
		if !ok {
			results[i] = types.Message{
				Role:       types.RoleToolResult,
				ToolCallID: call.ID,
				Content:    fmt.Sprintf("Error: tool '%s' not found", call.Name),
			}
			continue
		}

		// 执行工具
		result, err := a.executor.Execute(ctx, tool, call.Arguments)
		if err != nil {
			results[i] = types.Message{
				Role:       types.RoleToolResult,
				ToolCallID: call.ID,
				Content:    fmt.Sprintf("Error: %v", err),
			}
			continue
		}

		results[i] = types.Message{
			Role:       types.RoleToolResult,
			ToolCallID: call.ID,
			Content:    fmt.Sprint(result),
		}
	}

	// 追加工具结果到消息历史
	a.mu.Lock()
	a.messages = append(a.messages, results...)
	a.mu.Unlock()

	return nil
}

// getToolsForProvider 获取 Provider 格式的工具定义
func (a *Agent) getToolsForProvider() []types.ToolDefinition {
	tools := make([]types.ToolDefinition, 0, len(a.toolMap))
	for _, tool := range a.toolMap {
		tools = append(tools, types.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		})
	}
	return tools
}

// getMaxTokens 获取最大 token 数
func (a *Agent) getMaxTokens() int {
	if a.config.ModelConfig != nil && a.config.ModelConfig.MaxTokens > 0 {
		return a.config.ModelConfig.MaxTokens
	}
	return 4096 // 默认值
}

// persistMessage 持久化消息
func (a *Agent) persistMessage(ctx context.Context, msg *types.Message) error {
	// TODO: 实现消息持久化
	return nil
}

// persistEvent 持久化事件
func (a *Agent) persistEvent(ctx context.Context, event *session.Event) error {
	// TODO: 实现事件持久化到 SessionService
	return nil
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// generateEventID 生成事件 ID
func generateEventID() string {
	return "evt_" + uuid.New().String()
}
