package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/wordflowlab/agentsdk/pkg/middleware"
	"github.com/wordflowlab/agentsdk/pkg/provider"
	"github.com/wordflowlab/agentsdk/pkg/tools"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// processMessages 处理消息队列
func (a *Agent) processMessages(ctx context.Context) {
	a.mu.Lock()
	if a.state != types.AgentStateReady {
		a.mu.Unlock()
		return // 已经在处理中
	}
	a.state = types.AgentStateWorking
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state = types.AgentStateReady
		a.mu.Unlock()
	}()

	// 发送状态变更事件
	a.eventBus.EmitMonitor(&types.MonitorStateChangedEvent{
		State: types.AgentStateWorking,
	})

	// 设置断点
	a.setBreakpoint(types.BreakpointPreModel)

	// 调用模型
	if err := a.runModelStep(ctx); err != nil {
		a.eventBus.EmitMonitor(&types.MonitorErrorEvent{
			Severity: "error",
			Phase:    "model",
			Message:  err.Error(),
		})
	}

	// 发送完成事件
	a.eventBus.EmitProgress(&types.ProgressDoneEvent{
		Step:   a.stepCount,
		Reason: "completed",
	})

	// 发送状态变更事件
	a.eventBus.EmitMonitor(&types.MonitorStateChangedEvent{
		State: types.AgentStateReady,
	})
}

// runModelStep 运行模型步骤
func (a *Agent) runModelStep(ctx context.Context) error {
	// 检查执行模式
	executionMode := a.getExecutionMode()
	if executionMode == types.ExecutionModeNonStreaming {
		log.Printf("[runModelStep] Using NON-STREAMING mode (fast execution)")
		return a.runNonStreamingStep(ctx)
	}

	log.Printf("[runModelStep] Using STREAMING mode (real-time feedback)")
	a.setBreakpoint(types.BreakpointStreamingModel)

	// 准备工具Schema
	toolSchemas := make([]provider.ToolSchema, 0, len(a.toolMap))
	for _, tool := range a.toolMap {
		toolSchemas = append(toolSchemas, provider.ToolSchema{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		})
	}
	toolNames := make([]string, len(toolSchemas))
	for i, ts := range toolSchemas {
		toolNames[i] = ts.Name
	}
	log.Printf("[runModelStep] Agent %s: Prepared %d tool schemas: %v", a.id, len(toolSchemas), toolNames)

	// 调用模型
	// 确保系统提示词包含工具手册（如果还没有注入）
	a.mu.RLock()
	hasManual := strings.Contains(a.template.SystemPrompt, "### Tools Manual")
	toolMapSize := len(a.toolMap)
	currentSystemPrompt := a.template.SystemPrompt
	messages := a.messages // 复制当前消息列表
	a.mu.RUnlock()

	if !hasManual && toolMapSize > 0 {
		log.Printf("[runModelStep] Agent %s: Manual not found, injecting... (toolMap size: %d)", a.id, toolMapSize)
		a.injectToolManual()
		a.mu.RLock()
		currentSystemPrompt = a.template.SystemPrompt
		hasManual = strings.Contains(currentSystemPrompt, "### Tools Manual")
		a.mu.RUnlock()
		log.Printf("[runModelStep] Agent %s: After injection, system prompt length: %d, contains manual: %v", a.id, len(currentSystemPrompt), hasManual)
	} else if toolMapSize == 0 {
		log.Printf("[runModelStep] Agent %s: No tools in toolMap, cannot inject manual", a.id)
	}

	log.Printf("[runModelStep] Agent %s: Final system prompt length: %d, contains manual: %v", a.id, len(currentSystemPrompt), strings.Contains(currentSystemPrompt, "### Tools Manual"))

	// 通过 Middleware Stack 调用模型 (Phase 6C)
	var assistantMessage types.Message
	var modelErr error

	if a.middlewareStack != nil {
		// 使用 middleware stack
		req := &middleware.ModelRequest{
			Messages:     messages,
			SystemPrompt: currentSystemPrompt,
			Tools:        nil, // TODO: 转换 toolMap 为 []tools.Tool
			Metadata:     make(map[string]interface{}),
		}

		// 定义 finalHandler: 实际调用 Provider
		finalHandler := func(ctx context.Context, req *middleware.ModelRequest) (*middleware.ModelResponse, error) {
			streamOpts := &provider.StreamOptions{
				Tools:     toolSchemas,
				MaxTokens: 4096,
				System:    req.SystemPrompt,
			}

			stream, err := a.provider.Stream(ctx, req.Messages, streamOpts)
			if err != nil {
				return nil, fmt.Errorf("stream model: %w", err)
			}

			// 处理流式响应
			message, err := a.handleStreamResponse(ctx, stream)
			if err != nil {
				return nil, err
			}

			return &middleware.ModelResponse{
				Message:  message,
				Metadata: make(map[string]interface{}),
			}, nil
		}

		// 通过 middleware stack 执行
		resp, err := a.middlewareStack.ExecuteModelCall(ctx, req, finalHandler)
		if err != nil {
			modelErr = err
		} else {
			assistantMessage = resp.Message
		}
	} else {
		// 没有 middleware, 直接调用
		streamOpts := &provider.StreamOptions{
			Tools:     toolSchemas,
			MaxTokens: 4096,
			System:    currentSystemPrompt,
		}

		stream, err := a.provider.Stream(ctx, messages, streamOpts)
		if err != nil {
			modelErr = err
		} else {
			assistantMessage, err = a.handleStreamResponse(ctx, stream)
			if err != nil {
				modelErr = err
			}
		}
	}

	// 处理模型调用错误
	if modelErr != nil {
		return fmt.Errorf("model call: %w", modelErr)
	}

	// 保存助手消息
	a.mu.Lock()
	a.messages = append(a.messages, assistantMessage)
	a.mu.Unlock()

	// 持久化
	if err := a.deps.Store.SaveMessages(ctx, a.id, a.messages); err != nil {
		return fmt.Errorf("save messages: %w", err)
	}

	// 检查是否有工具调用
	toolUses := make([]*types.ToolUseBlock, 0)
	for _, block := range assistantMessage.ContentBlocks {
		if tu, ok := block.(*types.ToolUseBlock); ok {
			toolUses = append(toolUses, tu)
		}
	}

	log.Printf("[runModelStep] Agent %s: Found %d tool uses in response", a.id, len(toolUses))
	if len(toolUses) > 0 {
		for _, tu := range toolUses {
			log.Printf("[runModelStep] Agent %s: Tool use - Name: %s, ID: %s, Input: %v", a.id, tu.Name, tu.ID, tu.Input)
		}
		a.setBreakpoint(types.BreakpointToolPending)
		return a.executeTools(ctx, toolUses)
	} else {
		log.Printf("[runModelStep] Agent %s: No tool uses found, only text response", a.id)
	}

	return nil
}

// executeTools 执行工具
func (a *Agent) executeTools(ctx context.Context, toolUses []*types.ToolUseBlock) error {
	toolResults := make([]types.ContentBlock, 0, len(toolUses))

	for _, tu := range toolUses {
		result := a.executeSingleTool(ctx, tu)
		toolResults = append(toolResults, result)
	}

	// 保存工具结果
	a.mu.Lock()
	a.messages = append(a.messages, types.Message{
		Role:          types.MessageRoleUser,
		ContentBlocks: toolResults,
	})
	a.stepCount++
	a.mu.Unlock()

	// 持久化
	if err := a.deps.Store.SaveMessages(ctx, a.id, a.messages); err != nil {
		return fmt.Errorf("save messages: %w", err)
	}

	// 持久化工具记录
	records := make([]types.ToolCallRecord, 0, len(a.toolRecords))
	for _, record := range a.toolRecords {
		records = append(records, *record)
	}
	if err := a.deps.Store.SaveToolCallRecords(ctx, a.id, records); err != nil {
		return fmt.Errorf("save tool records: %w", err)
	}

	// 继续处理
	return a.runModelStep(ctx)
}

// executeSingleTool 执行单个工具
func (a *Agent) executeSingleTool(ctx context.Context, tu *types.ToolUseBlock) types.ContentBlock {
	// 创建工具调用记录
	record := tools.NewToolCallRecord(tu.ID, tu.Name, tu.Input).Build()
	a.mu.Lock()
	a.toolRecords[tu.ID] = record
	a.mu.Unlock()

	// 发送工具开始事件
	a.eventBus.EmitProgress(&types.ProgressToolStartEvent{
		Call: types.ToolCallSnapshot{
			ID:        record.ID,
			Name:      record.Name,
			State:     record.State,
			Arguments: record.Input,
		},
	})

	// 获取工具
	tool, ok := a.toolMap[tu.Name]
	if !ok {
		// 工具未找到
		errorMsg := fmt.Sprintf("tool not found: %s", tu.Name)
		a.updateToolRecord(tu.ID, types.ToolCallStateFailed, errorMsg)
		a.eventBus.EmitProgress(&types.ProgressToolErrorEvent{
			Call: types.ToolCallSnapshot{
				ID:        tu.ID,
				Name:      tu.Name,
				State:     types.ToolCallStateFailed,
				Arguments: tu.Input,
			},
			Error: errorMsg,
		})
		return &types.ToolResultBlock{
			ToolUseID: tu.ID,
			IsError:   true,
		}
	}

	// 设置断点
	a.setBreakpoint(types.BreakpointPreTool)

	// 执行工具
	a.updateToolRecord(tu.ID, types.ToolCallStateExecuting, "")
	a.setBreakpoint(types.BreakpointToolExecuting)

	startTime := time.Now()

	toolCtx := &tools.ToolContext{
		AgentID: a.id,
		Sandbox: a.sandbox,
		Signal:  ctx,
	}

	// 通过 Middleware Stack 执行工具 (Phase 6C)
	var execResult *tools.ExecuteResult
	if a.middlewareStack != nil {
		// 使用 middleware stack
		req := &middleware.ToolCallRequest{
			ToolCallID: tu.ID,
			ToolName:   tu.Name,
			ToolInput:  tu.Input,
			Tool:       tool,
			Context:    toolCtx,
			Metadata:   make(map[string]interface{}),
		}

		// 定义 finalHandler: 实际执行工具
		finalHandler := func(ctx context.Context, req *middleware.ToolCallRequest) (*middleware.ToolCallResponse, error) {
			result := a.executor.Execute(ctx, &tools.ExecuteRequest{
				Tool:    req.Tool,
				Input:   req.ToolInput,
				Context: req.Context,
				Timeout: 60 * time.Second,
			})

			return &middleware.ToolCallResponse{
				Result:   result,
				Metadata: make(map[string]interface{}),
			}, nil
		}

		// 通过 middleware stack 执行
		resp, err := a.middlewareStack.ExecuteToolCall(ctx, req, finalHandler)
		if err != nil {
			// 如果 middleware 返回错误,创建失败结果
			execResult = &tools.ExecuteResult{
				Success: false,
				Error:   err,
			}
		} else {
			execResult = resp.Result.(*tools.ExecuteResult)
		}
	} else {
		// 没有 middleware, 直接执行
		execResult = a.executor.Execute(ctx, &tools.ExecuteRequest{
			Tool:    tool,
			Input:   tu.Input,
			Context: toolCtx,
			Timeout: 60 * time.Second,
		})
	}

	endTime := time.Now()

	// 更新记录
	if execResult.Success {
		a.updateToolRecord(tu.ID, types.ToolCallStateCompleted, "")
		a.mu.Lock()
		a.toolRecords[tu.ID].Result = execResult.Output
		a.toolRecords[tu.ID].StartedAt = &startTime
		a.toolRecords[tu.ID].CompletedAt = &endTime
		durationMs := execResult.DurationMs
		a.toolRecords[tu.ID].DurationMs = &durationMs
		a.mu.Unlock()
	} else {
		errorMsg := ""
		if execResult.Error != nil {
			errorMsg = execResult.Error.Error()
		}
		a.updateToolRecord(tu.ID, types.ToolCallStateFailed, errorMsg)
	}

	// 发送工具结束事件
	a.mu.RLock()
	finalRecord := a.toolRecords[tu.ID]
	a.mu.RUnlock()

	a.eventBus.EmitProgress(&types.ProgressToolEndEvent{
		Call: types.ToolCallSnapshot{
			ID:        tu.ID,
			Name:      tu.Name,
			State:     finalRecord.State,
			Arguments: finalRecord.Input,
			Result:    finalRecord.Result,
			Error:     finalRecord.Error,
		},
	})

	// 设置断点
	a.setBreakpoint(types.BreakpointPostTool)

	// 构建工具结果
	if execResult.Success {
		return &types.ToolResultBlock{
			ToolUseID: tu.ID,
			Content:   fmt.Sprintf("%v", execResult.Output),
			IsError:   false,
		}
	} else {
		errorMsg := ""
		if execResult.Error != nil {
			errorMsg = execResult.Error.Error()
		}
		return &types.ToolResultBlock{
			ToolUseID: tu.ID,
			Content:   fmt.Sprintf(`{"ok":false,"error":"%s"}`, errorMsg),
			IsError:   true,
		}
	}
}

// setBreakpoint 设置断点
func (a *Agent) setBreakpoint(state types.BreakpointState) {
	a.mu.Lock()
	previous := a.breakpoint
	a.breakpoint = state
	a.mu.Unlock()

	a.eventBus.EmitMonitor(&types.MonitorBreakpointChangedEvent{
		Previous:  previous,
		Current:   state,
		Timestamp: time.Now(),
	})
}

// updateToolRecord 更新工具记录
func (a *Agent) updateToolRecord(id string, state types.ToolCallState, errorMsg string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	record, ok := a.toolRecords[id]
	if !ok {
		return
	}

	now := time.Now()
	record.State = state
	record.UpdatedAt = now

	if errorMsg != "" {
		record.Error = errorMsg
		record.IsError = true
	}

	record.AuditTrail = append(record.AuditTrail, types.ToolCallAuditEntry{
		State:     state,
		Timestamp: now,
	})
}

// handleStreamResponse 处理流式响应(Phase 6C - 提取为独立方法以支持Middleware)
func (a *Agent) handleStreamResponse(ctx context.Context, stream <-chan provider.StreamChunk) (types.Message, error) {
	assistantContent := make([]types.ContentBlock, 0)
	currentBlockIndex := -1
	textBuffers := make(map[int]string)
	inputJSONBuffers := make(map[int]string)

	for chunk := range stream {
		switch chunk.Type {
		case "content_block_start":
			currentBlockIndex = chunk.Index
			if delta, ok := chunk.Delta.(map[string]interface{}); ok {
				blockType, _ := delta["type"].(string)
				if blockType == "text" {
					// 发送文本开始事件
					a.eventBus.EmitProgress(&types.ProgressTextChunkStartEvent{
						Step: a.stepCount,
					})
					// 初始化文本块
					for len(assistantContent) <= currentBlockIndex {
						assistantContent = append(assistantContent, nil)
					}
					assistantContent[currentBlockIndex] = &types.TextBlock{Text: ""}
					textBuffers[currentBlockIndex] = ""
				} else if blockType == "tool_use" {
					log.Printf("[handleStreamResponse] Received tool_use block! ID: %v, Name: %v", delta["id"], delta["name"])
					// 初始化工具调用块
					for len(assistantContent) <= currentBlockIndex {
						assistantContent = append(assistantContent, nil)
					}

					// 处理不同的工具调用格式（Anthropic vs OpenAI兼容格式）
					toolID := ""
					toolName := ""
					if id, ok := delta["id"].(string); ok {
						toolID = id
					} else if id, ok := delta["id"].(float64); ok {
						toolID = fmt.Sprintf("%.0f", id)
					}

					if name, ok := delta["name"].(string); ok {
						toolName = name
					}

					assistantContent[currentBlockIndex] = &types.ToolUseBlock{
						ID:    toolID,
						Name:  toolName,
						Input: make(map[string]interface{}),
					}
				} else {
					log.Printf("[handleStreamResponse] Unknown block type: %s", blockType)
				}
			}

		case "content_block_delta":
			if delta, ok := chunk.Delta.(map[string]interface{}); ok {
				deltaType, _ := delta["type"].(string)
				if deltaType == "text_delta" {
					text, _ := delta["text"].(string)
					if currentBlockIndex >= 0 {
						for len(assistantContent) <= currentBlockIndex {
							assistantContent = append(assistantContent, nil)
						}
						if assistantContent[currentBlockIndex] == nil {
							assistantContent[currentBlockIndex] = &types.TextBlock{Text: ""}
							textBuffers[currentBlockIndex] = ""
						}
						if _, exists := textBuffers[currentBlockIndex]; !exists {
							textBuffers[currentBlockIndex] = ""
						}
						textBuffers[currentBlockIndex] += text
						if block, ok := assistantContent[currentBlockIndex].(*types.TextBlock); ok {
							block.Text = textBuffers[currentBlockIndex]
						}
					}
					// 发送文本增量事件
					a.eventBus.EmitProgress(&types.ProgressTextChunkEvent{
						Step:  a.stepCount,
						Delta: text,
					})
				} else if deltaType == "input_json_delta" {
					partialJSON, _ := delta["partial_json"].(string)
					if currentBlockIndex >= 0 {
						if _, exists := inputJSONBuffers[currentBlockIndex]; !exists {
							inputJSONBuffers[currentBlockIndex] = ""
						}
						inputJSONBuffers[currentBlockIndex] += partialJSON
					}
				} else if deltaType == "arguments" {
					partialArgs, _ := delta["arguments"].(string)
					blockIndex := chunk.Index
					if blockIndex < 0 {
						blockIndex = currentBlockIndex
					}
					if blockIndex >= 0 {
						if _, exists := inputJSONBuffers[blockIndex]; !exists {
							inputJSONBuffers[blockIndex] = ""
						}
						inputJSONBuffers[blockIndex] += partialArgs
					}
				}
			}

		case "content_block_stop":
			if currentBlockIndex >= 0 && currentBlockIndex < len(assistantContent) {
				if block, ok := assistantContent[currentBlockIndex].(*types.TextBlock); ok {
					a.eventBus.EmitProgress(&types.ProgressTextChunkEndEvent{
						Step: a.stepCount,
						Text: block.Text,
					})
				} else if block, ok := assistantContent[currentBlockIndex].(*types.ToolUseBlock); ok {
					if jsonStr, exists := inputJSONBuffers[currentBlockIndex]; exists && jsonStr != "" {
						var input map[string]interface{}
						if err := json.Unmarshal([]byte(jsonStr), &input); err == nil {
							block.Input = input
						} else {
							log.Printf("[handleStreamResponse] Failed to parse tool input JSON: %v", err)
						}
					}
				}
			}

		case "message_delta":
			if chunk.Usage != nil {
				a.eventBus.EmitMonitor(&types.MonitorTokenUsageEvent{
					InputTokens:  chunk.Usage.InputTokens,
					OutputTokens: chunk.Usage.OutputTokens,
					TotalTokens:  chunk.Usage.InputTokens + chunk.Usage.OutputTokens,
				})
			}
		}
	}

	// 流式响应结束后，解析所有累积的工具输入
	if len(inputJSONBuffers) > 0 {
		for i, block := range assistantContent {
			if tu, ok := block.(*types.ToolUseBlock); ok {
				if jsonStr, exists := inputJSONBuffers[i]; exists && jsonStr != "" {
					var input map[string]interface{}
					if err := json.Unmarshal([]byte(jsonStr), &input); err == nil {
						tu.Input = input
					}
				}
			}
		}
	}

	return types.Message{
		Role:          types.MessageRoleAssistant,
		ContentBlocks: assistantContent,
	}, nil
}

// runNonStreamingStep 非流式执行模型步骤（快速模式）
func (a *Agent) runNonStreamingStep(ctx context.Context) error {
	// 准备工具Schema
	toolSchemas := make([]provider.ToolSchema, 0, len(a.toolMap))
	for _, tool := range a.toolMap {
		toolSchemas = append(toolSchemas, provider.ToolSchema{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		})
	}

	// 准备消息
	a.mu.RLock()
	messages := make([]types.Message, len(a.messages))
	copy(messages, a.messages)
	currentSystemPrompt := a.template.SystemPrompt
	a.mu.RUnlock()

	// 创建Provider选项
	streamOpts := &provider.StreamOptions{
		Tools:       toolSchemas,
		System:      currentSystemPrompt,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	log.Printf("[runNonStreamingStep] Calling Complete API with %d messages, %d tools",
		len(messages), len(toolSchemas))

	// 调用Complete API（非流式）
	response, err := a.provider.Complete(ctx, messages, streamOpts)
	if err != nil {
		return fmt.Errorf("complete call failed: %w", err)
	}

	// 添加响应消息
	a.mu.Lock()
	a.messages = append(a.messages, response.Message)
	a.mu.Unlock()

	// 提取工具调用从ContentBlocks
	toolUses := make([]*types.ToolUseBlock, 0)
	for _, block := range response.Message.ContentBlocks {
		if tu, ok := block.(*types.ToolUseBlock); ok {
			toolUses = append(toolUses, tu)
		}
	}

	log.Printf("[runNonStreamingStep] Received response with %d tool calls", len(toolUses))

	// 处理工具调用
	if len(toolUses) > 0 {
		// 执行工具
		if err := a.executeTools(ctx, toolUses); err != nil {
			return fmt.Errorf("execute tools failed: %w", err)
		}

		// 递归调用继续处理
		return a.runNonStreamingStep(ctx)
	}

	// 没有工具调用，完成
	a.mu.Lock()
	a.state = types.AgentStateReady
	a.mu.Unlock()

	log.Printf("[runNonStreamingStep] Execution completed")

	return nil
}
