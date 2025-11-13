---
title: 带长期记忆的 Agent
description: 使用 filesystem + agent_memory 中间件构建纯文件式长期记忆
---

# 带长期记忆的 Agent

本示例演示如何同时启用 **FilesystemMiddleware** 和 **AgentMemoryMiddleware**，构建一个基于“普通文件+grep 搜索”的长期记忆系统，而不是向量/RAG。

- 记忆文件全部存放在 `./memories` 目录，对 Agent 暴露为 `/memories/` 路径。
- Agent 可以使用：
  - `memory_write` 向 Markdown 记忆文件追加/覆盖 Note。
  - `memory_search` 在记忆目录中进行全文搜索。
  - `fs_read` / `fs_write` / `fs_grep` 等工具查看和操作记忆文件。
- `/agent.md` 的内容会被自动注入到 System Prompt，作为基础“人格/长期指令”。

> 示例代码路径：`examples/memory-agent/main.go`

## 场景说明

我们希望 Agent 能够：

- 记住用户的偏好和约定（比如“用户喜欢 grep 风格的搜索”）。
- 在后续对话中先查长期记忆，再回答问题。
- 所有记忆都用普通 Markdown 文件保存，方便人工编辑和版本管理。

## 核心代码

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/wordflowlab/agentsdk/pkg/agent"
    "github.com/wordflowlab/agentsdk/pkg/backends"
    "github.com/wordflowlab/agentsdk/pkg/middleware"
    "github.com/wordflowlab/agentsdk/pkg/provider"
    "github.com/wordflowlab/agentsdk/pkg/sandbox"
    "github.com/wordflowlab/agentsdk/pkg/store"
    "github.com/wordflowlab/agentsdk/pkg/tools"
    "github.com/wordflowlab/agentsdk/pkg/tools/builtin"
    "github.com/wordflowlab/agentsdk/pkg/types"
)

func main() {
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    if apiKey == "" {
        log.Fatal("ANTHROPIC_API_KEY environment variable is required")
    }

    ctx := context.Background()

    // 1. 工具注册表
    toolRegistry := tools.NewRegistry()
    builtin.RegisterAll(toolRegistry)

    // 2. Sandbox / Provider / Store / 模板
    sandboxFactory := sandbox.NewFactory()
    providerFactory := &provider.AnthropicFactory{}
    jsonStore, _ := store.NewJSONStore(".agentsdk-memory-agent")

    templateRegistry := agent.NewTemplateRegistry()
    templateRegistry.Register(&types.AgentTemplateDefinition{
        ID: "memory-assistant",
        SystemPrompt: "You are an assistant with file access and long-term memory. " +
            "Always prefer reading from and writing to memory files when users ask to remember or recall information.",
        Model: "claude-sonnet-4-5",
        Tools: []interface{}{"fs_read", "fs_write", "bash_run"},
    })

    deps := &agent.Dependencies{
        Store:            jsonStore,
        SandboxFactory:   sandboxFactory,
        ToolRegistry:     toolRegistry,
        ProviderFactory:  providerFactory,
        TemplateRegistry: templateRegistry,
    }

    // 3. Backend: /workspace/ + /memories/
    workDir := "./workspace"
    os.MkdirAll(workDir, 0o755)
    os.MkdirAll("./memories", 0o755)

    stateBackend := backends.NewStateBackend()
    localWorkspace := backends.NewLocalBackend(workDir)
    localMemBackend := backends.NewLocalBackend("./memories")

    fsBackend := backends.NewCompositeBackend(
        stateBackend,
        []backends.RouteConfig{
            {Prefix: "/workspace/", Backend: localWorkspace},
            {Prefix: "/memories/", Backend: localMemBackend},
        },
    )

    // 4. Filesystem + AgentMemory 中间件
    filesMW := middleware.NewFilesystemMiddleware(&middleware.FilesystemMiddlewareConfig{
        Backend: fsBackend,
    })

    memoryMW, err := middleware.NewAgentMemoryMiddleware(&middleware.AgentMemoryMiddlewareConfig{
        Backend:    fsBackend,
        MemoryPath: "/memories/",
    })
    if err != nil {
        log.Fatalf("create AgentMemoryMiddleware failed: %v", err)
    }

    // 5. AgentConfig: 启用 filesystem + agent_memory
    config := &types.AgentConfig{
        TemplateID: "memory-assistant",
        ModelConfig: &types.ModelConfig{
            Provider: "anthropic",
            Model:    "claude-sonnet-4-5",
            APIKey:   apiKey,
        },
        Sandbox: &types.SandboxConfig{
            Kind:    types.SandboxKindLocal,
            WorkDir: workDir,
        },
        Middlewares: []string{"filesystem", "agent_memory"},
    }

    ag, err := agent.Create(ctx, config, deps)
    if err != nil {
        log.Fatalf("create agent failed: %v", err)
    }
    defer ag.Close()

    fmt.Printf("✅ Memory Agent created: %s\n", ag.ID())

    // 6. 订阅进度事件, 简单打印输出
    eventCh := ag.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)
    go func() {
        for envelope := range eventCh {
            if evt, ok := envelope.Event.(types.EventType); ok {
                switch e := evt.(type) {
                case *types.ProgressTextChunkStartEvent:
                    fmt.Print("\n[Assistant] ")
                case *types.ProgressTextChunkEvent:
                    fmt.Print(e.Delta)
                case *types.ProgressDoneEvent:
                    fmt.Printf("\n[Done] Step %d (%s)\n", e.Step, e.Reason)
                }
            }
        }
    }()

    // 7. 第一次对话: 要求 Agent 将用户偏好写入长期记忆
    prompt1 := `
我叫 Alice, 以后请记住:
- 我喜欢 grep 风格的代码搜索
- 我喜欢简洁的代码 diff
- 遇到复杂问题时, 先搜索已有记忆再回答

请把这些信息保存到你的长期记忆中, 并告诉我你保存到了哪个记忆文件。`

    if _, err := ag.Chat(ctx, prompt1); err != nil {
        log.Fatalf("chat 1 failed: %v", err)
    }

    time.Sleep(2 * time.Second)

    // 8. 第二次对话: 让 Agent 先搜索记忆再回答
    prompt2 := `
还记得我对代码工具的偏好吗? 请先在你的长期记忆中搜索相关记录, 然后用一小段话总结出来回答我。`

    if _, err := ag.Chat(ctx, prompt2); err != nil {
        log.Fatalf("chat 2 failed: %v", err)
    }

    time.Sleep(2 * time.Second)
}
```

## 运行示例

```bash
cd examples

# 设置 Anthropic API Key
export ANTHROPIC_API_KEY=your_api_key_here

# 运行示例
go run memory-agent/main.go
```

运行过程中你会看到：

- Agent 首次对话时，会调用 `memory_write` 把 Alice 的偏好写入某个 `/memories/user/...` 文件。
- 第二次对话时，会先调用 `memory_search` 在记忆文件中查找 “grep 风格”“代码 diff” 等关键词，然后基于记忆回答。
- 在本地 `./memories` 目录里，可以直接打开对应的 Markdown 文件，看到结构化的 Note 内容。

## 与 `examples/memory` 的区别

- `examples/memory/main.go`：
  - 直接以工具形式调用 `memory_write` / `memory_search`，用于验证 Memory 管理器和工具本身。
  - 不涉及真正的模型/对话。

- `examples/memory-agent/main.go`（本示例）：
  - 创建了一个真实 Agent，启用 filesystem + agent_memory 中间件。
  - 由 LLM 自己决定何时调用 `memory_*` 工具，实现“长期记忆”的真实对话体验。

