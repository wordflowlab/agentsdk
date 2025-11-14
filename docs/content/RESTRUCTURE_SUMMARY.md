# 文档结构重组总结

## 重组日期
2025-11-14

## 重组目标
参考 Mastra 的文档组织方式，将 AgentSDK 文档从 7 个分类扩展到 15 个分类，使文档结构更清晰、更易导航。

## 主要变化

### 1. 目录结构

#### 之前（7个主分类）
```
1. introduction/      - 介绍
2. core-concepts/     - 核心概念
3. providers/         - 模型提供商
4. memory/            - 记忆系统
5. guides/            - 指南（混乱！）
6. examples/          - 示例
6. api-reference/     - API参考
7. best-practices/    - 最佳实践
```

#### 之后（15个主分类）
```
1. introduction/      - 介绍
2. core-concepts/     - 核心概念
3. providers/         - 模型提供商
4. memory/            - 记忆系统
5. tools/             - 工具系统 ⭐ 新增
6. middleware/        - 中间件 ⭐ 新增
7. workflows/         - 工作流 ⭐ 新增
8. multi-agent/       - 多Agent系统 ⭐ 新增
9. deployment/        - 部署指南 ⭐ 新增
10. observability/    - 可观测性 ⭐ 新增
11. evals/            - 评估系统 ⭐ 新增
12. examples/         - 代码示例（重组）
13. guides/           - 教程指南（重组）
14. api-reference/    - API参考（扩展）
15. best-practices/   - 最佳实践
```

### 2. 文档移动

#### 从 guides/ 移出
- `2.tools/` → `5.tools/2.builtin/`
- `3.middleware/` → `6.middleware/2.builtin/`
- `4.multi-agent/` → `8.multi-agent/1.overview/`
- `5.workflow-agents/` → `7.workflows/1.overview/`
- `memory*.md` (5个文件) → `12.examples/2.memory/`
- `logging.md` → `10.observability/1.logging/`
- `evals.md` → `11.evals/1.overview/`
- `server-http.md` → `9.deployment/2.local/`
- `mcp-server.md` → `5.tools/3.mcp/`
- `router.md` → `12.examples/8.scenarios/`
- `runtime.md` → `9.deployment/1.overview/`
- `0.quick-start.md` → `13.guides/1.quickstart/`
- `1.basic-agent.md` → `13.guides/1.quickstart/`

#### 从 examples/ 重组
- `1.agent/` → `12.examples/1.basic/`
- `2.providers/` → `12.examples/1.basic/`
- `3.tools/` → `12.examples/3.tools/`
- `4.scenarios/` → `12.examples/8.scenarios/`

#### 从 4.examples/ 合并
- `memory-semantic.md` → `12.examples/2.memory/`
- `workflow-semantic.md` → `12.examples/5.workflows/`

### 3. 新增内容

#### 索引文件
为每个主分类创建了索引文件（index.md）：
- 5.tools/index.md
- 6.middleware/index.md
- 7.workflows/index.md
- 8.multi-agent/index.md
- 9.deployment/index.md
- 10.observability/index.md
- 11.evals/index.md
- 12.examples/index.md
- 13.guides/index.md
- 14.api-reference/index.md

#### 配置文件
为每个主分类创建了 _dir.yml 配置文件，定义：
- 中文标题
- Heroicons 图标
- 导航顺序

### 4. 细分结构

#### Tools（工具系统）
```
5.tools/
├── 1.overview/
├── 2.builtin/      - 内置工具
├── 3.mcp/          - MCP协议
└── 4.custom/       - 自定义工具
```

#### Middleware（中间件）
```
6.middleware/
├── 1.overview/
├── 2.builtin/      - 内置中间件
└── 3.custom/       - 自定义中间件
```

#### Workflows（工作流）
```
7.workflows/
├── 1.overview/
├── 2.basic/        - 基础工作流
└── 3.advanced/     - 高级模式
```

#### Multi-Agent（多Agent）
```
8.multi-agent/
├── 1.overview/
├── 2.pool/         - Agent Pool
├── 3.room/         - Agent Room
└── 4.scheduler/    - Scheduler
```

#### Deployment（部署）
```
9.deployment/
├── 1.overview/
├── 2.local/        - 本地部署
├── 3.docker/       - Docker
├── 4.kubernetes/   - K8s
├── 5.serverless/   - Serverless
└── 6.cloud-sandbox/ - 云端沙箱
```

#### Observability（可观测性）
```
10.observability/
├── 1.logging/      - 日志
├── 2.monitoring/   - 监控
├── 3.tracing/      - 追踪
└── 4.debugging/    - 调试
```

#### Evals（评估）
```
11.evals/
├── 1.overview/
├── 2.builtin-scorers/
├── 3.custom-scorers/
└── 4.ci-integration/
```

#### Examples（示例）
```
12.examples/
├── 1.basic/        - 基础示例
├── 2.memory/       - 记忆示例（6个文件）
├── 3.tools/        - 工具示例
├── 4.middleware/   - 中间件示例
├── 5.workflows/    - 工作流示例
├── 6.multi-agent/  - 多Agent示例
├── 7.integration/  - 集成示例
└── 8.scenarios/    - 场景示例
```

#### Guides（教程）
```
13.guides/
├── 1.quickstart/   - 快速开始
├── 2.tutorials/    - 完整教程
└── 3.migrations/   - 迁移指南
```

#### API Reference（API参考）
```
14.api-reference/
├── 1.agent/
├── 2.provider/
├── 3.tools/
├── 4.middleware/
├── 5.memory/
├── 6.workflow/
├── 7.events/
└── 8.types/
```

## 优势

### 1. 更清晰的职责分离
- **Tools、Middleware、Workflows、Multi-Agent** 各自独立，不再混在 Guides 里
- **Deployment、Observability、Evals** 有明确的独立分类

### 2. 更好的学习路径
- **Introduction** → **Core Concepts** → **具体功能** → **Examples** → **Guides** → **Best Practices**

### 3. 更易扩展
- 每个功能模块都有独立的目录和子分类
- 新增文档只需放到对应分类下

### 4. 对标行业最佳实践
- 参考 Mastra、LangChain 等优秀文档结构
- 细粒度分类，便于搜索和导航

## 后续工作

### 高优先级
1. 为每个 API 创建独立文档（目标：40+个）
2. 补充更多代码示例（目标：50+个）
3. 创建完整的端到端教程（目标：4个项目）

### 中优先级
4. 更新文档内部链接
5. 添加文档间的"Related"链接
6. 创建学习路径地图

### 低优先级
7. 录制视频教程
8. 添加交互式示例
9. 国际化（英文版）

## 兼容性

- 所有原有文档都已保留，只是位置调整
- 文档内容未做修改，只需更新内部链接即可
- 目录编号从 1 到 15，保持连续性

## 验证

运行以下命令验证文档构建：

```bash
cd /Users/coso/Documents/dev/ai/wordflowlab/agentsdk/docs
npm run dev
```

访问 http://localhost:3000 查看新的文档结构。
