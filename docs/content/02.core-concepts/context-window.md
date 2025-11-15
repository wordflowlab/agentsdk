---
title: Context Window Management
description: 上下文窗口管理 - Token计数、智能压缩和自动优化
navigation:
  icon: i-lucide-panel-top
---

# 上下文窗口管理 (Context Window Management)

## 概述

Context Window Manager 是 AgentSDK 的核心组件，负责管理与 LLM 的对话上下文窗口。它提供自动 Token 计数、智能压缩和多种压缩策略，确保对话始终在 Token 预算内运行。

## 核心功能

- **自动 Token 计数** - 支持 GPT-4、Claude 等多种模型的 Token 估算
- **智能压缩策略** - 滑动窗口、优先级、Token 预算、混合策略
- **预算管理** - 可配置的 Token 预算和警告阈值
- **消息优先级** - 智能保留重要消息
- **压缩历史** - 记录每次压缩的详细信息
- **并发安全** - 支持多线程安全访问

## 快速开始

### 1. 创建 Window Manager

```go
package main

import (
    "context"
    "fmt"
    "github.com/wordflowlab/agentsdk/pkg/context"
)

func main() {
    // 创建 Token 计数器
    tokenCounter := context.NewGPT4Counter()

    // 创建压缩策略（滑动窗口）
    strategy := context.NewSlidingWindowStrategy(10) // 保留 10 条消息

    // 创建 Window Manager
    config := context.DefaultWindowManagerConfig()
    manager := context.NewContextWindowManager(config, tokenCounter, strategy)

    // 添加消息
    err := manager.AddMessage(context.Background(), context.Message{
        Role:    "user",
        Content: "Hello, how are you?",
    })
    if err != nil {
        panic(err)
    }

    // 获取统计信息
    stats := manager.GetStats()
    fmt.Printf("当前消息数: %d\n", stats.CurrentMessages)
    fmt.Printf("Token 使用: %d (%.1f%%)\n",
        stats.CurrentTokens, stats.UsagePercentage)
}
```

### 2. 配置 Token 预算

```go
config := context.WindowManagerConfig{
    Budget: context.TokenBudget{
        MaxTokens:        128000, // GPT-4 Turbo 上下文窗口
        ReservedTokens:   4096,   // 预留给输出
        WarningThreshold: 0.8,    // 80% 时发出警告
    },
    AutoCompress:         true,
    CompressionThreshold: 0.85, // 85% 时自动压缩
    MinMessagesToKeep:    3,
    AlwaysKeepSystem:     true,
    AlwaysKeepRecent:     2,
}
```

## Token 计数器

### 支持的模型

AgentSDK 提供多种预配置的 Token 计数器：

```go
// GPT 系列
gpt4Counter := context.NewGPT4Counter()
gpt4Counter := context.NewSimpleTokenCounter(context.GPT4Config)
gpt35Counter := context.NewSimpleTokenCounter(context.GPT35TurboConfig)

// Claude 系列
claudeCounter := context.NewClaudeCounter()
claudeCounter := context.NewSimpleTokenCounter(context.ClaudeOpusConfig)

// 自定义配置
customCounter := context.NewSimpleTokenCounter(context.ModelConfig{
    Name:              "custom-model",
    CharsPerToken:     4.0,
    BaseTokenOverhead: 3,
    RoleTokenCost:     1,
})
```

### Token 计数方法

```go
// 1. 单个文本计数
count, err := counter.Count(ctx, "Hello, world!")

// 2. 批量计数
texts := []string{"Message 1", "Message 2", "Message 3"}
counts, err := counter.CountBatch(ctx, texts)

// 3. 消息列表计数
messages := []context.Message{
    {Role: "system", Content: "You are helpful."},
    {Role: "user", Content: "Hello"},
}
totalTokens, err := counter.EstimateMessages(ctx, messages)
```

### 多模型支持

```go
// 创建多模型计数器
multi := context.NewMultiModelTokenCounter()

// 注册不同模型
multi.RegisterCounter("gpt-4", context.NewGPT4Counter())
multi.RegisterCounter("claude", context.NewClaudeCounter())

// 为特定模型计数
count, err := multi.CountForModel(ctx, "gpt-4-turbo", text)

// 自动模糊匹配
count, err := multi.CountForModel(ctx, "gpt-4-turbo-preview", text)
// 会匹配到 "gpt-4"
```

### 详细计数信息

```go
detailedCounter := context.NewDetailedTokenCounter(baseCounter)

estimate, err := detailedCounter.EstimateMessagesDetailed(ctx, messages)

fmt.Printf("总 Token: %d\n", estimate.TotalTokens)
fmt.Printf("内容 Token: %d\n", estimate.Breakdown["content"])
fmt.Printf("开销 Token: %d\n", estimate.Breakdown["overhead"])

for i, tokens := range estimate.MessageTokens {
    fmt.Printf("消息 %d: %d tokens\n", i, tokens)
}
```

## 压缩策略

### 1. 滑动窗口策略 (Sliding Window)

保留最近的 N 条消息，删除旧消息：

```go
strategy := context.NewSlidingWindowStrategy(10)

// 特点：
// - 简单高效
// - 保证消息顺序
// - 适合一般对话场景
```

### 2. 优先级策略 (Priority-Based)

根据消息优先级智能保留重要消息：

```go
calc := context.NewDefaultPriorityCalculator()
strategy := context.NewPriorityBasedStrategy(10, calc)

// 优先级计算考虑：
// - 最近度（越新越重要）
// - 角色（system > user > assistant）
// - 长度（较长消息可能包含更多信息）
```

**自定义优先级计算器**：

```go
type MyPriorityCalculator struct{}

func (c *MyPriorityCalculator) CalculatePriority(
    ctx context.Context,
    msg context.Message,
    position int,
    totalMessages int,
) (context.MessagePriority, float64) {
    // 自定义逻辑
    if strings.Contains(msg.Content, "重要") {
        return context.PriorityCritical, 1.0
    }
    return context.PriorityMedium, 0.5
}

strategy := context.NewPriorityBasedStrategy(10, &MyPriorityCalculator{})
```

### 3. Token 预算策略 (Token-Based)

从旧消息开始删除，直到满足 Token 预算：

```go
strategy := context.NewTokenBasedStrategy(tokenCounter, 0.7) // 目标 70%

// 特点：
// - 精确控制 Token 使用
// - 自动平衡消息数量和 Token 预算
// - 适合严格 Token 限制的场景
```

### 4. 混合策略 (Hybrid)

结合多种策略的优点：

```go
strategies := []context.CompressionStrategy{
    context.NewSlidingWindowStrategy(10),
    context.NewPriorityBasedStrategy(10, calc),
    context.NewTokenBasedStrategy(counter, 0.7),
}

weights := []float64{0.3, 0.5, 0.2} // 各策略权重

strategy := context.NewHybridStrategy(strategies, weights)

// 特点：
// - 综合多种策略优势
// - 更智能的消息选择
// - 适合复杂场景
```

## 高级用法

### 1. 自动压缩

```go
config := context.DefaultWindowManagerConfig()
config.AutoCompress = true
config.CompressionThreshold = 0.85 // 85% 触发压缩

manager := context.NewContextWindowManager(config, counter, strategy)

// 添加消息时自动检查并压缩
for _, msg := range messages {
    _ = manager.AddMessage(ctx, msg)
    // 当 Token 使用超过 85% 时自动触发压缩
}
```

### 2. 手动压缩

```go
// 手动触发压缩
err := manager.Compress(ctx)
if err != nil {
    log.Printf("压缩失败: %v", err)
}

// 查看压缩历史
history := manager.GetCompressionHistory()
for _, event := range history {
    fmt.Printf("压缩时间: %s\n", event.Timestamp)
    fmt.Printf("消息: %d -> %d\n", event.BeforeMessages, event.AfterMessages)
    fmt.Printf("Token: %d -> %d\n", event.BeforeTokens, event.AfterTokens)
    fmt.Printf("压缩率: %.2f%%\n", event.CompressionRatio*100)
}
```

### 3. 预算监控

```go
// 检查是否在预算内
if !manager.IsWithinBudget() {
    log.Println("警告：超出 Token 预算！")
}

// 检查是否应该警告
if manager.ShouldWarn() {
    usage := manager.GetUsagePercentage()
    log.Printf("警告：Token 使用率达到 %.1f%%\n", usage)
}

// 获取详细统计
stats := manager.GetStats()
fmt.Printf("当前消息数: %d\n", stats.CurrentMessages)
fmt.Printf("累计消息数: %d\n", stats.TotalMessages)
fmt.Printf("当前 Token: %d\n", stats.CurrentTokens)
fmt.Printf("剩余 Token: %d\n", stats.RemainingTokens)
fmt.Printf("使用百分比: %.1f%%\n", stats.UsagePercentage)
fmt.Printf("压缩次数: %d\n", stats.CompressionCount)
```

### 4. 消息优先级管理

```go
calc := context.NewDefaultPriorityCalculator()

// 计算所有消息的优先级
messagesWithPriority := context.CalculateMessagePriorities(ctx, messages, calc)

// 按优先级排序
context.SortMessagesByPriority(messagesWithPriority, true) // 降序

// 过滤低优先级消息
highPriorityMessages := context.FilterMessagesByPriority(
    messagesWithPriority,
    context.PriorityHigh,
)

for _, msg := range highPriorityMessages {
    fmt.Printf("[%d] %.2f - %s: %s\n",
        msg.Priority, msg.Score, msg.Message.Role, msg.Message.Content)
}
```

### 5. 清空和重置

```go
// 清空消息（保留历史）
manager.Clear()

// 完全重置（包括历史）
manager.Reset()
```

## 最佳实践

### 1. 选择合适的压缩策略

- **一般对话**: 使用 `SlidingWindowStrategy`
- **重要信息保留**: 使用 `PriorityBasedStrategy`
- **严格 Token 限制**: 使用 `TokenBasedStrategy`
- **复杂场景**: 使用 `HybridStrategy`

### 2. Token 预算配置

```go
// 生产环境建议配置
config.Budget.MaxTokens = 128000       // 根据模型调整
config.Budget.ReservedTokens = 4096    // 为输出预留充足空间
config.Budget.WarningThreshold = 0.8   // 提前警告

config.CompressionThreshold = 0.85     // 留有缓冲空间
```

### 3. 保留关键消息

```go
config.AlwaysKeepSystem = true  // 始终保留 system prompt
config.AlwaysKeepRecent = 2     // 保留最近 2 条消息（上下文连贯）
config.MinMessagesToKeep = 3    // 最少保留 3 条消息
```

### 4. 监控和日志

```go
// 定期检查使用情况
ticker := time.NewTicker(1 * time.Minute)
go func() {
    for range ticker.C {
        stats := manager.GetStats()
        if stats.ShouldWarn {
            log.Printf("警告：Token 使用率 %.1f%%\n", stats.UsagePercentage)
        }
    }
}()

// 记录压缩事件
history := manager.GetCompressionHistory()
if len(history) > 0 {
    last := history[len(history)-1]
    log.Printf("最近压缩: 消息 %d->%d, Token %d->%d, 策略: %s\n",
        last.BeforeMessages, last.AfterMessages,
        last.BeforeTokens, last.AfterTokens,
        last.Strategy)
}
```

## 配置参考

### WindowManagerConfig

```go
type WindowManagerConfig struct {
    Budget               TokenBudget  // Token 预算配置
    AutoCompress         bool         // 是否自动压缩
    CompressionThreshold float64      // 压缩触发阈值 (0.0-1.0)
    MinMessagesToKeep    int          // 最少保留消息数
    AlwaysKeepSystem     bool         // 始终保留 system 消息
    AlwaysKeepRecent     int          // 始终保留最近 N 条消息
    EnablePrioritization bool         // 是否启用优先级
}
```

### TokenBudget

```go
type TokenBudget struct {
    MaxTokens        int     // 最大 Token 数
    ReservedTokens   int     // 预留 Token 数
    WarningThreshold float64 // 警告阈值 (0.0-1.0)
}

// 常用配置
DefaultTokenBudget() // GPT-4 Turbo 默认配置 (128K)
```

### ModelConfig

```go
type ModelConfig struct {
    Name              string  // 模型名称
    CharsPerToken     float64 // 平均字符/Token
    BaseTokenOverhead int     // 每条消息基础开销
    RoleTokenCost     int     // Role 标签开销
}

// 预定义配置
GPT4Config, GPT4TurboConfig, GPT35TurboConfig
ClaudeConfig, ClaudeOpusConfig, ClaudeSonnetConfig
DefaultConfig
```

## 性能优化

### 1. Token 计数缓存

```go
// Token 计数是轻量级的，但可以缓存结果
type CachedCounter struct {
    base  context.TokenCounter
    cache map[string]int
    mu    sync.RWMutex
}
```

### 2. 批量操作

```go
// 使用 AddMessages 代替多次 AddMessage
manager.AddMessages(ctx, messages) // 更高效
```

### 3. 异步压缩

```go
// 对于非关键路径，可以异步压缩
go func() {
    if manager.ShouldWarn() {
        _ = manager.Compress(context.Background())
    }
}()
```

## 测试

运行 Context Window Manager 测试：

```bash
go test ./pkg/context/... -v
```

## 相关文档

- [Memory System](/memory) - 记忆系统集成
- [Middleware](/middleware) - 中间件系统
- [Best Practices](/best-practices) - 最佳实践

## 参考资源

- [Google Context Engineering Whitepaper](https://arxiv.org/abs/2410.01600)
- [OpenAI Tokenizer](https://platform.openai.com/tokenizer)
- [Claude Token Counting](https://docs.anthropic.com/claude/docs/models-overview)
