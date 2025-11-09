# AgentSDK 快速开始

## 🚀 5分钟快速上手

### 安装

```bash
go get github.com/wordflowlab/agentsdk
```

### 基础示例

```go
package main

import (
    "context"
    "fmt"

    "github.com/wordflowlab/agentsdk/pkg/backends"
    "github.com/wordflowlab/agentsdk/pkg/middleware"
)

func main() {
    ctx := context.Background()

    // 1. 创建 Backend
    backend := backends.NewStateBackend()

    // 2. 创建 FilesystemMiddleware
    fsMiddleware := middleware.NewFilesystemMiddleware(&middleware.FilesystemMiddlewareConfig{
        Backend:        backend,
        EnableEviction: true,
    })

    // 3. 创建 Middleware Stack
    stack := middleware.NewStack([]middleware.Middleware{
        fsMiddleware,
    })

    // 4. 获取所有工具
    tools := stack.Tools()
    fmt.Printf("可用工具: %d 个\n", len(tools))
    for _, tool := range tools {
        fmt.Printf("- %s: %s\n", tool.Name(), tool.Description())
    }

    // 5. 使用 Backend
    backend.Write(ctx, "/hello.txt", "Hello AgentSDK!")
    content, _ := backend.Read(ctx, "/hello.txt", 0, 0)
    fmt.Printf("\n文件内容: %s\n", content)
}
```

### 运行示例

```bash
cd examples/subagent
go run main.go
```

## 📚 核心概念

### Backend - 存储抽象层

4种可选的存储后端:

```go
// 1. 内存临时存储
state := backends.NewStateBackend()

// 2. 持久化存储
store := backends.NewStoreBackend(storeImpl, "agent-id")

// 3. 真实文件系统
fs := backends.NewFilesystemBackend(sandboxFS)

// 4. 路由组合(混合策略)
composite := backends.NewCompositeBackend(
    state, // 默认
    []backends.RouteConfig{
        {Prefix: "/memories/", Backend: store},
        {Prefix: "/workspace/", Backend: fs},
    },
)
```

### Middleware - 可组合功能

洋葱模型的中间件架构:

```go
// 文件系统中间件 (6个工具)
fsMiddleware := middleware.NewFilesystemMiddleware(&middleware.FilesystemMiddlewareConfig{
    Backend:        backend,
    EnableEviction: true,  // 自动驱逐大结果
    TokenLimit:     20000, // 20k tokens
})

// 子代理中间件 (task工具)
subagentMiddleware, _ := middleware.NewSubAgentMiddleware(&middleware.SubAgentMiddlewareConfig{
    Specs: []middleware.SubAgentSpec{
        {Name: "researcher", Description: "Research expert"},
        {Name: "coder", Description: "Coding expert"},
    },
    Factory: mySubAgentFactory,
})

// 创建栈(自动按优先级排序)
stack := middleware.NewStack([]middleware.Middleware{
    fsMiddleware,      // priority: 100
    subagentMiddleware, // priority: 200
})
```

## 🛠️ 可用工具

### 文件系统工具 (FilesystemMiddleware)

| 工具 | 功能 | 示例 |
|-----|------|------|
| `fs_read` | 读取文件 | 支持分页: `offset`, `limit` |
| `fs_write` | 写入文件 | 覆盖写入 |
| `fs_ls` | 列出目录 | 显示大小、时间 |
| `fs_edit` | 精确编辑 | 字符串替换 |
| `fs_glob` | Glob匹配 | `**/*.go` |
| `fs_grep` | 正则搜索 | 显示行号 |

### 子代理工具 (SubAgentMiddleware)

| 工具 | 功能 | 示例 |
|-----|------|------|
| `task` | 任务委托 | 启动子代理执行隔离任务 |

## 📖 进阶使用

### 自定义 Backend

```go
type MyBackend struct {
    // 你的实现
}

func (b *MyBackend) Read(ctx, path, offset, limit) (string, error) {
    // 实现读取逻辑
}

// 实现其他 BackendProtocol 方法...
```

### 自定义 Middleware

```go
type MyMiddleware struct {
    *middleware.BaseMiddleware
}

func NewMyMiddleware() *MyMiddleware {
    return &MyMiddleware{
        BaseMiddleware: middleware.NewBaseMiddleware("my-middleware", 300),
    }
}

func (m *MyMiddleware) Tools() []tools.Tool {
    return []tools.Tool{&MyTool{}}
}

func (m *MyMiddleware) WrapToolCall(ctx, req, handler) (*ToolCallResponse, error) {
    // 前置处理
    log.Printf("Before: %s", req.ToolName)

    // 调用下一层
    resp, err := handler(ctx, req)

    // 后置处理
    log.Printf("After: %v", resp.Result)

    return resp, err
}
```

### 子代理配置

```go
specs := []middleware.SubAgentSpec{
    {
        Name:        "researcher",
        Description: "Deep research and analysis expert",
        Prompt:      "You are a research specialist. Provide detailed analysis.",
    },
    {
        Name:        "coder",
        Description: "Code writing expert",
        Prompt:      "You are a professional programmer. Write clean code.",
    },
}

factory := func(ctx context.Context, spec middleware.SubAgentSpec) (middleware.SubAgent, error) {
    // 创建你的 Agent 实例
    // 或使用 SimpleSubAgent 快速原型
    return middleware.NewSimpleSubAgent(spec.Name, spec.Prompt, myExecFunc), nil
}

subagentMiddleware, _ := middleware.NewSubAgentMiddleware(&middleware.SubAgentMiddlewareConfig{
    Specs:          specs,
    Factory:        factory,
    EnableParallel: true,
})
```

## 🎯 最佳实践

### 1. Backend 选择

- **临时数据**: `StateBackend` (内存快速)
- **持久数据**: `StoreBackend` (跨会话)
- **工作文件**: `FilesystemBackend` (真实FS)
- **混合场景**: `CompositeBackend` (路由策略)

### 2. Middleware 优先级

```go
const (
    PrioritySystem   = 0   // 0-100: 系统核心
    PriorityFeature  = 100 // 100-500: 通用功能
    PriorityBusiness = 500 // 500-1000: 业务逻辑
)
```

### 3. 错误处理

```go
// 工具应返回结构化错误信息(不要返回 error)
return map[string]interface{}{
    "ok":    false,
    "error": "详细错误信息",
    "recommendations": []string{
        "建议1",
        "建议2",
    },
}, nil
```

## 📊 性能指标

基于 Apple M1:

```
BenchmarkMiddlewareStack-8    31301286    36.21 ns/op    96 B/op    1 allocs/op
BenchmarkBackendWrite-8        4662870   257.9 ns/op   480 B/op    5 allocs/op
```

- **Middleware Stack**: 36.21 ns/op (每秒 ~2760万次)
- **Backend Write**: 257.9 ns/op (每秒 ~387万次)

## 📚 更多资源

- 📖 [完整架构文档](ARCHITECTURE.md)
- 📝 [实施计划详情](IMPLEMENTATION_PLAN.md)
- 💻 [完整示例代码](examples/subagent/main.go)
- 🐍 [DeepAgents (Python参考)](https://github.com/anthropics/deepagents)

## 🎉 核心优势

✅ **灵活的存储策略** - 4种可组合Backend
✅ **强大的扩展性** - Middleware插件化
✅ **丰富的工具集** - 7个内置工具
✅ **极致的性能** - Go语言优势
✅ **低内存占用** - 96-480 B/op
✅ **真正的并发** - Goroutine支持

## 🤝 参与贡献

欢迎提交 Issue 和 Pull Request!

## 📄 License

MIT License
