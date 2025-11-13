---
title: 示例代码
description: 通过实际示例学习 AgentSDK 的各种功能和最佳实践
---

# 示例代码

通过真实可运行的代码示例，快速学习 AgentSDK 的核心功能。所有示例代码都可以在 [GitHub 仓库](https://github.com/wordflowlab/agentsdk/tree/main/examples) 中找到并直接运行。

## 📚 示例分类

### 🚀 快速入门

<div class="grid grid-cols-1 md:grid-cols-2 gap-4 my-6">
  <a href="/examples/basic-agent" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">基础 Agent</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">创建第一个 Agent，发送消息并接收响应</p>
  </a>
</div>

### 🛠️ 工具系统

<div class="grid grid-cols-1 md:grid-cols-2 gap-4 my-6">
  <a href="/examples/tools/builtin" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">内置工具</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">使用文件系统、Bash、HTTP 等内置工具</p>
  </a>
  <a href="/examples/tools/mcp" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">MCP 工具</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">集成 MCP 协议工具服务器</p>
  </a>
  <a href="/examples/tools/custom" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">自定义工具</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">创建和注册自定义工具</p>
  </a>
</div>

### 🧅 中间件系统

<div class="grid grid-cols-1 md:grid-cols-2 gap-4 my-6">
  <a href="/examples/middleware" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">中间件使用</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">文件系统、子Agent、总结等中间件的使用</p>
  </a>
</div>

### 🌊 工作流 Agent

<div class="grid grid-cols-1 md:grid-cols-2 gap-4 my-6">
  <a href="/examples/workflow-agents" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">工作流编排</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">ParallelAgent、SequentialAgent、LoopAgent 三种编排模式</p>
  </a>
</div>

### 👥 多 Agent 协作

<div class="grid grid-cols-1 md:grid-cols-2 gap-4 my-6">
  <a href="/examples/multi-agent" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">协作模式</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">Agent Pool、Room、Scheduler 等协作模式</p>
  </a>
</div>

### 💾 数据持久化

<div class="grid grid-cols-1 md:grid-cols-2 gap-4 my-6">
  <a href="/examples/session" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">Session 持久化</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">PostgreSQL 和 MySQL 持久化 Agent 会话和事件</p>
  </a>
</div>

### 🚀 部署实践

<div class="grid grid-cols-1 md:grid-cols-2 gap-4 my-6">
  <a href="/examples/deployment" class="block p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-primary-500 transition-colors">
    <h3 class="font-semibold mb-2">部署方式</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400">本地、阿里云、火山引擎等部署方式</p>
  </a>
</div>

## 💡 如何使用示例

### 1. 克隆仓库

```bash
git clone https://github.com/wordflowlab/agentsdk.git
cd agentsdk/examples
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 运行示例

每个示例都可以直接运行：

```bash
# 运行简单示例
go run simple/main.go

# 运行 MCP 示例
go run mcp/server/main.go
```

### 4. 查看帮助

大多数示例支持命令行参数：

```bash
go run simple/main.go -help
```

## 📖 学习路径

**建议按以下顺序学习**：

1. **基础 Agent** - 理解 Agent 的创建和基本使用
2. **内置工具** - 学习如何让 Agent 使用工具
3. **自定义工具** - 创建适合业务需求的工具
4. **中间件** - 掌握洋葱模型中间件
5. **多 Agent 协作** - 构建复杂的 Agent 系统
6. **部署实践** - 了解生产环境部署

## 🔗 相关资源

- [完整示例仓库](https://github.com/wordflowlab/agentsdk/tree/main/examples)
- [API 参考](/api-reference)
- [最佳实践](/best-practices)
- [核心概念](/core-concepts)

## 🤝 贡献示例

如果您有好的示例想要分享，欢迎提交 Pull Request！请参考 [贡献指南](https://github.com/wordflowlab/agentsdk/blob/main/CONTRIBUTING.md)。
