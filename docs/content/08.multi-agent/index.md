---
title: 多Agent系统总览
description: 构建多个 Agent 协作的复杂系统
navigation: false
---

# 多Agent系统总览

多Agent系统支持 Agent 之间的协作、通信和任务调度。

## 📚 分类

### [Agent Pool](/multi-agent/pool)
Agent 池管理，支持动态创建和销毁

### [Agent Room](/multi-agent/room)
Agent 房间，实现多Agent消息路由

### [Scheduler](/multi-agent/scheduler)
任务调度器，智能分配任务给合适的 Agent

## 🚀 快速开始

```go
// 创建 Agent Pool
pool := agent.NewPool(config)
pool.RegisterAgent("coder", coderAgent)
pool.RegisterAgent("reviewer", reviewerAgent)

// 分发任务
result, err := pool.Execute(ctx, task)
```

## 📖 相关文档

- [多Agent示例](/examples/multi-agent)
- [最佳实践：多Agent协作](/best-practices)
