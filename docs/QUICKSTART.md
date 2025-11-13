# 🚀 AgentSDK 快速开始

> 5 分钟快速上手 - 从零开始构建你的第一个 AI Agent

## 📋 前置要求

- Go 1.21+ 
- API Key (Anthropic/OpenAI/DeepSeek/GLM 任选其一)
- Docker (可选，用于数据库持久化)

## 🎯 安装

```bash
go get github.com/wordflowlab/agentsdk
```

## 🌟 Hello World - 你的第一个 Agent

创建 `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/wordflowlab/agentsdk/pkg/agent"
    "github.com/wordflowlab/agentsdk/pkg/provider"
    "github.com/wordflowlab/agentsdk/pkg/sandbox"
    "github.com/wordflowlab/agentsdk/pkg/store"
    "github.com/wordflowlab/agentsdk/pkg/tools"
    "github.com/wordflowlab/agentsdk/pkg/tools/builtin"
    "github.com/wordflowlab/agentsdk/pkg/types"
)

func main() {
    // 1. 创建工具注册表
    toolRegistry := tools.NewRegistry()
    builtin.RegisterAll(toolRegistry)

    // 2. 创建依赖
    jsonStore, _ := store.NewJSONStore("./.agentsdk")
    deps := &agent.Dependencies{
        Store:            jsonStore,
        SandboxFactory:   sandbox.NewFactory(),
        ToolRegistry:     toolRegistry,
        ProviderFactory:  &provider.AnthropicFactory{},
        TemplateRegistry: agent.NewTemplateRegistry(),
    }

    // 3. 注册 Agent 模板
    deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
        ID:           "assistant",
        SystemPrompt: "你是一个有用的助手，能够访问文件系统和执行 bash 命令。",
        Model:        "claude-sonnet-4-5",
        Tools:        []interface{}{"fs_read", "fs_write", "bash_run"},
    })

    // 4. 创建 Agent
    ag, err := agent.Create(context.Background(), &types.AgentConfig{
        TemplateID: "assistant",
        ModelConfig: &types.ModelConfig{
            Provider: "anthropic",
            Model:    "claude-sonnet-4-5",
            APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
        },
        Sandbox: &types.SandboxConfig{
            Kind:    types.SandboxKindLocal,
            WorkDir: "./workspace",
        },
    }, deps)
    if err != nil {
        log.Fatal(err)
    }
    defer ag.Close()

    // 5. 与 Agent 对话
    result, err := ag.Chat(context.Background(), "创建一个 hello.txt 文件，内容是 'Hello, AgentSDK!'")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Agent 回复: %s\n", result.Text)
}
```

运行:

```bash
export ANTHROPIC_API_KEY=your_api_key
go run main.go
```

**输出**:
```
Agent 回复: 我已经创建了 hello.txt 文件，内容为 'Hello, AgentSDK!'
```

## 🔄 流式响应 - 实时获取输出

```go
// 流式处理 Agent 响应
for event, err := range ag.Stream(ctx, "分析当前目录的文件结构") {
    if err != nil {
        log.Printf("错误: %v", err)
        break
    }

    // 实时打印
    if event.Content.Role == types.RoleAssistant {
        fmt.Print(event.Content.Content)
    }
}
```

**特点**:
- ✅ 内存占用 O(1) vs 传统 O(n)
- ✅ 实时响应，无需等待
- ✅ 支持取消和背压控制

## 🔧 工具系统 - 扩展 Agent 能力

### 使用内置工具

```go
// 注册所有内置工具
builtin.RegisterAll(toolRegistry)

// 或选择性注册
builtin.RegisterFilesystem(toolRegistry)
builtin.RegisterBash(toolRegistry)
builtin.RegisterNetwork(toolRegistry)
```

### 自定义工具

```go
// 1. 定义工具结构
type WeatherTool struct {
    tools.BaseTool
}

func NewWeatherTool() *WeatherTool {
    return &WeatherTool{
        BaseTool: tools.BaseTool{
            ToolName:        "get_weather",
            ToolDescription: "获取指定城市的天气信息",
            ToolInputSchema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "city": map[string]interface{}{
                        "type":        "string",
                        "description": "城市名称",
                    },
                },
                "required": []string{"city"},
            },
        },
    }
}

// 2. 实现 Execute 方法
func (t *WeatherTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    city := args["city"].(string)
    
    // 调用天气 API
    weather := fmt.Sprintf("%s: 晴天，25°C", city)
    
    return weather, nil
}

// 3. 注册工具
toolRegistry.Register(NewWeatherTool())

// 4. 在 Agent 中使用
deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
    ID:    "weather-assistant",
    Tools: []interface{}{"get_weather"},
})
```

## 🌊 工作流 Agent - 编排复杂任务

### 顺序工作流

```go
import "github.com/wordflowlab/agentsdk/pkg/agent/workflow"

// 创建子 Agent
collector := NewDataCollectorAgent()
analyzer := NewAnalyzerAgent()
reporter := NewReporterAgent()

// 组合成顺序工作流
sequential, _ := workflow.NewSequentialAgent(workflow.SequentialConfig{
    Name: "DataPipeline",
    SubAgents: []workflow.Agent{
        collector,  // 步骤1: 收集数据
        analyzer,   // 步骤2: 分析数据
        reporter,   // 步骤3: 生成报告
    },
})

// 执行工作流
for event, err := range sequential.Execute(ctx, "处理用户数据") {
    fmt.Printf("步骤 %s: %s\n", event.AgentID, event.Content.Content)
}
```

### 并行工作流

```go
// 并行执行多个方案
parallel, _ := workflow.NewParallelAgent(workflow.ParallelConfig{
    Name: "MultiSolver",
    SubAgents: []workflow.Agent{
        algorithmA,  // 方案A
        algorithmB,  // 方案B
        algorithmC,  // 方案C
    },
})

// 并发执行，收集所有结果
for event, err := range parallel.Execute(ctx, "求解问题") {
    fmt.Printf("方案 %s 结果: %s\n", event.AgentID, event.Content.Content)
}
```

### 循环工作流

```go
// 循环优化直到满足条件
loop, _ := workflow.NewLoopAgent(workflow.LoopConfig{
    Name:          "CodeOptimizer",
    SubAgents:     []workflow.Agent{critic, improver},
    MaxIterations: 5,
    StopCondition: func(event *session.Event) bool {
        // 质量达标后停止
        score := event.Metadata["quality_score"].(int)
        return score >= 90
    },
})

for event, err := range loop.Execute(ctx, "优化代码") {
    fmt.Printf("迭代 %d: %s\n", 
        event.Metadata["loop_iteration"], 
        event.Content.Content)
}
```

## 💾 数据持久化 - PostgreSQL/MySQL

### PostgreSQL

```go
import "github.com/wordflowlab/agentsdk/pkg/session/postgres"

// 创建 PostgreSQL Session 服务
sessionService, _ := postgres.NewService(&postgres.Config{
    DSN: "host=localhost port=5432 user=postgres dbname=agentsdk",
    AutoMigrate: true,
})
defer sessionService.Close()

// 创建 Session
sess, _ := sessionService.Create(ctx, &session.CreateRequest{
    AppName: "my-app",
    UserID:  "user-001",
    AgentID: "agent-001",
})

// 追加事件
event := &session.Event{
    ID:       "evt-001",
    AgentID:  "agent-001",
    Content:  types.Message{Role: types.RoleUser, Content: "Hello"},
}
sessionService.AppendEvent(ctx, sess.ID, event)

// 查询事件
events, _ := sessionService.GetEvents(ctx, sess.ID, nil)
```

### MySQL 8.0+

```go
import "github.com/wordflowlab/agentsdk/pkg/session/mysql"

mysqlService, _ := mysql.NewService(&mysql.Config{
    DSN: "root:password@tcp(127.0.0.1:3306)/agentsdk?charset=utf8mb4",
    AutoMigrate: true,
})
// 使用方式与 PostgreSQL 相同
```

## 📊 可观测性 - OpenTelemetry

```go
import "github.com/wordflowlab/agentsdk/pkg/telemetry"

// 1. 创建 Tracer
tracer, _ := telemetry.NewOTelTracer("agentsdk",
    telemetry.WithJaegerExporter("localhost:14268"),
)
defer tracer.Shutdown(context.Background())

// 2. 创建 Span
ctx, span := tracer.StartSpan(context.Background(), "agent.execute")
defer tracer.EndSpan(ctx)

// 3. 添加属性
tracer.AddEvent(ctx, "tool.execute", map[string]interface{}{
    "tool": "fs_read",
    "args": "/path/to/file",
})

// 4. Agent 执行（自动追踪）
result, _ := ag.Chat(ctx, "读取文件")
```

在 Jaeger UI 中查看追踪:
```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest

# 访问 http://localhost:16686
```

## 🎨 Middleware - 扩展执行流程

```go
// 1. 自定义 Middleware
type LoggingMiddleware struct{}

func (m *LoggingMiddleware) Name() string {
    return "logging"
}

func (m *LoggingMiddleware) Priority() int {
    return 100
}

func (m *LoggingMiddleware) OnMessageProcess(ctx middleware.Context, next middleware.NextFunc) error {
    log.Printf("处理消息: %v", ctx.Messages())
    return next(ctx)
}

// 2. 注册 Middleware
agent.RegisterMiddleware(&LoggingMiddleware{})

// 3. 在 Agent 中启用
ag, _ := agent.Create(ctx, &types.AgentConfig{
    TemplateID: "assistant",
    Middlewares: []string{"logging", "summarization"},
}, deps)
```

## 🔗 多 Agent 协作

```go
// 1. 创建 Agent Pool
pool := agent.NewPool()

// 2. 注册多个 Agent
pool.Register("researcher", researcherAgent)
pool.Register("writer", writerAgent)
pool.Register("reviewer", reviewerAgent)

// 3. 创建协作 Room
room := agent.NewRoom(pool, &agent.RoomConfig{
    Agents: []string{"researcher", "writer", "reviewer"},
})

// 4. 多 Agent 协作执行任务
result, _ := room.Execute(ctx, "写一篇关于 AI 的文章")
```

## 📚 更多示例

| 示例 | 说明 | 路径 |
|------|------|------|
| 基础 Agent | 最简单的 Agent 使用 | `examples/agent` |
| 流式处理 | iter.Seq2 流式接口 | `examples/streaming` |
| 工作流 Agent | Sequential/Parallel/Loop | `examples/workflow-agents` |
| 长时运行工具 | 异步任务管理 | `examples/long-running-tools` |
| PostgreSQL | Session 持久化 | `examples/session-postgres` |
| MySQL | Session 持久化 | `examples/session-mysql` |
| OpenTelemetry | 分布式追踪 | `examples/telemetry` |
| MCP 集成 | MCP 工具扩展 | `examples/mcp` |

## 🛠️ 故障排查

### Agent 创建失败

**问题**: `Failed to create agent: template not found`

**解决**: 确保先注册 Agent 模板
```go
deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
    ID: "assistant",
    // ...
})
```

### 工具执行失败

**问题**: `Tool 'xxx' not found`

**解决**: 确保工具已注册
```go
toolRegistry.Register(NewYourTool())
```

### API 调用超时

**问题**: `context deadline exceeded`

**解决**: 增加超时时间
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
```

## 🎓 下一步

- 📖 阅读 [完整文档](https://wordflowlab.github.io/agentsdk/)
- 🏗️ 了解 [架构设计](../ARCHITECTURE.md)
- 🔧 查看 [API 文档](../docs/API.md)
- 💡 参考 [最佳实践](../docs/BEST_PRACTICES.md)
- 🐛 [报告问题](https://github.com/wordflowlab/agentsdk/issues)

## ❓ 常见问题

### Q: AgentSDK 与其他框架的区别？

**A**: AgentSDK 专注于企业级生产环境:
- ✅ 事件驱动架构（Progress/Control/Monitor 三通道）
- ✅ 云端沙箱集成（阿里云、火山引擎）
- ✅ 完整的可观测性（OpenTelemetry）
- ✅ 数据持久化（PostgreSQL/MySQL）
- ✅ 工作流编排（Parallel/Sequential/Loop）

### Q: 支持哪些大模型？

**A**: 当前支持:
- Anthropic (Claude)
- OpenAI (GPT-4)
- DeepSeek
- GLM (智谱)

### Q: 如何贡献代码？

**A**: 欢迎贡献！请查看 [CONTRIBUTING.md](../CONTRIBUTING.md)

---

**开始构建你的 AI Agent 吧！** 🚀
