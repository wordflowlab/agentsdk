---
title: 模型路由示例
description: 使用 StaticRouter 根据 routing_profile 在多个模型之间进行路由
navigation: false
---

# 模型路由示例

本示例展示如何使用 `pkg/router` 提供的 `StaticRouter`，在相同模板下，根据不同的 `routing_profile` 选择不同的模型配置，例如：

- `quality-first`：优先使用高质量模型（如 anthropic 的高级模型）；
- `cost-first`：优先使用低成本模型（如 deepseek）。

## 示例位置

- 源码路径：`examples/router/main.go`

## 运行前准备

1. 设置相应模型提供商的 API Key，例如：

```bash
export ANTHROPIC_API_KEY=你的_anthropic_key
export DEEPSEEK_API_KEY=你的_deepseek_key
```

2. 进入示例目录并运行：

```bash
cd examples
go run ./router
```

## 代码结构概览

示例主要包含以下几个部分：

1. 初始化基础依赖：内存 Store、本地 Sandbox、ProviderFactory；
2. 注册一个简单的模板 `router-demo`；
3. 创建一个 `StaticRouter`，定义 `quality` 和 `cost` 两种路由策略；
4. 分别创建 `quality-first` 和 `cost-first` 两个 Agent，发送相同的问题；
5. 打印两个 Agent 的回复。

关键代码片段：

```go
defaultModel := &types.ModelConfig{
    Provider: "anthropic",
    Model:    "claude-3-5-sonnet-20241022",
}

routes := []router.StaticRouteEntry{
    {
        Task:     "chat",
        Priority: router.PriorityQuality,
        Model: &types.ModelConfig{
            Provider: "anthropic",
            Model:    "claude-3-5-sonnet-20241022",
        },
    },
    {
        Task:     "chat",
        Priority: router.PriorityCost,
        Model: &types.ModelConfig{
            Provider: "deepseek",
            Model:    "deepseek-chat",
        },
    },
}

rt := router.NewStaticRouter(defaultModel, routes)
```

创建 Agent 时，通过 `RoutingProfile` 指定不同的路由偏好：

```go
qualityAgent, _ := agent.Create(ctx, &types.AgentConfig{
    TemplateID:     "router-demo",
    RoutingProfile: string(router.PriorityQuality),
}, deps)

costAgent, _ := agent.Create(ctx, &types.AgentConfig{
    TemplateID:     "router-demo",
    RoutingProfile: string(router.PriorityCost),
}, deps)
```

## 运行效果

运行示例后，你会看到类似输出：

```text
==== quality-first ====
quality-first reply:
...（来自高质量模型的回答）...

==== cost-first ====
cost-first reply:
...（来自成本友好模型的回答）...
```

由于提示词中让模型“自我介绍”，实际输出会根据你配置的模型而有所不同。

## 设计要点与下一步

- 当前能力:
  - 支持按任务 + 优先级选择模型；
  - 支持为不同业务场景定义不同 `routing_profile`；
  - Router 作为可选层, 不影响现有简单用法。

后续可以在此基础上继续演进:

- 接入统计信息, 做动态路由与回退；
- 从配置文件或远程配置中心加载路由规则；
- 在 Playground 中暴露 `routing_profile` 作为可选项。
