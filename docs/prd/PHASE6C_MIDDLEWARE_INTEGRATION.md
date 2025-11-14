# Phase 6C: Agent Middleware 支持与 Summarization 集成

## 概述

Phase 6C 实现了 Agent 层的 Middleware 架构支持,使 Agent 能够通过 Middleware Stack 处理模型调用和工具调用。这是一个重要的架构升级,为未来扩展提供了强大的基础。

## 实现目标

1. ✅ Agent 支持通过配置启用 Middleware
2. ✅ 模型调用通过 Middleware Stack 执行
3. ✅ 工具调用通过 Middleware Stack 执行
4. ✅ Summarization Middleware 可直接使用
5. ✅ 完全向后兼容(不配置 Middleware 时行为不变)

## 核心变更

### 1. Middleware Registry (新增)

**文件**: `pkg/middleware/registry.go` (117 行)

创建了 Middleware 工厂注册表,支持动态注册和创建 Middleware:

```go
// MiddlewareFactory 中间件工厂函数
type MiddlewareFactory func(config *MiddlewareFactoryConfig) (Middleware, error)

// Registry 中间件注册表
type Registry struct {
    mu        sync.RWMutex
    factories map[string]MiddlewareFactory
}

// 全局默认注册表
var DefaultRegistry = NewRegistry()
```

**内置 Middleware**:
- `summarization`: 自动总结对话历史以管理上下文窗口

### 2. Agent Config 扩展

**文件**: `pkg/types/config.go:151`

```go
type AgentConfig struct {
    AgentID         string
    TemplateID      string
    // ... 其他字段
    Middlewares     []string  // Middleware 列表 (Phase 6C)
    // ... 其他字段
}
```

### 3. Agent 结构扩展

**文件**: `pkg/agent/agent.go`

#### 3.1 添加 middlewareStack 字段

```go
type Agent struct {
    // ... 其他字段

    // Middleware 支持 (Phase 6C)
    middlewareStack *middleware.Stack

    // ... 其他字段
}
```

#### 3.2 Create() 函数中初始化

```go
// 初始化 Middleware Stack (Phase 6C)
var middlewareStack *middleware.Stack
if len(config.Middlewares) > 0 {
    middlewareList := make([]middleware.Middleware, 0, len(config.Middlewares))
    for _, name := range config.Middlewares {
        mw, err := middleware.DefaultRegistry.Create(name, &middleware.MiddlewareFactoryConfig{
            Provider: prov,
            AgentID:  config.AgentID,
            Metadata: config.Metadata,
            Sandbox:  sb,
        })
        if err != nil {
            log.Printf("[Agent Create] Failed to create middleware %s: %v", name, err)
            continue
        }
        middlewareList = append(middlewareList, mw)
        log.Printf("[Agent Create] Middleware loaded: %s (priority: %d)", name, mw.Priority())
    }
    if len(middlewareList) > 0 {
        middlewareStack = middleware.NewStack(middlewareList)
        log.Printf("[Agent Create] Middleware stack created with %d middlewares", len(middlewareList))
    }
}
```

#### 3.3 生命周期回调

```go
// initialize() 中
if a.middlewareStack != nil {
    if err := a.middlewareStack.OnAgentStart(ctx, a.id); err != nil {
        return fmt.Errorf("middleware onAgentStart: %w", err)
    }
}

// Close() 中
if a.middlewareStack != nil {
    ctx := context.Background()
    if err := a.middlewareStack.OnAgentStop(ctx, a.id); err != nil {
        log.Printf("[Agent Close] Middleware OnAgentStop error: %v", err)
    }
}
```

### 4. 模型调用集成

**文件**: `pkg/agent/processor.go`

#### 4.1 提取流式响应处理

创建了独立的 `handleStreamResponse()` 方法(152 行):

```go
// handleStreamResponse 处理流式响应(Phase 6C - 提取为独立方法以支持Middleware)
func (a *Agent) handleStreamResponse(ctx context.Context, stream <-chan provider.StreamChunk) (types.Message, error) {
    // 完整的流式响应处理逻辑
    // ...
    return types.Message{
        Role:    types.MessageRoleAssistant,
        Content: assistantContent,
    }, nil
}
```

#### 4.2 重构 runModelStep()

通过 Middleware Stack 调用模型:

```go
func (a *Agent) runModelStep(ctx context.Context) error {
    // ... 准备工作

    var assistantMessage types.Message
    var modelErr error

    if a.middlewareStack != nil {
        // 使用 middleware stack
        req := &middleware.ModelRequest{
            Messages:     messages,
            SystemPrompt: currentSystemPrompt,
            Tools:        nil,
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
        // 没有 middleware, 直接调用 (向后兼容)
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

    // ... 保存消息和检查工具调用
}
```

### 5. 工具调用集成

**文件**: `pkg/agent/processor.go:executeSingleTool()`

```go
func (a *Agent) executeSingleTool(ctx context.Context, tu *types.ToolUseBlock) types.ContentBlock {
    // ... 准备工作

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
            execResult = &tools.ExecuteResult{
                Success: false,
                Error:   err,
            }
        } else {
            execResult = resp.Result.(*tools.ExecuteResult)
        }
    } else {
        // 没有 middleware, 直接执行 (向后兼容)
        execResult = a.executor.Execute(ctx, &tools.ExecuteRequest{
            Tool:    tool,
            Input:   tu.Input,
            Context: toolCtx,
            Timeout: 60 * time.Second,
        })
    }

    // ... 更新记录和返回结果
}
```

## 使用示例

### 1. 启用 Summarization Middleware

```go
config := &types.AgentConfig{
    TemplateID: "my-template",
    Middlewares: []string{"summarization"},  // 启用自动总结
    ModelConfig: &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-3-5-sonnet-20241022",
    },
}

agent, err := agent.Create(ctx, config, deps)
if err != nil {
    log.Fatal(err)
}
defer agent.Close()

// Agent 现在会自动:
// 1. 监控消息历史的 token 数量
// 2. 当超过 170,000 tokens 时,自动总结旧消息
// 3. 保留最近 6 条消息
// 4. 用总结消息替换旧的历史记录
```

### 2. 不使用 Middleware (向后兼容)

```go
config := &types.AgentConfig{
    TemplateID: "my-template",
    // 不设置 Middlewares 字段
    ModelConfig: &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-3-5-sonnet-20241022",
    },
}

agent, err := agent.Create(ctx, config, deps)
// Agent 的行为与之前完全一致
```

### 3. 自定义 Middleware

```go
// 注册自定义 Middleware
middleware.DefaultRegistry.Register("my_custom", func(config *middleware.MiddlewareFactoryConfig) (middleware.Middleware, error) {
    return &MyCustomMiddleware{
        BaseMiddleware: middleware.NewBaseMiddleware("my_custom", 100),
    }, nil
})

// 使用自定义 Middleware
config := &types.AgentConfig{
    TemplateID: "my-template",
    Middlewares: []string{"summarization", "my_custom"},
}
```

## 测试结果

所有测试通过:

```
✅ pkg/agent        - PASS (0.708s)
✅ pkg/backends     - PASS (0.788s)
✅ pkg/middleware   - PASS (1.845s)
✅ pkg/tools/builtin - PASS (8.172s)
✅ pkg/tools/mcp    - PASS (2.343s)
✅ pkg/core         - PASS (3.840s)
```

## 代码统计

| 文件 | 变更类型 | 行数 |
|------|---------|------|
| pkg/middleware/registry.go | 新增 | 117 行 |
| pkg/types/config.go | 修改 | +1 行 |
| pkg/agent/agent.go | 修改 | +38 行 |
| pkg/agent/processor.go | 重构 | +200 行 |
| **总计** | | **~356 行** |

## 架构优势

### 1. **洋葱模型**
Middleware 采用洋葱模型,支持请求和响应的双向拦截:
```
Request → Middleware1 → Middleware2 → Provider/Tool → Middleware2 → Middleware1 → Response
```

### 2. **优先级控制**
每个 Middleware 有优先级,数值越小越早执行:
- 0-100: 系统核心 Middleware
- 100-500: 功能 Middleware (如 Summarization)
- 500-1000: 用户自定义 Middleware

### 3. **生命周期管理**
- `OnAgentStart()`: Agent 启动时初始化
- `WrapModelCall()`: 包装模型调用
- `WrapToolCall()`: 包装工具调用
- `OnAgentStop()`: Agent 关闭时清理

### 4. **完全可扩展**
可以轻松添加新的 Middleware:
- **Logging Middleware**: 记录所有调用
- **Caching Middleware**: 缓存模型响应
- **Rate Limiting Middleware**: API 调用速率限制
- **Monitoring Middleware**: 性能监控
- **Security Middleware**: 输入输出安全检查

## 与 DeepAgents 对比

| 特性 | WriteFlow-SDK | DeepAgents | 对齐度 |
|------|--------------|-----------|--------|
| Middleware Registry | ✅ | ✅ | 100% |
| 模型调用拦截 | ✅ | ✅ | 100% |
| 工具调用拦截 | ✅ | ✅ | 100% |
| Summarization | ✅ | ✅ | 100% |
| 生命周期回调 | ✅ | ✅ | 100% |
| 优先级排序 | ✅ | ✅ | 100% |

**结论**: WriteFlow-SDK 的 Middleware 架构与 DeepAgents 100% 对齐。

## 未来扩展

### 1. Prompt Caching Middleware
支持 Anthropic Prompt Caching API:
```go
type PromptCachingMiddleware struct {
    *BaseMiddleware
    cacheBreakpoints []string
}

func (m *PromptCachingMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
    // 在 system prompt 中插入 cache_control 标记
    req.SystemPrompt = addCacheControl(req.SystemPrompt)
    return handler(ctx, req)
}
```

### 2. Logging Middleware
自动记录所有调用:
```go
type LoggingMiddleware struct {
    *BaseMiddleware
    logger *log.Logger
}

func (m *LoggingMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
    m.logger.Printf("[Model Call] Messages: %d, System: %d chars", len(req.Messages), len(req.SystemPrompt))
    start := time.Now()
    resp, err := handler(ctx, req)
    m.logger.Printf("[Model Call] Duration: %v, Error: %v", time.Since(start), err)
    return resp, err
}
```

### 3. Rate Limiting Middleware
API 调用速率限制:
```go
type RateLimitingMiddleware struct {
    *BaseMiddleware
    limiter *rate.Limiter
}

func (m *RateLimitingMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
    if err := m.limiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limit: %w", err)
    }
    return handler(ctx, req)
}
```

## 总结

Phase 6C 成功实现了 Agent 层的 Middleware 架构支持,这是一个重要的架构升级:

1. ✅ **功能完整**: 模型调用和工具调用都支持 Middleware
2. ✅ **架构优雅**: 洋葱模型,优先级控制,生命周期管理
3. ✅ **向后兼容**: 不破坏现有代码
4. ✅ **可扩展性强**: 易于添加新的 Middleware
5. ✅ **DeepAgents 对齐**: 100% 对齐 DeepAgents 架构

这为 WriteFlow-SDK 的未来发展奠定了坚实的基础,使其能够灵活应对各种需求变化。
