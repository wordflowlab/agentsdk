---
title: HTTP Server 接入示例
description: 使用 pkg/server 暴露 AgentSDK 的 Chat 接口
navigation: false
---

# HTTP Server 接入示例

本示例展示如何使用 `pkg/server` 为 AgentSDK 提供一个最小可用的 HTTP 接入层,便于前端和第三方服务集成。

- 路径: `examples/server-http/main.go`
- 功能:
  - 启动一个本地 HTTP 服务器 (`:8080`)
  - 提供 `POST /v1/agents/chat` 接口
  - 内部创建 Agent 并调用一次 `Chat`, 返回最终文本结果

> 这是构建标准化 HTTP Chat 服务的第一步, 后续可以在此基础上扩展 streaming、会话管理、tools introspection 等能力。

## 1. Server 初始化 (同步接口)

```go
// pkg/server/server.go
type Server struct {
    deps *agent.Dependencies
}

func New(deps *agent.Dependencies) *Server {
    return &Server{deps: deps}
}

// ChatRequest/ChatResponse 定义了同步 Chat 的 HTTP 请求/响应结构。
type ChatRequest struct {
    TemplateID  string                 `json:"template_id"`
    Input       string                 `json:"input"`
    ModelConfig *types.ModelConfig     `json:"model_config,omitempty"`
    Sandbox     *types.SandboxConfig   `json:"sandbox,omitempty"`
    Middlewares []string               `json:"middlewares,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ChatResponse struct {
    AgentID      string `json:"agent_id"`
    Text         string `json:"text"`
    Status       string `json:"status"`
    ErrorMessage string `json:"error_message,omitempty"`
}
```

### ChatHandler

```go
// ChatHandler 处理 POST /v1/agents/chat:
// - 解析请求 JSON
// - 使用 TemplateID + ModelConfig + Sandbox + Middlewares + Metadata 创建 Agent
// - 调用 ag.Chat(ctx, input)
// - 返回同步结果
func (s *Server) ChatHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }

        var req ChatRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid JSON body", http.StatusBadRequest)
            return
        }

        if req.TemplateID == "" || req.Input == "" {
            http.Error(w, "template_id and input are required", http.StatusBadRequest)
            return
        }

        ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
        defer cancel()

        agentConfig := &types.AgentConfig{
            TemplateID:  req.TemplateID,
            ModelConfig: req.ModelConfig,
            Sandbox:     req.Sandbox,
            Middlewares: req.Middlewares,
            Metadata:    req.Metadata,
        }

        ag, err := agent.Create(ctx, agentConfig, s.deps)
        if err != nil {
            writeJSON(w, http.StatusInternalServerError, &ChatResponse{
                Status:       "error",
                ErrorMessage: err.Error(),
            })
            return
        }
        defer ag.Close()

        result, err := ag.Chat(ctx, req.Input)
        if err != nil {
            writeJSON(w, http.StatusInternalServerError, &ChatResponse{
                AgentID:      ag.ID(),
                Status:       "error",
                ErrorMessage: err.Error(),
            })
            return
        }

        writeJSON(w, http.StatusOK, &ChatResponse{
            AgentID: ag.ID(),
            Text:    result.Text,
            Status:  "ok",
        })
    })
}
```

## 2. 流式接口: ChatStreamHandler (SSE)

在同步接口基础上, `pkg/server` 还提供了一个基于 **Server-Sent Events** 的流式接口:

```go
// ChatStreamHandler 提供基于 SSE 的流式 Chat 接口:
//   POST /v1/agents/chat/stream
//
// 请求体与 ChatHandler 相同, 响应为 text/event-stream。
// 每条消息为一行 JSON:
//   data: {"cursor":1,"bookmark":{...},"event":{...}}\n
func (s *Server) ChatStreamHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }

        flusher, ok := w.(http.Flusher)
        if !ok {
            http.Error(w, "streaming unsupported", http.StatusInternalServerError)
            return
        }

        var req ChatRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid JSON body", http.StatusBadRequest)
            return
        }

        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
        defer cancel()

        agentConfig := &types.AgentConfig{ /* 同 ChatHandler */ }
        ag, err := agent.Create(ctx, agentConfig, s.deps)
        if err != nil {
            http.Error(w, "create agent failed: "+err.Error(), http.StatusInternalServerError)
            return
        }
        defer ag.Close()

        // SSE 头
        w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")

        // 订阅进度+监控事件
        eventCh := ag.Subscribe(
            []types.AgentChannel{types.ChannelProgress, types.ChannelMonitor},
            nil,
        )
        defer ag.Unsubscribe(eventCh)

        // 启动 Chat, 并在后台运行
        go func() {
            _, _ = ag.Chat(ctx, req.Input)
        }()

        enc := json.NewEncoder(w)

        // 不断从 eventCh 读取事件并写出到 SSE 流
        for {
            select {
            case <-ctx.Done():
                return
            case env, ok := <-eventCh:
                if !ok {
                    return
                }

                w.Write([]byte("data: "))
                if err := enc.Encode(env); err != nil {
                    return
                }
                w.Write([]byte("\n"))
                flusher.Flush()

                if evt, ok := env.Event.(types.EventType); ok && evt.EventType() == "done" {
                    return
                }
            }
        }
    })
}
```

前端可以使用 `EventSource` 或类似工具按需解析 `env.Event` 中的细粒度事件(文本块、工具调用、token 使用等)。

## 3. 示例 main: `examples/server-http/main.go`

```go
func main() {
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    if apiKey == "" {
        log.Println("[WARN] ANTHROPIC_API_KEY not set, server will still start but chat requests will fail until a key is provided.")
    }

    ctx := context.Background()

    // 1. 工具注册表 + 内置工具
    toolRegistry := tools.NewRegistry()
    builtin.RegisterAll(toolRegistry)

    // 2. Sandbox / Provider / Store / 模板
    sandboxFactory := sandbox.NewFactory()
    providerFactory := &provider.AnthropicFactory{}
    jsonStore, _ := store.NewJSONStore(".agentsdk-server")

    templateRegistry := agent.NewTemplateRegistry()
    templateRegistry.Register(&types.AgentTemplateDefinition{
        ID:    "assistant",
        Model: "claude-sonnet-4-5",
        SystemPrompt: "You are a helpful assistant with file and memory access. " +
            "Use filesystem and memory tools when appropriate.",
        Tools: []interface{}{"Read", "Write", "Bash"},
    })

    deps := &agent.Dependencies{
        Store:            jsonStore,
        SandboxFactory:   sandboxFactory,
        ToolRegistry:     toolRegistry,
        ProviderFactory:  providerFactory,
        TemplateRegistry: templateRegistry,
    }

    // 3. 创建 Server 并注册路由
    srv := server.New(deps)

    mux := http.NewServeMux()
    mux.Handle("/v1/agents/chat", srv.ChatHandler())
    mux.Handle("/v1/agents/chat/stream", srv.ChatStreamHandler())

    addr := ":8080"
    httpServer := &http.Server{
        Addr:              addr,
        Handler:           mux,
        ReadHeaderTimeout: 10 * time.Second,
    }

    fmt.Printf("HTTP server started at http://localhost%s\n", addr)
    fmt.Println("POST /v1/agents/chat with JSON body for sync chat.")
    fmt.Println("POST /v1/agents/chat/stream with JSON body for SSE streaming.")

    if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("HTTP server failed: %v", err)
    }
}
```

## 4. 调用示例

### 同步 Chat 请求

```bash
curl -X POST http://localhost:8080/v1/agents/chat \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": "assistant",
    "input": "请帮我总结一下 README",
    "metadata": {"user_id": "alice"},
    "middlewares": ["filesystem", "agent_memory"]
  }'
```

### 响应

```json
{
  "agent_id": "agt:...",
  "text": "这是 README 的总结...",
  "status": "ok"
}
```

### 流式 Chat 请求 (SSE)

```bash
curl -N -X POST http://localhost:8080/v1/agents/chat/stream \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": "assistant",
    "input": "请边思考边输出, 并展示工具调用过程",
    "metadata": {"user_id": "alice"},
    "middlewares": ["filesystem", "agent_memory"]
  }'
```

输出将是一系列 `data: {...}` 行, 每行一个 JSON 事件。

## 5. 后续扩展方向

在这个最小版 HTTP Server 基础上,可以逐步扩展:

- **Streaming**: 改造 `ChatHandler`, 支持 SSE/WebSocket, 将 `EventBus` 的 Progress 事件实时推给前端。
- **Session API**: 暴露基于 `pkg/session` 的会话创建/查询接口, 支持跨请求持续会话。
- **Tools Introspection**: 暴露当前可用工具列表, 方便前端展示和调试。
- **OpenAPI 生成**: 为 `/v1/agents/*` 接口生成 OpenAPI 文档, 方便生成类型安全客户端或接入 API 网关。

这些都可以在 `pkg/server` 目录中按子文件逐步演进, 不影响核心 Agent 运行时。
