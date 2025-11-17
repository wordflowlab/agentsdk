---
title: 错误处理
description: Agent 应用的错误处理和容错策略
navigation:
  icon: i-lucide-alert-triangle
---

# 错误处理最佳实践

构建可靠的 Agent 应用需要完善的错误处理策略。

## 🎯 核心原则

1. **永不忽略错误** - 所有错误都应处理或记录
2. **快速失败** - 尽早发现并报告问题
3. **优雅降级** - 部分功能失败不影响整体
4. **错误上下文** - 提供足够的调试信息

## 📊 错误分类

### 1. 可恢复错误 (Recoverable)

**特征**: 临时性、可重试、不影响系统稳定性

```go
// Token 超限 - 触发总结
case *types.TokenLimitError:
    log.Printf("Token limit reached, triggering summarization")
    return ag.Summarize(ctx)

// 网络错误 - 重试
case *types.NetworkError:
    return retry WithBackoff(func() error {
        return ag.Chat(ctx, message)
    }, 3, time.Second)

// 工具执行失败 - 降级
case *types.ToolExecutionError:
    log.Printf("Tool %s failed: %v", err.Tool Name, err)
    return fallbackBehavior(ctx)
```

### 2. 不可恢复错误 (Fatal)

**特征**: 配置错误、权限问题、资源耗尽

```go
// 配置错误 - 立即失败
if config.APIKey == "" {
    return nil, fmt.Errorf("API key is required")
}

// 资源耗尽 - 拒绝请求
if pool.Size() >= pool.MaxAgents {
    return nil, fmt.Errorf("agent pool is full")
}

// 权限不足 - 返回错误
if !hasPermission(user, operation) {
    return nil, fmt.Errorf("permission denied")
}
```

## 🛡️ 错误处理模式

### 模式1: 包装错误（Error Wrapping）

```go
// ✅ 提供错误上下文
func createAgent(ctx context.Context, userID string) (*agent.Agent, error) {
    config, err := loadConfig(userID)
    if err != nil {
        return nil, fmt.Errorf("failed to load config for user %s: %w", userID, err)
    }

    ag, err := agent.Create(ctx, config, deps)
    if err != nil {
        return nil, fmt.Errorf("failed to create agent for user %s: %w", userID, err)
    }

    return ag, nil
}

// ❌ 丢失上下文
func createAgentBad(ctx context.Context, userID string) (*agent.Agent, error) {
    config, _ := loadConfig(userID)  // 忽略错误
    ag, err := agent.Create(ctx, config, deps)
    return ag, err  // 没有上下文信息
}
```

### 模式2: 重试机制（Retry Pattern）

```go
func retryWithBackoff(fn func() error, maxRetries int, initialDelay time.Duration) error {
    var err error
    delay := initialDelay

    for i := 0; i < maxRetries; i++ {
        err = fn()
        if err == nil {
            return nil  // 成功
        }

        // 判断是否可重试
        if !isRetryable(err) {
            return err  // 不可重试的错误，直接返回
        }

        log.Printf("Attempt %d failed: %v, retrying in %v", i+1, err, delay)
        time.Sleep(delay)
        delay *= 2  // 指数退避
    }

    return fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, err)
}

// 判断错误是否可重试
func isRetryable(err error) bool {
    switch err.(type) {
    case *types.NetworkError:
        return true
    case *types.RateLimitError:
        return true
    case *types.TimeoutError:
        return true
    default:
        return false
    }
}

// 使用示例
err := retryWithBackoff(func() error {
    return ag.Chat(ctx, message)
}, 3, time.Second)
```

### 模式3: 优雅降级（Graceful Degradation）

```go
func handleToolFailure(ctx context.Context, toolName string, err error) (interface{}, error) {
    log.Printf("Tool %s failed: %v", toolName, err)

    // 根据工具类型选择降级策略
    switch toolName {
    case "WebSearch":
        // 搜索失败 → 使用缓存结果
        if cached, ok := getFromCache(toolName); ok {
            log.Printf("Using cached result for %s", toolName)
            return cached, nil
        }
        return nil, fmt.Errorf("search unavailable and no cache")

    case "HttpRequest":
        // HTTP 请求失败 → 使用备用 API
        log.Printf("Trying fallback API")
        return callFallbackAPI(ctx)

    case "database_query":
        // 数据库查询失败 → 返回默认值
        log.Printf("Returning default value")
        return getDefaultData(), nil

    default:
        return nil, err  // 无降级策略，返回原错误
    }
}
```

### 模式4: 错误边界（Error Boundaries）

```go
// HTTP Handler 错误边界
func chatHandler(w http.ResponseWriter, r *http.Request) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Panic recovered: %v\n%s", r, debug.Stack())
            http.Error(w, "Internal server error", 500)
        }
    }()

    // 业务逻辑
    result, err := processChat(r)
    if err != nil {
        handleHTTPError(w, err)
        return
    }

    json.NewEncoder(w).Encode(result)
}

func handleHTTPError(w http.ResponseWriter, err error) {
    switch err.(type) {
    case *types.ValidationError:
        http.Error(w, err.Error(), 400)  // Bad Request
    case *types.AuthenticationError:
        http.Error(w, "Unauthorized", 401)
    case *types.RateLimitError:
        http.Error(w, "Too many requests", 429)
    default:
        log.Printf("Unexpected error: %v", err)
        http.Error(w, "Internal server error", 500)
    }
}
```

## 📝 错误日志记录

### 结构化日志

```go
// ✅ 结构化日志 (JSON 格式)
log.Printf(`{
    "level": "error",
    "agent_id": "%s",
    "operation": "tool_call",
    "tool": "%s",
    "error": "%v",
    "timestamp": "%s",
    "user_id": "%s"
}`, agentID, toolName, err, time.Now().Format(time.RFC3339), userID)

// 或使用结构化日志库
logger.Error("Tool execution failed",
    zap.String("agent_id", agentID),
    zap.String("tool", toolName),
    zap.Error(err),
    zap.String("user_id", userID),
)

// ❌ 非结构化日志
log.Printf("Error: %v", err)  // 难以解析和分析
```

### 日志级别

```go
// ERROR - 需要关注的错误
log.Error("Agent creation failed", zap.Error(err))

// WARN - 可能的问题
log.Warn("Tool took longer than expected", zap.Duration("duration", dur))

// INFO - 重要事件
log.Info("Agent started", zap.String("agent_id", id))

// DEBUG - 调试信息
log.Debug("Tool call parameters", zap.Any("params", params))
```

## 🔔 错误监控和告警

### 指标收集

```go
// 错误计数
metrics.Increment("agent.errors.total", 1,
    tag("error_type", "tool_execution"),
    tag("tool", toolName),
)

// 错误率
errorRate := float64(errors) / float64(total)
metrics.Gauge("agent.error_rate", errorRate)

// 失败的工具调用
if err != nil {
    metrics.Increment("agent.tool_calls.failed", 1,
        tag("tool", toolName),
    )
}
```

### 告警规则

```yaml
# Prometheus 告警配置示例
groups:
  - name: agent_alerts
    rules:
      # 错误率过高
      - alert: HighErrorRate
        expr: rate(agent_errors_total[5m]) > 0.1
        for: 5m
        annotations:
          summary: "Agent error rate is high"

      # Agent 创建失败
      - alert: AgentCreationFailure
        expr: increase(agent_creation_failures[1m]) > 5
        annotations:
          summary: "Multiple agent creation failures"

      # Token 超限频繁
      - alert: FrequentTokenLimits
        expr: increase(agent_token_limit_errors[10m]) > 10
        annotations:
          summary: "Token limits being hit frequently"
```

## ⚠️ 常见错误场景

### 场景1: Token 超限

```go
result, err := ag.Chat(ctx, message)
if err != nil {
    if tokenErr, ok := err.(*types.TokenLimitError); ok {
        // 自动触发总结
        log.Printf("Token limit: %d/%d, triggering summary",
            tokenErr.Used, tokenErr.Limit)

        // 使用 Summarization 中间件（自动）
        // 或手动总结
        if err := ag.Summarize(ctx); err != nil {
            return fmt.Errorf("summarization failed: %w", err)
        }

        // 重试请求
        return ag.Chat(ctx, message)
    }
}
```

### 场景2: API 速率限制

```go
// 检测并处理速率限制
err := ag.Chat(ctx, message)
if rateLimitErr, ok := err.(*types.RateLimitError); ok {
    // 等待指定时间后重试
    waitTime := rateLimitErr.RetryAfter
    log.Printf("Rate limited, waiting %v", waitTime)
    time.Sleep(waitTime)

    // 重试
    return ag.Chat(ctx, message)
}
```

### 场景3: 网络超时

```go
// 设置超时
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := ag.Chat(ctx, message)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Printf("Request timeout after 30s")
        return handleTimeout()
    }
}
```

### 场景4: 并发错误

```go
// 安全的并发访问
var mu sync.Mutex
var errors []error

var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(idx int) {
        defer wg.Done()

        if err := process(idx); err != nil {
            mu.Lock()
            errors = append(errors, err)
            mu.Unlock()
        }
    }(i)
}
wg.Wait()

// 汇总错误
if len(errors) > 0 {
    log.Printf("Encountered %d errors", len(errors))
    for _, err := range errors {
        log.Printf("  - %v", err)
    }
}
```

## ✅ 检查清单

在部署到生产环境前，确保：

- [ ] 所有错误都被捕获和处理
- [ ] 错误信息包含足够的上下文
- [ ] 实现了重试机制（对于可恢复错误）
- [ ] 配置了错误监控和告警
- [ ] 有优雅降级策略
- [ ] 错误日志结构化且可搜索
- [ ] 设置了合理的超时时间
- [ ] 测试了各种错误场景

## 🔗 相关资源

- [性能优化](/best-practices/performance)
- [监控运维](/best-practices/monitoring)
- [API 参考 - 错误类型](/api-reference/errors)
