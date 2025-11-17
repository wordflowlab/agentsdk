# AgentSDK Production Server

> 🚀 **生产级 AI 应用服务器** - 完整的认证、监控、部署支持

---

## 📋 概览

AgentSDK Server 是一个生产就绪的应用服务器层，提供：

- ✅ **认证授权**: API Key、JWT
- ✅ **速率限制**: 可配置的请求限制
- ✅ **CORS 支持**: 完整的跨域配置
- ✅ **结构化日志**: JSON 格式日志
- ✅ **健康检查**: Kubernetes 就绪探针
- ✅ **指标收集**: Prometheus 集成
- ✅ **Docker 支持**: 多阶段构建
- ✅ **Kubernetes**: 完整的 K8s 部署配置

---

## 🚀 快速开始

### 使用默认配置

```go
package main

import (
    "log"
    "github.com/wordflowlab/agentsdk/pkg/store"
    "github.com/wordflowlab/agentsdk/server"
)

func main() {
    // 创建存储
    st, _ := store.NewJSONStore(".data")
    
    // 创建依赖
    deps := &server.Dependencies{
        Store: st,
    }
    
    // 创建服务器（使用默认配置）
    srv, err := server.New(server.DefaultConfig(), deps)
    if err != nil {
        log.Fatal(err)
    }
    
    // 启动服务器
    srv.Start()
}
```

### 自定义配置

```go
config := &server.Config{
    Host: "0.0.0.0",
    Port: 8080,
    Mode: "production",
    
    // 认证配置
    Auth: server.AuthConfig{
        APIKey: server.APIKeyConfig{
            Enabled: true,
            HeaderName: "X-API-Key",
            Keys: []string{"your-secure-api-key"},
        },
    },
    
    // CORS 配置
    CORS: server.CORSConfig{
        Enabled: true,
        AllowOrigins: []string{"https://yourdomain.com"},
        AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    },
    
    // 速率限制
    RateLimit: server.RateLimitConfig{
        Enabled: true,
        RequestsPerIP: 1000,
        WindowSize: time.Minute,
    },
}

srv, _ := server.New(config, deps)
srv.Start()
```

---

## 🐳 Docker 部署

### 构建镜像

```bash
docker build -t agentsdk/server:latest -f server/deploy/docker/Dockerfile .
```

### 运行容器

```bash
docker run -p 8080:8080 \
  -e API_KEY=your-api-key \
  -e MODE=production \
  agentsdk/server:latest
```

### 使用 Docker Compose

```bash
cd server/deploy/docker
docker-compose up -d
```

---

## ☸️ Kubernetes 部署

### 应用配置

```bash
kubectl apply -f server/deploy/k8s/
```

### 检查状态

```bash
kubectl get pods -l app=agentsdk
kubectl get svc agentsdk-server
```

### 查看日志

```bash
kubectl logs -f deployment/agentsdk-server
```

### 扩容

```bash
kubectl scale deployment agentsdk-server --replicas=5
```

---

## 📝 环境变量

| 变量 | 描述 | 默认值 |
|------|------|--------|
| `HOST` | 服务器监听地址 | `0.0.0.0` |
| `PORT` | 服务器端口 | `8080` |
| `MODE` | 运行模式 (`development`/`production`) | `development` |
| `API_KEY` | API 密钥 | `dev-key-12345` |

---

## 🔐 认证

### API Key 认证

```bash
curl -H "X-API-Key: your-api-key" \
  http://localhost:8080/v1/agents
```

### JWT 认证

```bash
curl -H "Authorization: Bearer your-jwt-token" \
  http://localhost:8080/v1/agents
```

---

## 📊 监控

### 健康检查

```bash
curl http://localhost:8080/health
```

响应：
```json
{
  "status": "healthy",
  "checks": {
    "database": "ok",
    "timestamp": "2024-11-17T12:00:00Z"
  },
  "version": "2.0.0"
}
```

### Prometheus 指标

```bash
curl http://localhost:8080/metrics
```

---

## 🔧 配置选项

### CORS 配置

```go
CORS: server.CORSConfig{
    Enabled: true,
    AllowOrigins: []string{"https://app.example.com"},
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders: []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge: 86400,
}
```

### 速率限制配置

```go
RateLimit: server.RateLimitConfig{
    Enabled: true,
    RequestsPerIP: 100,        // 每个时间窗口的请求数
    WindowSize: time.Minute,    // 时间窗口大小
    BurstSize: 20,              // 突发容量
}
```

### 日志配置

```go
Logging: server.LoggingConfig{
    Level: "info",              // debug, info, warn, error
    Format: "json",             // json 或 text
    Output: "stdout",           // stdout 或文件路径
    Structured: true,           // 结构化日志
}
```

---

## 📡 API 端点

### 核心业务

- `POST/GET/DELETE /v1/agents` - Agent 管理
- `POST /v1/agents/chat` - 对话
- `POST /v1/agents/chat/stream` - 流式对话
- `GET/PUT /v1/memory/working` - 工作记忆
- `POST /v1/memory/semantic/search` - 语义搜索
- `POST/GET/DELETE /v1/sessions` - 会话管理
- `POST/GET/DELETE /v1/workflows` - 工作流管理

### 可观测性

- `GET /health` - 健康检查
- `GET /metrics` - Prometheus 指标
- `POST /v1/telemetry/metrics` - 记录指标
- `POST /v1/telemetry/traces/query` - 查询追踪

完整 API 文档请参考: [API Reference](../../docs/content/14.api-reference/)

---

## 🏗️ 架构

```
┌─────────────────────────────────────┐
│   Client SDKs                        │
│   - client-js, React, AI SDK         │
└────────────┬────────────────────────┘
             │ HTTP/WebSocket
┌────────────▼────────────────────────┐
│   server/ (生产级应用服务器)         │
│   ├── 认证授权                       │
│   ├── 速率限制                       │
│   ├── CORS 处理                      │
│   ├── 结构化日志                     │
│   ├── 健康检查                       │
│   └── 指标收集                       │
└────────────┬────────────────────────┘
             │ 纯 Go 接口
┌────────────▼────────────────────────┐
│   pkg/ (核心 SDK)                   │
│   - Agent, Memory, Workflow...       │
└─────────────────────────────────────┘
```

---

## 🔄 与 cmd/agentsdk 的对比

| 特性 | cmd/agentsdk | server/ |
|------|--------------|---------|
| **定位** | 演示/开发 | 生产部署 |
| **认证** | ❌ | ✅ API Key + JWT |
| **速率限制** | ❌ | ✅ |
| **CORS** | 基础 | 完整配置 |
| **日志** | 简单 | 结构化 |
| **监控** | ❌ | ✅ Health + Metrics |
| **部署** | 手动 | Docker + K8s |
| **生产就绪** | ❌ | ✅ |

---

## 🛠️ 开发

### 本地运行

```bash
go run ./cmd/agentsdk-server
```

### 构建

```bash
go build -o agentsdk-server ./cmd/agentsdk-server
```

### 测试

```bash
go test ./server/...
```

---

## 📚 相关文档

- [架构设计](../SERVER_ARCHITECTURE.md) - 完整架构文档
- [核心 SDK](../docs/content/18.architecture/2.core-sdk.md) - pkg/ 设计
- [HTTP 层](../docs/content/18.architecture/3.http-layer.md) - 原 cmd/ 设计
- [客户端 SDK](../docs/content/18.architecture/4.client-sdk.md) - client-sdks 设计

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

## 📄 License

MIT License - see LICENSE file for details

---

**AgentSDK Server - 让 AI 应用部署变得简单！** 🚀
