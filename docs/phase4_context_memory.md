# Phase 4: 上下文管理与记忆系统

本文档介绍 AgentSDK Phase 4 新增的两个核心中间件:**SummarizationMiddleware** 和 **AgentMemoryMiddleware**。

## 目录

- [概述](#概述)
- [SummarizationMiddleware](#summarizationmiddleware)
- [AgentMemoryMiddleware](#agentmemorymiddleware)
- [集成指南](#集成指南)
- [最佳实践](#最佳实践)
- [常见问题](#常见问题)

---

## 概述

Phase 4 解决了长对话和个性化 Agent 的两个核心问题:

1. **上下文窗口管理**: 当对话历史超过模型上下文限制时,如何保持对话连贯性?
2. **Agent 记忆持久化**: 如何让 Agent 记住用户偏好和个性化设置?

### 适用场景

- ✅ 需要支持长时间对话的应用(如代码助手、客服机器人)
- ✅ 需要个性化 Agent 行为的应用
- ✅ 多会话间需要保持一致性的应用
- ✅ 需要学习用户偏好的应用

---

## SummarizationMiddleware

### 功能特性

自动监控对话历史的 token 数量,当超过阈值时触发总结,将旧对话压缩为摘要,保留最近的消息。

**核心特性**:
- 🔍 实时 token 监控(默认每 4 个字符 ≈ 1 token)
- 🤖 可插拔的总结生成器(支持自定义 LLM 调用)
- 📦 智能消息保留(区分 system messages 和常规消息)
- 🛡️ 错误容错(总结失败时保留原始消息)
- 📊 统计和监控(总结触发次数)

### 快速开始

```go
import (
    "github.com/wordflowlab/agentsdk/pkg/middleware"
    "github.com/wordflowlab/agentsdk/pkg/types"
)

// 1. 创建自定义总结器(使用真实 LLM)
customSummarizer := func(ctx context.Context, messages []types.Message) (string, error) {
    // 调用 LLM 生成总结
    return llmProvider.Summarize(ctx, messages)
}

// 2. 创建中间件
summarizationMW, err := middleware.NewSummarizationMiddleware(&middleware.SummarizationMiddlewareConfig{
    Summarizer:             customSummarizer,
    MaxTokensBeforeSummary: 170000,  // 170k tokens
    MessagesToKeep:         6,       // 保留最近 6 条
    SummaryPrefix:          "## Previous conversation summary:",
})
```

### 配置参数详解

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Summarizer` | `SummarizerFunc` | defaultSummarizer | 总结生成函数,建议生产环境使用真实 LLM |
| `MaxTokensBeforeSummary` | `int` | 170000 | 触发总结的 token 阈值,建议设为模型窗口的 85% |
| `MessagesToKeep` | `int` | 6 | 总结后保留的最近消息数量(不含 system messages) |
| `SummaryPrefix` | `string` | "## Previous..." | 总结消息的前缀标记 |
| `TokenCounter` | `TokenCounterFunc` | defaultTokenCounter | 自定义 token 计数函数 |

### 工作流程

```
1. WrapModelCall 被调用
   ↓
2. 计算当前消息的 token 数
   ↓
3. 判断是否超过阈值?
   ├─ 否 → 直接调用下一层
   └─ 是 → 继续
       ↓
4. 分离 system messages 和常规消息
   ↓
5. 检查常规消息数量 > MessagesToKeep?
   ├─ 否 → 跳过总结
   └─ 是 → 继续
       ↓
6. 调用 Summarizer 生成总结
   ├─ 失败 → 保留原始消息
   └─ 成功 → 继续
       ↓
7. 构建新消息列表:
   [system messages] + [总结消息] + [最近 N 条消息]
   ↓
8. 更新请求并调用下一层
```

### 使用示例

#### 基础用法(使用默认总结器)

```go
middleware, _ := middleware.NewSummarizationMiddleware(&middleware.SummarizationMiddlewareConfig{
    MaxTokensBeforeSummary: 100000,
    MessagesToKeep:         10,
})
```

#### 生产环境用法(自定义 LLM 总结)

```go
// 使用 Anthropic Claude 生成总结
customSummarizer := func(ctx context.Context, messages []types.Message) (string, error) {
    summaryPrompt := `Provide a concise summary (200-300 words) of the following conversation.
Focus on: main topics, key decisions, action items, and technical details.`

    summaryMessages := []types.Message{
        {Role: types.MessageRoleSystem, Content: []types.ContentBlock{&types.TextBlock{Text: summaryPrompt}}},
    }
    summaryMessages = append(summaryMessages, messages...)

    resp, err := provider.Stream(ctx, summaryMessages, &provider.StreamOptions{
        Temperature: 0.3,  // 低温度保证稳定性
        MaxTokens:   500,
    })
    if err != nil {
        return "", err
    }

    // 收集流式响应
    var summary strings.Builder
    for chunk := range resp {
        if chunk.Delta != nil {
            summary.WriteString(chunk.Delta.(string))
        }
    }

    return summary.String(), nil
}

middleware, _ := middleware.NewSummarizationMiddleware(&middleware.SummarizationMiddlewareConfig{
    Summarizer:             customSummarizer,
    MaxTokensBeforeSummary: 170000,
    MessagesToKeep:         6,
})
```

#### 自定义 Token 计数器(使用官方 tokenizer)

```go
// 使用模型的官方 tokenizer
customTokenCounter := func(messages []types.Message) int {
    totalTokens := 0
    for _, msg := range messages {
        // 使用官方 tokenizer
        tokens := anthropic.CountTokens(msg)
        totalTokens += tokens
    }
    return totalTokens
}

middleware, _ := middleware.NewSummarizationMiddleware(&middleware.SummarizationMiddlewareConfig{
    TokenCounter:           customTokenCounter,
    MaxTokensBeforeSummary: 190000,  // 更精确的阈值
    MessagesToKeep:         6,
})
```

### 监控和调试

```go
// 获取配置信息
config := middleware.GetConfig()
fmt.Printf("Max Tokens: %v\n", config["max_tokens_before_summary"])
fmt.Printf("Messages to Keep: %v\n", config["messages_to_keep"])

// 获取总结触发次数
count := middleware.GetSummarizationCount()
fmt.Printf("Total Summarizations: %d\n", count)

// 重置计数器
middleware.ResetSummarizationCount()

// 动态更新配置
middleware.UpdateConfig(200000, 8)
```

---

## AgentMemoryMiddleware

### 功能特性

从后端存储加载 Agent 的个性化设置(默认从 `/agent.md`),并注入到每次模型调用的 System Prompt 中。

**核心特性**:
- 📁 灵活的存储后端(支持 Filesystem、Store、Composite)
- 🔄 懒加载机制(首次使用时自动加载)
- 📝 自动注入到 System Prompt
- 📚 内置长期记忆使用指南
- 🔃 支持重新加载(ReloadMemory)

### 快速开始

```go
import (
    "github.com/wordflowlab/agentsdk/pkg/middleware"
    "github.com/wordflowlab/agentsdk/pkg/backends"
)

// 1. 创建后端
composite := backends.NewCompositeBackend([]backends.Route{
    {Prefix: "/memories/", Backend: storeBackend},
    {Prefix: "/", Backend: filesystemBackend},
})

// 2. 创建中间件
memoryMW, err := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
    Backend:    composite,
    MemoryPath: "/memories/",
})
```

### 配置参数详解

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Backend` | `BackendProtocol` | (必需) | 存储后端,建议使用 CompositeBackend |
| `MemoryPath` | `string` | "/memories/" | 长期记忆文件的路径前缀 |
| `SystemPromptTemplate` | `string` | `<agent_memory>...</agent_memory>` | 记忆注入模板 |

### agent.md 文件结构

创建 `/agent.md` 文件,定义 Agent 的个性和行为:

```markdown
# Agent Personality

You are Claude, a helpful AI assistant specializing in software development.

## Core Principles

1. **Code Quality**: Always write clean, maintainable code
2. **Testing**: Write tests before implementing features (TDD)
3. **Security**: Check for vulnerabilities (SQL injection, XSS, CSRF)
4. **Documentation**: Provide clear comments and documentation

## User Preferences

- **Programming Languages**: Prefers Go > Python > JavaScript
- **Testing Framework**: Uses table-driven tests in Go
- **Code Style**: Follows official Go style guide
- **Commit Messages**: Prefers conventional commits format

## Project Context

- Working on AgentSDK, a Go-based AI agent framework
- Focus: Production-ready, well-tested middleware
- Recent work: Implemented Phase 4 (Context Management)

## Learnings from Past Interactions

- User values incremental progress over big-bang changes
- User prefers explicit error handling over silent failures
- User likes detailed logging for debugging
```

### 注入格式

AgentMemoryMiddleware 会将 agent.md 内容按以下格式注入:

```
<agent_memory>
{agent.md 的完整内容}
</agent_memory>

{原始 system_prompt}

## Long-term Memory

You have access to a long-term memory system...

### When to Check Memory
- At the start of a new session
- Before answering questions about previous work
...

### Usage Examples
# List available memory files
ls /memories/

# Read specific memory
read_file '/memories/agent.md'
...
```

### 使用示例

#### 基础用法

```go
// 1. 创建 Filesystem Backend
fsBackend, _ := backends.NewFilesystemBackend(&backends.FilesystemConfig{
    RootDir: "/path/to/workspace",
})

// 2. 创建中间件
memoryMW, _ := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
    Backend:    fsBackend,
    MemoryPath: "/memories/",
})

// 3. agent.md 会在首次 WrapModelCall 时自动加载
```

#### 使用 CompositeBackend(推荐)

```go
// 将记忆文件存储在 Store(持久化),其他文件在 Filesystem
composite := backends.NewCompositeBackend([]backends.Route{
    {
        Prefix:  "/memories/",
        Backend: storeBackend,  // 持久化存储,跨会话保留
    },
    {
        Prefix:  "/",
        Backend: filesystemBackend,  // 工作区文件
    },
})

memoryMW, _ := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
    Backend:    composite,
    MemoryPath: "/memories/",
})
```

#### 自定义注入模板

```go
memoryMW, _ := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
    Backend:              backend,
    MemoryPath:           "/memories/",
    SystemPromptTemplate: "### Agent Configuration\n%s\n### End Configuration",
})
```

#### 手动重新加载记忆

```go
// Agent 运行过程中更新了 agent.md
err := memoryMW.ReloadMemory(ctx)
if err != nil {
    log.Printf("Failed to reload memory: %v", err)
}
```

### 查询记忆状态

```go
// 检查记忆是否已加载
if memoryMW.IsMemoryLoaded() {
    content := memoryMW.GetMemoryContent()
    fmt.Printf("Memory loaded: %d chars\n", len(content))
}

// 获取配置
config := memoryMW.GetConfig()
fmt.Printf("Memory Path: %v\n", config["memory_path"])
fmt.Printf("Memory File: %v\n", config["memory_file"])
fmt.Printf("Memory Size: %v\n", config["memory_size"])
```

---

## 集成指南

### 完整的中间件栈示例

```go
package main

import (
    "context"
    "github.com/wordflowlab/agentsdk/pkg/agent"
    "github.com/wordflowlab/agentsdk/pkg/middleware"
    "github.com/wordflowlab/agentsdk/pkg/backends"
)

func createAgent() (*agent.Agent, error) {
    ctx := context.Background()

    // 1. 创建存储后端
    composite := backends.NewCompositeBackend([]backends.Route{
        {Prefix: "/memories/", Backend: storeBackend},
        {Prefix: "/", Backend: filesystemBackend},
    })

    // 2. 创建 AgentMemoryMiddleware (优先级 5,最早执行)
    memoryMW, _ := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
        Backend:    composite,
        MemoryPath: "/memories/",
    })

    // 3. 创建自定义 Summarizer
    customSummarizer := func(ctx context.Context, messages []types.Message) (string, error) {
        return generateSummaryWithLLM(ctx, messages)
    }

    // 4. 创建 SummarizationMiddleware (优先级 40)
    summarizationMW, _ := middleware.NewSummarizationMiddleware(&middleware.SummarizationMiddlewareConfig{
        Summarizer:             customSummarizer,
        MaxTokensBeforeSummary: 170000,
        MessagesToKeep:         6,
    })

    // 5. 构建中间件栈
    middlewares := []middleware.Middleware{
        memoryMW,                       // 优先级 5: 注入记忆
        middleware.NewTodoListMiddleware(nil),  // 优先级 10: 任务管理
        summarizationMW,                // 优先级 40: 上下文管理
        middleware.NewPatchToolCallsMiddleware(nil), // 优先级 50: 错误恢复
    }

    // 6. 创建 Agent 配置
    config := &types.AgentConfig{
        AgentID:    "my-agent",
        TemplateID: "default",
        // ... 其他配置
    }

    // 7. 创建 Agent
    return agent.Create(ctx, config, &agent.Dependencies{
        // ... 依赖注入
    })
}
```

### 中间件执行顺序

```
请求流入:
  AgentMemoryMiddleware (优先级 5)
    ↓ 注入 agent.md 到 system prompt
  TodoListMiddleware (优先级 10)
    ↓ 提供 write_todos 工具
  SummarizationMiddleware (优先级 40)
    ↓ 检查并总结对话历史
  PatchToolCallsMiddleware (优先级 50)
    ↓ 错误恢复
  → 模型调用

响应返回:
  模型响应
    ↓
  PatchToolCallsMiddleware
    ↓
  SummarizationMiddleware
    ↓
  TodoListMiddleware
    ↓
  AgentMemoryMiddleware
    ↓
  返回给用户
```

---

## 最佳实践

### SummarizationMiddleware

#### ✅ 推荐做法

1. **使用真实 LLM 生成总结**
   ```go
   // 生产环境: 使用模型生成总结
   customSummarizer := func(ctx context.Context, messages []types.Message) (string, error) {
       return llmProvider.Summarize(ctx, messages)
   }
   ```

2. **设置合理的阈值**
   ```go
   // Claude 3.5 Sonnet: 200k 上下文
   MaxTokensBeforeSummary: 170000  // 85% 的上下文窗口

   // GPT-4 Turbo: 128k 上下文
   MaxTokensBeforeSummary: 110000  // 85% 的上下文窗口
   ```

3. **监控总结频率**
   ```go
   count := middleware.GetSummarizationCount()
   if count > 20 {
       log.Warn("High summarization frequency, consider adjusting threshold")
   }
   ```

4. **使用便宜的模型做总结**
   ```go
   // 使用 Claude 3 Haiku 生成总结(成本更低)
   summarizer := createHaikuSummarizer()
   ```

#### ❌ 避免的做法

1. ❌ 在生产环境使用默认总结器(太简单)
2. ❌ 设置过低的阈值(频繁总结影响性能)
3. ❌ 保留过多消息(失去总结的意义)
4. ❌ 忽略总结失败(应该有降级策略)

### AgentMemoryMiddleware

#### ✅ 推荐做法

1. **结构化 agent.md**
   ```markdown
   # 使用清晰的标题结构
   ## Core Principles
   ## User Preferences
   ## Project Context
   ## Learnings
   ```

2. **版本控制**
   ```bash
   git add agent.md
   git commit -m "Update agent personality based on user feedback"
   ```

3. **定期更新**
   ```go
   // 从用户反馈中学习
   if userGaveFeedback {
       updateAgentMemory(feedback)
       memoryMW.ReloadMemory(ctx)
   }
   ```

4. **使用 CompositeBackend**
   ```go
   // 记忆文件持久化,工作文件临时存储
   composite := backends.NewCompositeBackend([]backends.Route{
       {Prefix: "/memories/", Backend: storeBackend},
       {Prefix: "/", Backend: filesystemBackend},
   })
   ```

#### ❌ 避免的做法

1. ❌ agent.md 过长(超过 2000 字)
2. ❌ 包含敏感信息(密码、API Key)
3. ❌ 从不更新(失去学习能力)
4. ❌ 多个 Agent 共享同一个 agent.md(应该各自独立)

---

## 常见问题

### Q1: 总结会丢失重要信息吗?

**A**: 可能会。缓解方法:
- 使用高质量的 LLM 生成总结
- 在总结提示词中强调"保留关键技术细节"
- 增加 `MessagesToKeep` 保留更多最近消息
- 对关键对话轮次打标记,强制保留

### Q2: agent.md 应该多长?

**A**: 建议 500-2000 字:
- 太短:无法表达完整的个性
- 太长:占用过多 token,影响性能
- 如果内容过多,考虑拆分为多个文件

### Q3: 如何测试中间件是否工作?

**A**: 使用测试代码验证:

```go
func TestMiddlewareIntegration(t *testing.T) {
    // 创建测试用的 backend
    backend := createTestBackend()
    backend.Write(ctx, "/agent.md", "Test personality")

    // 创建中间件
    memoryMW, _ := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
        Backend: backend,
    })

    // 创建模拟请求
    req := &middleware.ModelRequest{
        SystemPrompt: "Original prompt",
    }

    // 调用中间件
    handler := func(ctx context.Context, req *middleware.ModelRequest) (*middleware.ModelResponse, error) {
        // 验证 system prompt 包含记忆
        if !strings.Contains(req.SystemPrompt, "Test personality") {
            t.Error("Memory not injected")
        }
        return &middleware.ModelResponse{}, nil
    }

    memoryMW.WrapModelCall(ctx, req, handler)
}
```

### Q4: 总结器失败时会怎样?

**A**: 中间件会保留原始消息,记录错误日志,但不会中断请求:

```go
if err != nil {
    log.Printf("[SummarizationMiddleware] Failed to generate summary: %v, keeping original messages", err)
    return handler(ctx, req) // 保留原始消息继续
}
```

### Q5: 如何优化性能?

**A**: 几个优化建议:

1. **Token 计数优化**:
   ```go
   // 使用缓存避免重复计算
   var cachedTokenCount int
   var cachedMessagesHash string

   customTokenCounter := func(messages []types.Message) int {
       hash := calculateHash(messages)
       if hash == cachedMessagesHash {
           return cachedTokenCount
       }
       // 计算 token...
   }
   ```

2. **异步总结**:
   ```go
   // 后台异步生成总结,不阻塞主流程
   go func() {
       summary := generateSummary(messages)
       cache.Set("summary_"+conversationID, summary)
   }()
   ```

3. **使用更快的模型**:
   ```go
   // Claude 3 Haiku: 更快,更便宜
   summarizer := createHaikuSummarizer()
   ```

### Q6: 多用户场景如何处理?

**A**: 每个用户应该有独立的 agent.md:

```go
// 使用用户 ID 作为路径前缀
userMemoryPath := fmt.Sprintf("/memories/%s/", userID)

memoryMW, _ := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
    Backend:    composite,
    MemoryPath: userMemoryPath,
})
```

---

## 总结

Phase 4 的两个中间件为 AgentSDK 带来了:

✅ **长对话支持**: 自动管理上下文,支持无限长度的对话
✅ **个性化能力**: 记住用户偏好,提供一致的体验
✅ **生产就绪**: 完善的错误处理和监控能力
✅ **灵活可扩展**: 可插拔的设计,易于定制

开始使用 Phase 4 功能,让你的 Agent 更智能、更个性化!

---

**相关文档**:
- [GAP_CLOSURE.md](../GAP_CLOSURE.md) - 完整的功能对比和实现报告
- [examples/phase4_integration.go](../examples/phase4_integration.go) - 集成示例代码
- [pkg/middleware/summarization.go](../pkg/middleware/summarization.go) - 源代码
- [pkg/middleware/agent_memory.go](../pkg/middleware/agent_memory.go) - 源代码

**问题反馈**:
- GitHub Issues: [wordflowlab/agentsdk/issues](https://github.com/wordflowlab/agentsdk/issues)
