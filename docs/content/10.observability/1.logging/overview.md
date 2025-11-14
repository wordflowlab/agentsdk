---
title: Logging 示例
description: 使用 pkg/logging 实现可扩展的日志输出
navigation: false
---

# Logging 示例

AgentSDK 提供了 `pkg/logging` 包, 用于统一管理结构化日志输出, 并通过 Transport 抽象支持多目标输出。

目标:
- 使用统一的 `Logger` 接口和 `Transport` 抽象。
- 支持多个输出目标(Stdout、文件等)。
- 与现有 telemetry(指标/追踪)互补, 共同构成完整的可观测性。

示例代码路径:
- `pkg/logging/logging.go`
- `examples/logging/main.go`

## 1. 核心概念

```go
// 日志级别
type Level string

const (
    LevelDebug Level = "debug"
    LevelInfo  Level = "info"
    LevelWarn  Level = "warn"
    LevelError Level = "error"
)

// 标准化日志记录结构
type LogRecord struct {
    Timestamp time.Time              `json:"ts"`
    Level     Level                  `json:"level"`
    Message   string                 `json:"message"`
    Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Transport 日志输出通道接口
type Transport interface {
    Name() string
    Log(ctx context.Context, rec *LogRecord) error
    Flush(ctx context.Context) error
}

// Logger 聚合多个 Transport
type Logger struct {
    // ...
}
```

## 2. 创建 Logger

### StdoutTransport

```go
// 将日志以 JSON 行写到 stdout
stdout := logging.NewStdoutTransport()

logger := logging.NewLogger(logging.LevelInfo, stdout)

logger.Info(ctx, "server.started", map[string]interface{}{
    "addr": ":8080",
    "env":  "dev",
})
```

### FileTransport

```go
fileTransport, err := logging.NewFileTransport("./logs/app.log")
if err != nil {
    panic(err)
}
defer fileTransport.Close()

fileLogger := logging.NewLogger(logging.LevelInfo, fileTransport)

fileLogger.Info(ctx, "agent.chat.started", map[string]interface{}{
    "agent_id":   "agt:demo",
    "user_id":    "alice",
    "templateID": "assistant",
})
```

### 全局 Default Logger

`pkg/logging` 提供了一个可选的全局 Logger, 方便快速集成:

```go
// 默认使用 LevelInfo + StdoutTransport
logging.Info(ctx, "request.completed", map[string]interface{}{
    "status":  "ok",
    "latency": 0.123,
})

logging.Error(ctx, "tool.call.failed", map[string]interface{}{
    "tool_name": "http_request",
    "error":     err.Error(),
})
``>

> 提示: 建议在长期维护的系统中, 显式创建自己的 `Logger` 实例并注入到业务组件, 而不是依赖全局 Default。

## 3. 示例: `examples/logging/main.go`

```go
func main() {
    ctx := context.Background()

    // 1. stdout logger
    stdLogger := logging.NewLogger(logging.LevelDebug, logging.NewStdoutTransport())

    stdLogger.Info(ctx, "server.started", map[string]interface{}{
        "addr": ":8080",
        "env":  "dev",
    })

    // 2. file logger
    fileTransport, err := logging.NewFileTransport("./logs/app.log")
    if err != nil {
        panic(fmt.Sprintf("failed to create file transport: %v", err))
    }
    defer fileTransport.Close()

    fileLogger := logging.NewLogger(logging.LevelInfo, fileTransport)

    fileLogger.Info(ctx, "agent.chat.started", map[string]interface{}{
        "agent_id":   "agt:demo",
        "user_id":    "alice",
        "templateID": "assistant",
    })

    // 模拟一次工具调用
    start := time.Now()
    time.Sleep(150 * time.Millisecond)
    duration := time.Since(start)

    fileLogger.Info(ctx, "tool.call.completed", map[string]interface{}{
        "agent_id":  "agt:demo",
        "tool_name": "fs_read",
        "duration":  duration.Seconds(),
        "success":   true,
    })

    // 3. 使用全局 Default logger
    logging.Info(ctx, "request.completed", map[string]interface{}{
        "status":  "ok",
        "latency": 0.123,
    })

    logging.Flush(ctx)
    fileLogger.Flush(ctx)
    stdLogger.Flush(ctx)
}
```

运行:

```bash
cd examples
go run logging/main.go
```

- stdout 中会打印 JSON 行日志。
- `./logs/app.log` 会包含文件日志。

## 4. 与 telemetry 的关系

`pkg/logging` 专注于**事件日志**(谁在什么时候做了什么), 而 `pkg/telemetry` 专注于:

- 指标(Metrics): `Metrics` / `AgentMetrics` – 统计请求次数、延迟、token 使用等。
- 追踪(Tracing): `Tracer` / `Span` – 跟踪某个 Chat 请求在 Agent/工具/中间件中的调用链路。

它们之间的推荐配合方式:

- 使用 `telemetry.Metrics` 对性能和错误率做聚合统计。
- 使用 `telemetry.Tracer` 追踪关键请求链路。
- 使用 `logging.Logger` 记录具体事件细节,例如:
  - 某次工具调用的输入/输出摘要。
  - 某个错误的上下文信息(AgentID/SessionID/UserID 等)。
  - 重要状态变更(如 Agent 恢复/中断)。

未来你也可以:

- 在 `LogRecord.Fields` 中加入 traceId/spanId(通过 `Tracer` 从 context 提取), 实现日志与追踪的关联。
- 实现更多 Transport, 如:
  - 发送到 ELK/ClickHouse 的 HTTP/UDP Transport。
  - 写入 Redis/Upstash 的队列, 用于构建更灵活的日志管道。
