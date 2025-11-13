---
title: 内置中间件
description: 使用 AgentSDK 提供的内置中间件扩展 Agent 能力
---

# 内置中间件使用指南

AgentSDK 提供了多个开箱即用的中间件，涵盖对话管理、文件操作、任务委托、人工审批等常见场景。

## 📦 中间件列表

| 中间件 | 优先级 | 功能 | 适用场景 |
|--------|--------|------|----------|
| [Summarization](#summarization) | 40 | 自动总结 | 长对话压缩 |
| [Filesystem](#filesystem) | 100 | 文件工具 | 文件读写操作 |
| [SubAgent](#subagent) | 200 | 子Agent | 复杂任务委托 |
| [AgentMemory](#memory) | 150 | 跨会话记忆 | 持久化记忆 |
| [HumanInTheLoop](#hitl) | 50 | 人工审批 | 敏感操作控制 |
| [TodoList](#todolist) | 120 | 任务跟踪 | 任务规划管理 |
| [PatchToolCalls](#patch) | 300 | 工具补丁 | 兼容性修复 |

## <a id="summarization"></a>📝 Summarization - 自动总结

**功能**: 自动监控对话历史，超过 Token 阈值时总结旧消息，释放上下文空间。

**使用场景**:
- 长时间对话（超过模型上下文窗口）
- 需要保留对话要点但减少 Token 使用
- 多轮交互的复杂任务

### 配置

```go
import "github.com/wordflowlab/agentsdk/pkg/middleware"

summaryMW, err := middleware.NewSummarizationMiddleware(&middleware.SummarizationMiddlewareConfig{
    MaxTokensBeforeSummary: 150000,  // 触发阈值（默认 170000）
    MessagesToKeep:         6,       // 保留最近消息数（默认 6）
    SummaryPrefix:          "## 对话历史总结:",  // 总结前缀
    TokenCounter:           customCounter,      // 自定义 Token 计数器（可选）
    Summarizer:             customSummarizer,   // 自定义总结器（可选）
})
```

### 参数说明

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| MaxTokensBeforeSummary | int | 170000 | 触发总结的 Token 数阈值 |
| MessagesToKeep | int | 6 | 总结后保留的最近消息数 |
| SummaryPrefix | string | "## Previous conversation summary:" | 总结消息的前缀 |
| TokenCounter | func | 默认计数器 | 自定义 Token 计数函数 |
| Summarizer | func | 默认总结器 | 自定义总结生成函数 |

### 工作流程

<Mermaid>
sequenceDiagram
    participant Agent
    participant MW as Summary MW
    participant LLM

    Agent->>MW: Chat("消息X")
    MW->>MW: 计算 Token 数
    alt Token < 阈值
        MW->>LLM: 正常调用
    else Token >= 阈值
        MW->>MW: 保留最近 N 条消息
        MW->>LLM: 请求总结旧消息
        LLM->>MW: 返回总结
        MW->>MW: 替换旧消息为总结
        MW->>LLM: 使用压缩后的历史调用
    end
    LLM->>MW: 返回响应
    MW->>Agent: 返回结果
</Mermaid>

### 使用示例

```go
func main() {
    // 创建中间件
    summaryMW, _ := middleware.NewSummarizationMiddleware(&middleware.SummarizationMiddlewareConfig{
        MaxTokensBeforeSummary: 100000,
        MessagesToKeep:         4,
    })

    // 注册到 Stack
    stack := middleware.NewStack()
    stack.Use(summaryMW)

    // 创建 Agent
    ag, _ := agent.Create(ctx, config, &agent.Dependencies{
        MiddlewareStack: stack,
        // ... 其他依赖
    })

    // 长对话示例
    for i := 0; i < 100; i++ {
        result, _ := ag.Chat(ctx, fmt.Sprintf("第 %d 轮对话", i))
        fmt.Printf("响应: %s\n", result.Text)
    }
    // → 超过阈值后自动总结，不会因上下文过长而失败
}
```

### 自定义总结器

```go
customSummarizer := func(ctx context.Context, messages []types.Message) (string, error) {
    // 自定义总结逻辑
    var summary strings.Builder
    summary.WriteString("## 对话要点:\n")

    for _, msg := range messages {
        if msg.Role == types.RoleUser {
            summary.WriteString(fmt.Sprintf("- 用户提问: %s\n", truncate(msg.Content)))
        }
    }

    return summary.String(), nil
}

summaryMW, _ := middleware.NewSummarizationMiddleware(&middleware.SummarizationMiddlewareConfig{
    Summarizer: customSummarizer,
})
```

---

## <a id="filesystem"></a>📁 Filesystem - 文件系统

**功能**: 注入文件操作工具，支持文件读写、目录遍历、内容搜索等。

**使用场景**:
- Agent 需要读写文件
- 处理大文件内容（自动驱逐大结果）
- 路径安全验证

### 配置

```go
import (
    "github.com/wordflowlab/agentsdk/pkg/middleware"
    "github.com/wordflowlab/agentsdk/pkg/backends"
)

filesMW := middleware.NewFilesystemMiddleware(&middleware.FilesystemMiddlewareConfig{
    Backend:              backends.NewLocalBackend("./workspace"),
    TokenLimit:           20000,         // 大结果驱逐阈值
    EnableEviction:       true,          // 启用自动驱逐
    AllowedPathPrefixes:  []string{"/workspace", "/tmp"},  // 允许的路径
    EnablePathValidation: true,          // 启用路径验证
})
```

### 注入的工具

| 工具名 | 功能 | 参数 |
|--------|------|------|
| `fs_read` | 读取文件 | path, offset?, limit? |
| `fs_write` | 写入文件 | path, content |
| `fs_edit` | 编辑文件 | path, old_string, new_string |
| `glob` | 文件模式匹配 | pattern |
| `grep` | 内容搜索 | pattern, path |

### 使用示例

```go
func main() {
    // 创建后端
    backend := backends.NewLocalBackend("./workspace")

    // 创建中间件
    filesMW := middleware.NewFilesystemMiddleware(&middleware.FilesystemMiddlewareConfig{
        Backend:    backend,
        TokenLimit: 15000,  // 超过 15k tokens 的结果保存到文件
    })

    stack := middleware.NewStack()
    stack.Use(filesMW)

    ag, _ := agent.Create(ctx, config, &agent.Dependencies{
        MiddlewareStack: stack,
    })

    // Agent 自动拥有文件操作能力
    ag.Chat(ctx, "请读取 config.json 文件")
    ag.Chat(ctx, "请在 data/ 目录创建 output.txt 文件")
    ag.Chat(ctx, "请查找所有包含 'error' 的日志文件")
}
```

### 大结果驱逐

当工具返回的内容超过 TokenLimit 时，自动保存到文件：

```go
// Agent 读取大文件
ag.Chat(ctx, "请读取 large-log.txt 文件")
// → fs_read 返回内容 > 20k tokens
// → 中间件自动保存到 .agent-outputs/output-xxx.txt
// → 返回给 LLM: "结果已保存到文件: .agent-outputs/output-xxx.txt，使用 fs_read 工具读取"
```

---

## <a id="subagent"></a>🤖 SubAgent - 子 Agent

**功能**: 提供 `task` 工具，允许主 Agent 委托任务给子 Agent 执行。

**使用场景**:
- 复杂任务分解
- 专业化分工（数据分析、代码审查等）
- 任务上下文隔离

### 配置

```go
subagentMW, _ := middleware.NewSubAgentMiddleware(&middleware.SubAgentMiddlewareConfig{
    Factory: mySubAgentFactory,  // 子 Agent 工厂函数
    EnableParallel: false,       // 是否支持并行（默认 false）
    EnableGeneralPurpose: true,  // 是否启用通用子 Agent（默认 true）
    Specs: []middleware.SubAgentSpec{
        {
            Name:        "data-analyst",
            Description: "数据分析专家，擅长处理和分析数据",
            Prompt:      "你是数据分析专家...",
            Tools:       []string{"python_exec", "pandas_query"},
        },
        {
            Name:        "code-reviewer",
            Description: "代码审查专家，检查代码质量和安全问题",
            Prompt:      "你是代码审查专家...",
            Tools:       []string{"fs_read", "grep"},
        },
    },
})
```

### 使用示例

```go
// 主 Agent 委托任务
ag.Chat(ctx, "请分析 sales-data.csv 文件，生成月度报告")
// → Agent 调用: task(
//     agent_name="data-analyst",
//     task="分析 sales-data.csv 并生成月度报告"
//   )
// → 子 Agent 独立执行任务
// → 返回结果给主 Agent
```

### 任务委托流程

<Mermaid>
sequenceDiagram
    participant User
    participant MainAgent as 主 Agent
    participant Task as task 工具
    participant SubAgent as 子 Agent
    participant Tools

    User->>MainAgent: "分析数据并生成报告"
    MainAgent->>Task: task(agent="data-analyst", ...)
    Task->>SubAgent: 创建/获取子 Agent
    SubAgent->>Tools: 调用专业工具
    Tools->>SubAgent: 返回结果
    SubAgent->>Task: 任务完成
    Task->>MainAgent: 返回结果
    MainAgent->>User: 综合响应
</Mermaid>

---

## <a id="hitl"></a>👤 HumanInTheLoop - 人工审批

**功能**: 拦截指定工具的调用，要求人工审批后才执行。

**使用场景**:
- 敏感操作（删除文件、执行命令）
- 外部 API 调用需要确认
- 数据修改需要审核

### 配置

```go
hitlMW, _ := middleware.NewHumanInTheLoopMiddleware(&middleware.HumanInTheLoopMiddlewareConfig{
    InterruptOn: map[string]interface{}{
        "bash_run":     true,  // 启用默认审批
        "http_request": true,
        "fs_write": middleware.InterruptConfig{
            Enabled: true,
            Message: "文件写入需要审批",
        },
    },
    ApprovalHandler: func(ctx context.Context, req *middleware.ReviewRequest) ([]middleware.Decision, error) {
        // 自定义审批逻辑
        for _, action := range req.ActionRequests {
            fmt.Printf("工具: %s\n", action.ToolName)
            fmt.Printf("参数: %+v\n", action.Input)
            fmt.Print("批准? (y/n/edit): ")

            var answer string
            fmt.Scanln(&answer)

            switch answer {
            case "y":
                return []middleware.Decision{{Type: middleware.DecisionApprove}}, nil
            case "n":
                return []middleware.Decision{{Type: middleware.DecisionReject, Reason: "用户拒绝"}}, nil
            case "edit":
                // 修改参数
                editedInput := make(map[string]interface{})
                // ... 收集编辑后的参数
                return []middleware.Decision{{
                    Type:        middleware.DecisionEdit,
                    EditedInput: editedInput,
                }}, nil
            }
        }
        return nil, fmt.Errorf("未知决策")
    },
})
```

### 审批流程

```go
ag.Chat(ctx, "请删除 /tmp/data.txt 文件")
// → Agent 调用 bash_run("rm /tmp/data.txt")
// → HITL 中间件拦截
// → 显示审批请求给用户
// → 用户批准/拒绝/编辑
// → 根据决策执行或跳过
```

---

## <a id="todolist"></a>✅ TodoList - 任务列表

**功能**: 注入 `write_todos` 工具，引导 Agent 进行任务规划和跟踪。

**使用场景**:
- 复杂多步骤任务
- 任务进度追踪
- 用户实时了解 Agent 进展

### 配置

```go
todoMW := middleware.NewTodoListMiddleware(&middleware.TodoListMiddlewareConfig{
    EnableSystemPrompt: true,  // 注入任务管理提示词
    StoreGetter: func() interface{} {
        // 从外部获取任务列表（可选）
        return currentTodos
    },
    StoreSetter: func(todos []middleware.TodoItem) {
        // 保存任务列表（可选）
        currentTodos = todos
    },
})
```

### 使用示例

```go
ag.Chat(ctx, "请创建一个完整的博客网站")
// → Agent 自动使用 write_todos 规划任务:
//   1. 设计网站结构
//   2. 创建 HTML 模板
//   3. 编写 CSS 样式
//   4. 实现路由逻辑
//   5. 测试网站功能

// → 执行过程中更新任务状态:
//   1. [completed] 设计网站结构
//   2. [in_progress] 创建 HTML 模板
//   3. [pending] 编写 CSS 样式
//   ...
```

### 任务状态

```go
type TodoStatus string

const (
    TodoStatusPending    TodoStatus = "pending"      // 待处理
    TodoStatusInProgress TodoStatus = "in_progress"  // 进行中
    TodoStatusCompleted  TodoStatus = "completed"    // 已完成
)
```

---

## <a id="memory"></a>🧠 AgentMemory - 跨会话记忆

**功能**: 持久化 Agent 记忆，跨会话保留重要信息。

**使用场景**:
- 多次对话需要记住之前的内容
- 用户偏好和习惯学习
- 长期项目上下文保留

### 配置

```go
memoryMW := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
    Backend: myMemoryBackend,  // 记忆存储后端
    MaxMemories: 50,           // 最大记忆数量
    SimilarityThreshold: 0.7,  // 相似度阈值（0-1）
})
```

### 使用示例

```go
// 第一次会话
ag.Chat(ctx, "我的名字是 Alice，我喜欢蓝色")
// → 记忆保存: {user_name: "Alice", favorite_color: "blue"}

// 关闭 Agent
ag.Close()

// 第二次会话（新 Agent 实例）
ag2, _ := agent.Create(ctx, config, deps)  // 使用相同配置
ag2.Chat(ctx, "我喜欢什么颜色？")
// → 从记忆中检索: favorite_color = "blue"
// → 响应: "您喜欢蓝色"
```

---

## <a id="patch"></a>🔧 PatchToolCalls - 工具调用补丁

**功能**: 修复或增强工具调用的兼容性问题。

**使用场景**:
- LLM 生成的工具调用参数格式不正确
- 工具接口变更后的兼容性适配
- 自动修正常见错误

### 配置

```go
patchMW := middleware.NewPatchToolCallsMiddleware(&middleware.PatchToolCallsMiddlewareConfig{
    Patches: map[string]middleware.PatchFunc{
        "http_request": func(input map[string]interface{}) (map[string]interface{}, error) {
            // 修正 URL 格式
            if url, ok := input["url"].(string); ok {
                if !strings.HasPrefix(url, "http") {
                    input["url"] = "https://" + url
                }
            }
            return input, nil
        },
    },
})
```

---

## 🎯 中间件组合最佳实践

### 完整功能 Agent

```go
stack := middleware.NewStack()

// 1. 人工审批（优先级最高，最先拦截）
stack.Use(hitlMW)  // Priority: 50

// 2. 自动总结（管理上下文）
stack.Use(summaryMW)  // Priority: 40

// 3. 文件系统（注入工具）
stack.Use(filesMW)  // Priority: 100

// 4. 任务列表（任务管理）
stack.Use(todoMW)  // Priority: 120

// 5. 记忆管理（持久化记忆）
stack.Use(memoryMW)  // Priority: 150

// 6. 子 Agent（任务委托）
stack.Use(subagentMW)  // Priority: 200

// 7. 工具补丁（最后执行）
stack.Use(patchMW)  // Priority: 300

// 创建 Agent
ag, _ := agent.Create(ctx, config, &agent.Dependencies{
    MiddlewareStack: stack,
})
```

### 轻量级 Agent

```go
stack := middleware.NewStack()

// 只使用必需的中间件
stack.Use(summaryMW)  // 防止上下文溢出
stack.Use(filesMW)    // 文件操作能力

ag, _ := agent.Create(ctx, config, &agent.Dependencies{
    MiddlewareStack: stack,
})
```

## 📊 监控中间件状态

```go
// 获取中间件统计
for _, mw := range stack.List() {
    fmt.Printf("中间件: %s (优先级: %d)\n", mw.Name(), mw.Priority())

    // 获取 Summarization 中间件统计
    if summaryMW, ok := mw.(*middleware.SummarizationMiddleware); ok {
        stats := summaryMW.Stats()
        fmt.Printf("  总结次数: %d\n", stats["summarization_count"])
    }
}
```

## ❓ 常见问题

### Q: 如何禁用某个中间件？

**方法 1**: 不添加到 Stack

```go
stack := middleware.NewStack()
// 不添加 summaryMW
stack.Use(filesMW)
stack.Use(subagentMW)
```

**方法 2**: 从 Stack 移除

```go
stack.Remove("summarization")
```

### Q: 中间件的执行顺序如何控制？

通过 `Priority()` 返回值控制，数值越小越先执行：

```go
func (m *MyMiddleware) Priority() int {
    return 50  // 在 Filesystem(100) 之前执行
}
```

### Q: 中间件会影响性能吗？

是的，每个中间件都会增加一定开销。建议：
- 只启用必需的中间件
- 避免在中间件中执行耗时操作
- 使用缓存优化重复操作

### Q: 可以动态添加/移除中间件吗？

不建议。中间件应在 Agent 创建时确定。如需动态调整，创建新的 Agent 实例。

## 🔗 相关资源

- [中间件系统概览](/examples/middleware) - 架构和设计
- [自定义中间件](/examples/middleware/custom) - 创建自己的中间件
- [API 参考](/api-reference/middleware) - 接口详细文档
