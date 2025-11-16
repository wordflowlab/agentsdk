---
description: 基于任务清单执行章节写作
argument-hint: "章节编号或任务ID"
allowed-tools: ["Read", "Write", "Bash"]
models:
  preferred:
    - claude-sonnet-4-5
    - gpt-4-turbo
    - qwen-max
  minimum-capabilities:
    - tool-calling
scripts:
  sh: scripts/bash/check-state.sh
  ps: scripts/powershell/check-state.ps1
---

# 章节写作工作流

基于七步方法论流程执行章节写作。

## 前置检查

1. 运行脚本 `{SCRIPT}` 检查写作状态
2. 确认规格说明和计划已准备就绪

## 执行流程

### 1. 加载上下文

**按顺序读取（优先级从高到低）：**

1. **创作宪法与风格**：
   - `memory/constitution.md`（创作原则 - 最高优先级）
   - `memory/style-reference.md`（风格参考 - 如有）

2. **规格说明与计划**：
   - `stories/*/specification.md`（故事规格）
   - `stories/*/creative-plan.md`（创作计划）
   - `stories/*/tasks.md`（当前任务）

3. **状态与数据**：
   - `spec/tracking/character-state.json`（角色状态）
   - `spec/tracking/relationships.json`（关系网络）
   - `spec/tracking/plot-tracker.json`（情节追踪 - 如有）

4. **知识库**：
   - `spec/knowledge/` 相关文件（世界观、角色档案等）
   - `stories/*/content/`（前文内容 - 了解前情）

### 2. 选择写作任务

从 `tasks.md` 中选择状态为 `pending` 的写作任务，标记为 `in_progress`。

### 3. 写作前提醒

**基于宪法原则提醒：**
- 核心价值观要点
- 质量标准要求
- 风格一致性准则

**基于规格要求提醒：**
- P0 必须包含的元素
- 目标读者特征
- 内容红线提醒

### 4. 执行写作

根据计划创作内容：
- **开场**：吸引读者，承接前文
- **发展**：推进情节，深化人物
- **转折**：制造冲突或悬念
- **收尾**：适当收束，引出下文

### 5. 质量自检

**宪法合规检查：**
- 是否符合核心价值观
- 是否达到质量标准
- 是否保持风格一致

**规格符合检查：**
- 是否包含必要元素
- 是否符合目标定位
- 是否遵守约束条件

**计划执行检查：**
- 是否按照章节架构
- 是否符合节奏设计
- 是否达到字数要求

### 6. 保存和更新

- 将章节内容保存到 `stories/*/content/`
- 更新任务状态为 `completed`
- 记录完成时间和字数

## 写作要点

- **遵循宪法**：始终符合创作原则
- **满足规格**：确保包含必要元素
- **执行计划**：按照技术方案推进
- **完成任务**：系统化推进任务清单
- **持续验证**：定期运行 `/analyze` 检查

## 完成后行动

完成后：
- 继续下一个写作任务
- 每5章运行 `/analyze` 进行质量检查
- 发现问题及时调整计划

## 与方法论的关系

```
/constitution → 提供创作原则
     ↓
/specify → 定义故事需求
     ↓
/clarify → 澄清关键决策
     ↓
/plan → 制定技术方案
     ↓
/tasks → 分解执行任务
     ↓
/write → 【当前】执行写作
     ↓
/analyze → 验证质量一致
```

记住：写作是执行层，要严格遵循上层的规格和计划。

