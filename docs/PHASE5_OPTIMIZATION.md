# Phase 5: 与 DeepAgents 对标优化

> 时间: 2025-11-09
> 参考: `/Users/coso/Documents/dev/python/deepagents`
> 状态: ✅ 完成

## 概述

Phase 5 对标 DeepAgents Python 实现,完成了以下优化:

1. ✅ **bash_run HITL 恢复机制评估** - 确认无需实现(WriteFlow-SDK 使用无状态设计)
2. ✅ **FilesystemMiddleware 路径安全验证** - 防止路径遍历攻击
3. ✅ **FilesystemMiddleware 配置灵活性** - 支持自定义工具描述和系统提示词
4. ✅ **SubAgentMiddleware 提示词扩展** - 提供详细的使用指南

## 1. bash_run HITL 恢复机制评估

### 分析结论

**无需实现** - WriteFlow-SDK 采用了更优的无状态设计:

| 对比维度 | DeepAgents | WriteFlow-SDK |
|---------|-----------|---------------|
| Shell 执行 | 持久化会话(UntrackedValue) | 无状态执行(Sandbox.Exec) |
| 状态管理 | 需要恢复机制 | 无需恢复(每次独立) |
| 设计复杂度 | 高(需处理状态丢失) | 低(无状态) |
| 可靠性 | 需要恢复逻辑 | 天然可靠 |

### 相关文件

- `pkg/tools/builtin/bash_run.go` - 无状态 bash 执行工具
- `pkg/sandbox/interface.go` - Sandbox 接口定义

---

## 2. FilesystemMiddleware 路径安全验证

### 新增功能

实现了完整的路径安全验证机制,防止路径遍历攻击:

```go
type FilesystemMiddlewareConfig struct {
    Backend              backends.BackendProtocol
    TokenLimit           int
    EnableEviction       bool

    // 🆕 Phase 5 新增
    AllowedPathPrefixes  []string  // 路径白名单
    EnablePathValidation bool      // 启用路径验证(默认 false)
}
```

### 路径验证规则

1. **阻止路径遍历**:
   - 禁止 `..` (父目录访问)
   - 禁止 `~` (home 目录访问)

2. **路径规范化**:
   - 使用 `filepath.Clean()` 规范化路径
   - 统一使用 `/` 分隔符(跨平台兼容)
   - 确保以 `/` 开头

3. **前缀白名单**:
   - 支持配置允许的路径前缀
   - 智能匹配(自动处理尾部斜杠)

### 使用示例

```go
middleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
    Backend:              myBackend,
    EnablePathValidation: true,
    AllowedPathPrefixes:  []string{"/workspace/", "/tmp/"},
})

// ✅ 允许: /workspace/file.txt
// ✅ 允许: /tmp/cache.dat
// ❌ 禁止: ../etc/passwd
// ❌ 禁止: ~/secrets.txt
// ❌ 禁止: /etc/passwd (不在白名单)
```

### 性能影响

基准测试结果(Apple M1):

- **启用验证**: 107.2 ns/op
- **禁用验证**: 2.579 ns/op
- **性能开销**: ~100 ns/op (微不足道)

### 集成的工具

路径验证已集成到所有 backend-based 工具:

- ✅ `fs_ls` - 目录列表
- ✅ `fs_edit` - 文件编辑
- ✅ `fs_glob` - 文件查找
- ✅ `fs_grep` - 内容搜索

**注意**: `fs_read` 和 `fs_write` 来自 builtin,使用 Sandbox 层面的安全控制。

### 相关文件

- `pkg/middleware/filesystem.go:220-275` - validatePath() 实现
- `pkg/middleware/filesystem_tools.go` - 工具集成
- `pkg/middleware/filesystem_security_test.go` - 安全测试

---

## 3. FilesystemMiddleware 配置灵活性

### 新增配置选项

```go
type FilesystemMiddlewareConfig struct {
    // ... 现有配置 ...

    // 🆕 Phase 5 新增
    CustomToolDescriptions map[string]string  // 自定义工具描述
    SystemPromptOverride   string             // 覆盖系统提示词
}
```

### 3.1 自定义工具描述

允许为每个工具自定义描述,优化 LLM 理解:

```go
middleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
    Backend: myBackend,
    CustomToolDescriptions: map[string]string{
        "fs_ls":   "列出目录内容(仅限项目目录)",
        "fs_edit": "精确编辑文件(支持多次替换)",
        "fs_glob": "查找文件(支持 **/*.go 等模式)",
        "fs_grep": "正则搜索文件内容",
    },
})
```

**支持的工具**: `fs_ls`, `fs_edit`, `fs_glob`, `fs_grep`

**实现原理**: 工具的 `Description()` 方法优先返回自定义描述:

```go
func (t *FsLsTool) Description() string {
    if t.middleware != nil && t.middleware.customToolDescriptions != nil {
        if customDesc, ok := t.middleware.customToolDescriptions["fs_ls"]; ok {
            return customDesc
        }
    }
    return "List directory contents with detailed file information"
}
```

### 3.2 SystemPrompt 覆盖

允许完全自定义文件系统工具的系统提示词:

```go
middleware := NewFilesystemMiddleware(&FilesystemMiddlewareConfig{
    Backend: myBackend,
    SystemPromptOverride: `## 项目文件系统规范

本项目使用严格的文件操作规范:
1. 所有路径必须在 /workspace/ 下
2. 编辑前必须先读取文件
3. 使用 fs_edit 而非 fs_write 修改现有文件
4. 大文件操作会自动分页
`,
})
```

**默认提示词** (无覆盖时):

```
### Filesystem Tools

You have access to the following filesystem tools:

- **fs_read**: Read file contents with optional offset/limit
- **fs_write**: Write content to a file
- **fs_ls**: List directory contents
- **fs_edit**: Edit files using string replacement
- **fs_glob**: Find files matching glob patterns
- **fs_grep**: Search for patterns in files

Guidelines:
- Always use relative paths from the sandbox root
- Large results will be automatically saved to files
- Use fs_edit for precise modifications
- Use fs_glob and fs_grep for code exploration
```

### 相关文件

- `pkg/middleware/filesystem.go:109-126` - SystemPrompt 注入
- `pkg/middleware/filesystem_tools.go` - 自定义描述实现

---

## 4. SubAgentMiddleware 提示词扩展

### 改进内容

参考 DeepAgents 的 `TASK_TOOL_DESCRIPTION` 和 `TASK_SYSTEM_PROMPT`,将 TaskTool 的提示词从 **~20 行扩展到 ~125 行**,提供:

1. **核心优势** (4 点):
   - 上下文隔离
   - 并行执行
   - Token 优化
   - 专注执行

2. **使用指南**:
   - ✅ 何时使用(4 种场景)
   - ❌ 何时不使用(4 种场景)

3. **4 个详细示例**:
   - 并行代码搜索
   - 顺序任务(依赖关系)
   - 错误案例(过度委派)
   - 批量处理

4. **最佳实践** (5 点):
   - 并行化优先
   - 详细指令
   - 利用隔离
   - 信任结果
   - 判断时机

### 示例片段

```go
func (t *TaskTool) Prompt() string {
    subagentTypes := t.middleware.ListSubAgents()
    agentList := "可用的子代理类型:\n"
    for _, name := range subagentTypes {
        agentList += fmt.Sprintf("  - %s\n", name)
    }

    return fmt.Sprintf(`启动短生命周期的子代理来处理复杂的、多步骤的独立任务...

%s

## 核心优势

1. **上下文隔离**: 每个子代理有独立的上下文窗口...
2. **并行执行**: 可以同时启动多个子代理...
3. **token优化**: 子代理处理完任务后只返回摘要结果...
4. **专注执行**: 每个子代理只需要关注一个独立任务...

## 何时使用 task 工具

✅ **应该使用的情况**:
- 任务复杂且需要多个步骤,可以完整地独立委派
- 任务之间相互独立,可以并行执行
- ...

❌ **不应该使用的情况**:
- 如果需要查看子代理完成后的中间推理或步骤
- 如果任务很简单...
- ...

## 使用示例

### 示例 1: 并行化搜索 ✨
...

## 重要提醒

1. **并行化是关键**: 尽可能使用并行执行来节省用户时间
2. **详细的指令**: 子代理无法回头问你问题,所以一次性给清楚
...
`, agentList)
}
```

### 效果

- **中文友好**: 面向中文用户,提升可读性
- **详细指导**: 覆盖常见使用场景和反模式
- **实战示例**: 4 个真实场景示例,便于理解

### 相关文件

- `pkg/middleware/subagent.go:239-365` - Prompt() 实现
- 参考: `deepagents/middleware/subagents.py:66-203`

---

## 测试覆盖

### 新增测试文件

**`pkg/middleware/filesystem_security_test.go`** (378 行):

1. **路径验证单元测试** (6 个场景):
   - 禁用验证 - 允许所有路径
   - 启用验证 - 阻止路径遍历 (..)
   - 启用验证 - 阻止 home 目录访问 (~)
   - 前缀白名单 - 允许合法路径
   - 前缀白名单 - 阻止非白名单路径
   - 路径规范化

2. **工具集成测试** (8 个工具场景):
   - fs_ls: 允许/阻止
   - fs_edit: 允许/阻止
   - fs_glob: 允许/阻止
   - fs_grep: 允许/阻止

3. **自定义配置测试**:
   - CustomToolDescriptions
   - SystemPromptOverride

4. **路径规范化测试** (4 种路径格式)

5. **性能基准测试** (2 个):
   - BenchmarkPathValidation
   - BenchmarkPathValidation_Disabled

### 测试结果

```bash
$ go test -v ./pkg/middleware/... -run "TestFilesystem.*"

✅ TestFilesystemMiddleware_PathValidation (6/6)
✅ TestFilesystemTools_PathValidationIntegration (8/8)
✅ TestFilesystemMiddleware_CustomToolDescriptions
✅ TestFilesystemMiddleware_SystemPromptOverride (2/2)
✅ TestFilesystemMiddleware_PathNormalization (4/4)

PASS
ok  	github.com/wordflowlab/agentsdk/pkg/middleware	2.033s
```

---

## 文件变更摘要

### 新增文件

- `docs/PHASE5_OPTIMIZATION.md` - 本文档

### 修改文件

1. **pkg/middleware/filesystem.go**:
   - 新增 `AllowedPathPrefixes`, `EnablePathValidation`, `CustomToolDescriptions`, `SystemPromptOverride` 配置字段
   - 实现 `validatePath()` 函数(220-275 行)
   - 支持 SystemPrompt 覆盖(109-126 行)

2. **pkg/middleware/filesystem_tools.go**:
   - 为所有工具添加 `middleware *FilesystemMiddleware` 字段
   - 集成 `validatePath()` 到 4 个工具的 Execute 方法
   - 实现自定义描述支持(Description 方法)

3. **pkg/middleware/subagent.go**:
   - 扩展 `TaskTool.Prompt()` 从 ~20 行到 ~125 行(239-365 行)
   - 添加详细的中文使用指南

4. **pkg/middleware/filesystem_security_test.go** (新增):
   - 378 行完整的安全和配置测试

---

## 对标 DeepAgents 完成度

| 功能模块 | DeepAgents | WriteFlow-SDK | 状态 |
|---------|-----------|---------------|-----|
| ResumableShell | ✅ (需要) | ❌ (无需,设计更优) | ✅ 确认无需 |
| Path Validation | ✅ | ✅ | ✅ 完成 |
| Tool Description | ✅ | ✅ | ✅ 完成 |
| SystemPrompt Override | ❌ | ✅ | ✅ 超越 |
| SubAgent Prompt | ✅ 详细 | ✅ 详细 | ✅ 对齐 |
| 中文文档 | ❌ | ✅ | ✅ 超越 |

---

## 后续建议

### 可选优化

1. **路径验证增强** (低优先级):
   - 支持符号链接检测
   - 支持路径长度限制
   - 支持文件类型过滤

2. **工具提示词优化** (低优先级):
   - 为 fs_read/fs_write 也支持自定义描述(需修改 builtin 包)
   - 支持工具级别的 SystemPrompt 注入

3. **监控和日志** (中优先级):
   - 添加路径验证失败的 metrics
   - 记录被阻止的路径访问尝试

### 架构演进

Phase 5 证明了 WriteFlow-SDK 在以下方面与 DeepAgents 对齐或超越:

- ✅ **安全性**: 完整的路径验证机制
- ✅ **灵活性**: 配置驱动的工具定制
- ✅ **可用性**: 详细的中文提示词
- ✅ **设计**: 无状态 bash 执行(优于 DeepAgents)

---

## 参考资料

- DeepAgents 项目: `/Users/coso/Documents/dev/python/deepagents`
- DeepAgents Filesystem: `deepagents/middleware/filesystem.py`
- DeepAgents SubAgent: `deepagents/middleware/subagents.py`
- Go filepath 文档: https://pkg.go.dev/path/filepath

---

**Phase 5 完成时间**: 2025-11-09
**总代码变更**: ~500 行(含测试)
**测试覆盖**: 20+ 测试用例
**性能影响**: < 200 ns/op
**向后兼容**: 100% (默认禁用新功能)
