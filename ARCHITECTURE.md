# AgentSDK 架构指南

## 架构概览

AgentSDK 采用三层可扩展架构,灵感来自 [DeepAgents](https://github.com/anthropics/deepagents) 项目:

```
┌─────────────────────────────────────────────────────────┐
│                      Agent Layer                        │
│  (通过 AgentConfig.Middlewares 配置启用)                │
│  ┌──────────────────────────────────────────────────┐  │
│  │             Middleware Stack                     │  │
│  │  ┌────────────────────────────────────────────┐  │  │
│  │  │  SummarizationMiddleware (priority: 40)   │  │  │
│  │  │  - Auto summarize (>170k tokens)         │  │  │
│  │  │  - Keep recent 6 messages                 │  │  │
│  │  └────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────┐  │  │
│  │  │  FilesystemMiddleware (priority: 100)     │  │  │
│  │  │  - Tools: [fs_read, fs_write, fs_ls,      │  │  │
│  │  │           fs_edit, fs_glob, fs_grep]      │  │  │
│  │  │  - Auto eviction (>20k tokens)            │  │  │
│  │  └────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────┐  │  │
│  │  │  SubAgentMiddleware (priority: 200)       │  │  │
│  │  │  - Tools: [task]                          │  │  │
│  │  │  - Context isolation                      │  │  │
│  │  └────────────────────────────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────┐  │  │
│  │  │  CustomMiddleware (priority: 500+)        │  │  │
│  │  └────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────┘  │
│         │ WrapModelCall      │ WrapToolCall            │
│         ▼                    ▼                         │
│  ┌─────────────┐      ┌──────────────┐                │
│  │  Provider   │      │ Tool Executor│                │
│  │  (Stream)   │      │  (Execute)   │                │
│  └─────────────┘      └──────────────┘                │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│                    Backend Layer                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │ StateBackend │  │ StoreBackend │  │FilesystemBE  │  │
│  │  (临时内存)  │  │  (持久化)    │  │  (真实FS)    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│                ┌──────────────────┐                     │
│                │ CompositeBackend │                     │
│                │   (路由组合)     │                     │
│                └──────────────────┘                     │
└─────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. Backend 抽象层 (`pkg/backends/`)

统一的存储接口,支持多种后端实现:

#### BackendProtocol 接口

```go
type BackendProtocol interface {
    ListInfo(ctx, path) ([]FileInfo, error)
    Read(ctx, path, offset, limit) (string, error)
    Write(ctx, path, content) (*WriteResult, error)
    Edit(ctx, path, old, new, replaceAll) (*EditResult, error)
    GrepRaw(ctx, pattern, path, glob) ([]GrepMatch, error)
    GlobInfo(ctx, pattern, path) ([]FileInfo, error)
}
```

#### 四种 Backend 实现

| Backend | 用途 | 生命周期 | 使用场景 |
|---------|------|---------|---------|
| **StateBackend** | 内存临时存储 | 会话级 | 临时文件、中间结果 |
| **StoreBackend** | 持久化存储 | 跨会话 | 知识库、记忆 |
| **FilesystemBackend** | 真实文件系统 | 永久 | 工作空间文件 |
| **CompositeBackend** | 路由组合 | - | 混合存储策略 |

#### 使用示例

```go
// 1. 单一后端
backend := backends.NewStateBackend()

// 2. 组合后端(路由)
composite := backends.NewCompositeBackend(
    backends.NewStateBackend(), // 默认
    []backends.RouteConfig{
        {Prefix: "/memories/", Backend: storeBackend},
        {Prefix: "/workspace/", Backend: fsBackend},
    },
)
```

### 2. Middleware 系统 (`pkg/middleware/`)

洋葱模型的可组合中间件架构:

#### Middleware 接口

```go
type Middleware interface {
    Name() string
    Priority() int  // 0-100: 系统, 100-500: 功能, 500+: 用户
    Tools() []tools.Tool

    // 双向拦截
    WrapModelCall(ctx, req, handler) (*ModelResponse, error)
    WrapToolCall(ctx, req, handler) (*ToolCallResponse, error)

    // 生命周期钩子
    OnAgentStart(ctx, agentID) error
    OnAgentStop(ctx, agentID) error
}
```

#### 内置 Middleware

##### SummarizationMiddleware (Phase 6C)
- **优先级**: 40
- **工具**: 无 (纯处理型)
- **功能**:
  - 自动监控消息历史 token 数
  - 超过阈值时自动总结旧消息 (默认 170k tokens)
  - 保留最近 N 条消息 (默认 6 条)
  - 用总结消息替换旧历史
- **配置示例**:
  ```go
  config := &types.AgentConfig{
      Middlewares: []string{"summarization"},
      // ...
  }
  ```

##### FilesystemMiddleware
- **优先级**: 100
- **工具**: fs_read, fs_write, fs_ls, fs_edit, fs_glob, fs_grep
- **功能**:
  - 自动大结果驱逐 (>20k tokens)
  - 系统提示词增强

##### SubAgentMiddleware
- **优先级**: 200
- **工具**: task
- **功能**:
  - 任务委托到子代理
  - 上下文隔离
  - 并发执行支持

#### 使用示例

```go
// 创建中间件
fsMiddleware := middleware.NewFilesystemMiddleware(&middleware.FilesystemMiddlewareConfig{
    Backend: backend,
    EnableEviction: true,
    TokenLimit: 20000,
})

subagentMiddleware, _ := middleware.NewSubAgentMiddleware(&middleware.SubAgentMiddlewareConfig{
    Specs: []middleware.SubAgentSpec{
        {Name: "researcher", Description: "Research expert"},
    },
    Factory: subagentFactory,
})

// 创建栈(自动按优先级排序)
stack := middleware.NewStack([]middleware.Middleware{
    fsMiddleware,
    subagentMiddleware,
})

// 获取所有工具
tools := stack.Tools()  // 7 个工具
```

## 工具清单

### 文件系统工具 (FilesystemMiddleware)

| 工具 | 描述 | 主要用途 |
|-----|------|---------|
| **fs_read** | 读取文件内容 | 支持分页读取 |
| **fs_write** | 写入文件 | 覆盖写入 |
| **fs_ls** | 列出目录 | 显示大小、时间 |
| **fs_edit** | 精确编辑 | 字符串替换 |
| **fs_glob** | Glob 匹配 | `**/*.go` |
| **fs_grep** | 正则搜索 | 显示行号、上下文 |

### 网络工具 (Phase 6B-1)

| 工具 | 描述 | 主要用途 |
|-----|------|---------|
| **http_request** | HTTP/HTTPS 请求 | GET/POST/PUT/DELETE/PATCH/HEAD |
| **web_search** | Web 搜索 | Tavily API (general/news/finance) |

### 子代理工具 (SubAgentMiddleware)

| 工具 | 描述 | 主要用途 |
|-----|------|---------|
| **task** | 任务委托 | 启动子代理执行隔离任务 |

## 性能指标

基于 Apple M1, Go 1.21:

| 操作 | 性能 | 内存 | 吞吐量 |
|------|------|------|--------|
| Middleware Stack | 36.21 ns/op | 96 B/op | ~27M ops/s |
| Backend Write | 257.9 ns/op | 480 B/op | ~3.8M ops/s |

## 设计模式

### 1. Protocol + Factory 模式

```go
// 定义协议
type BackendProtocol interface { ... }

// 工厂函数
type BackendFactory func(ctx) (BackendProtocol, error)

// 延迟初始化
backend := factory(ctx)
```

### 2. Middleware 洋葱模型

```
Request → M1 → M2 → M3 → Handler → M3 → M2 → M1 → Response
          ↓    ↓    ↓              ↑    ↑    ↑
      Before Before Before       After After After
```

### 3. 优先级排序

```go
type Middleware interface {
    Priority() int  // 数值越小优先级越高
}

// 自动排序
stack := NewStack(middlewares)  // 按 Priority 升序
```

### 4. 工具生成器

```go
func createTool(backend BackendProtocol) Tool {
    return &CustomTool{backend: backend}
}
```

## 扩展指南

### 创建自定义 Backend

```go
type MyBackend struct {
    // 你的存储逻辑
}

func (b *MyBackend) Read(ctx, path, offset, limit) (string, error) {
    // 实现读取
}

// ... 实现其他方法
```

### 创建自定义 Middleware

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
    return []tools.Tool{
        &MyTool{},
    }
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

## 最佳实践

### 1. Backend 选择

- **临时数据**: StateBackend
- **持久数据**: StoreBackend
- **工作文件**: FilesystemBackend
- **混合场景**: CompositeBackend

### 2. Middleware 优先级

- **0-100**: 系统核心功能
- **100-500**: 通用功能(文件、子代理等)
- **500-1000**: 业务逻辑

### 3. 工具命名

- 使用动词开头: `fs_read`, `fs_write`
- 分组前缀: `fs_*`, `git_*`, `api_*`

### 4. 错误处理

```go
// 工具返回结构化错误
return map[string]interface{}{
    "ok":    false,
    "error": "详细错误信息",
    "recommendations": []string{
        "建议1",
        "建议2",
    },
}, nil  // 不要返回 error,让 LLM 看到错误信息
```

## 对比 DeepAgents

| 特性 | DeepAgents (Python) | AgentSDK (Go) |
|------|-------------------|-------------------|
| Backend Protocol | ✅ 4种 | ✅ 4种 |
| Middleware 栈 | ✅ 洋葱模型 | ✅ 洋葱模型 |
| 文件工具 | ✅ 6个 | ✅ 6个 |
| 自动驱逐 | ✅ | ✅ |
| 子代理 | ✅ | ✅ |
| 性能 | 中等 | **极高** (Go) |
| 内存 | 高 | **低** (Go) |
| 并发 | GIL限制 | **真正并发** (Goroutine) |

## 参考资料

- [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) - 完整实施文档
- [examples/subagent/](examples/subagent/) - 完整示例
- [DeepAgents](https://github.com/anthropics/deepagents) - Python 参考实现
