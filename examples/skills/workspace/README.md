# Skills 包示例工作区

这是一个完整的 Skills 包示例，展示了如何组织命令（Commands）和技能（Skills）。

## 目录结构

```
workspace/
├── commands/                 # Slash Commands 定义
│   ├── write.md             # /write 命令 - 章节写作
│   ├── analyze.md           # /analyze 命令 - 质量分析
│   └── ...                  # 可以添加更多命令
│
├── skills/                  # Agent Skills 定义
│   ├── consistency-checker/ # 一致性检查技能
│   │   └── SKILL.md
│   ├── pdfmd/               # PDF → Markdown 转换技能示例
│   │   ├── SKILL.md
│   │   └── pdf_extract.py
│   └── ...                  # 可以添加更多技能
│
└── README.md               # 本文件
```

## 使用方式

### 1. 在 Agent 配置中引用

```go
agent, err := agent.Create(ctx, &types.AgentConfig{
    ModelConfig: &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-sonnet-4",
        APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
    },
    Sandbox: &types.SandboxConfig{
        Kind:    types.SandboxKindLocal,
        WorkDir: "./workspace",  // 指向此目录
    },
    SkillsPackage: &types.SkillsPackageConfig{
        Source:      "local",
        Path:        ".",        // 在 WorkDir 下
        CommandsDir: "commands", // 相对路径
        SkillsDir:   "skills",   // 相对路径
        EnabledCommands: []string{
            "write",
            "analyze",
        },
        EnabledSkills: []string{
            "consistency-checker",
        },
    },
}, deps)
```

### 2. 使用 Slash Commands

```go
// 执行写作命令
agent.Send(ctx, "/write 第1章")

// 执行分析命令
agent.Send(ctx, "/analyze 全部")
```

### 3. 自动激活 Skills

```go
// 普通对话，包含关键词 "一致性"
agent.Send(ctx, "帮我检查第1章的一致性问题")
// → consistency-checker skill 会自动激活

// 在执行 /write 时
agent.Send(ctx, "/write 第2章")
// → consistency-checker skill 也会自动激活（后台监控）
```

## 命令文件格式

每个命令文件（`.md`）包含两部分：

### YAML Frontmatter

```yaml
---
description: 命令描述
argument-hint: [参数提示]
allowed-tools: ["Read", "Write", "Bash"]
models:
  preferred:
    - claude-sonnet-4-5
    - gpt-4-turbo
  minimum-capabilities:
    - tool-calling
scripts:
  sh: scripts/bash/script.sh
  ps: scripts/powershell/script.ps1
---
```

### Markdown 内容

命令的详细执行流程和提示词模板。

## Skills 文件格式

每个技能文件（`SKILL.md`）也包含两部分：

### YAML Frontmatter

```yaml
---
name: skill-name
description: 技能描述
allowed-tools: ["Read", "Grep"]
triggers:
  - type: keyword
    keywords: ["关键词1", "关键词2"]
  - type: context
    condition: "during /write"
  - type: always
---
```

### Markdown 内容

技能的知识库内容，会注入到 SystemPrompt。

### 示例：pdfmd（PDF → Markdown）

本仓库内置了一个简单的 PDF 转 Markdown 示例 Skill：

- 目录：`skills/pdfmd/`
- 工具依赖：`Bash`、`Write`
- 脚本：`skills/pdfmd/pdf_extract.py`（只负责从 PDF 中提取纯文本）

在 Agent 配置中启用该技能示例：

```go
SkillsPackage: &types.SkillsPackageConfig{
    Source:      "local",
    Path:        ".",        // 在 WorkDir 下
    CommandsDir: "commands",
    SkillsDir:   "skills",
    EnabledCommands: []string{"write", "analyze"},
    EnabledSkills: []string{
        "consistency-checker",
        "pdfmd", // PDF → Markdown 示例 Skill
    },
},
```

然后，你可以直接在对话中说：

> 请把 docs/report.pdf 转成中文 Markdown，并保存为 docs/report.md

Agent 会按照 `skills/pdfmd/SKILL.md` 中的说明：

1. 使用 `Bash` 执行 `python workspace/skills/pdfmd/pdf_extract.py --input docs/report.pdf`
2. 将脚本输出的纯文本放入当前对话上下文
3. 用当前配置的大模型翻译并整理为 Markdown
4. 使用 `Write` 写入 `docs/report.md`（如果用户有这个需求）

## 触发器类型

### keyword 触发器

当用户消息包含指定关键词时激活：

```yaml
triggers:
  - type: keyword
    keywords: ["一致性", "检查", "consistency"]
```

### context 触发器

在特定上下文中激活：

```yaml
triggers:
  - type: context
    condition: "during /write"  # 在执行 /write 命令时
```

### always 触发器

始终激活（谨慎使用）：

```yaml
triggers:
  - type: always
```

## 多模型支持

所有命令和技能都是模型无关的，可以用于：

- ✅ Claude (Anthropic)
- ✅ GPT-4 (OpenAI)
- ✅ 通义千问 (Qwen)
- ✅ Gemini (Google)
- ✅ 其他支持的模型

SDK 会自动检测模型能力并适配：
- 支持 SystemPrompt 的模型 → Skills 注入到 SystemPrompt
- 不支持的模型 → Skills 信息添加到 UserMessage

## 扩展

### 添加新命令

1. 在 `commands/` 目录创建新的 `.md` 文件
2. 按照格式填写 YAML frontmatter 和内容
3. 在 `EnabledCommands` 中添加命令名

### 添加新技能

1. 在 `skills/` 目录创建新的子目录
2. 在子目录中创建 `SKILL.md` 文件
3. 按照格式填写 YAML frontmatter 和知识库内容
4. 在 `EnabledSkills` 中添加技能路径

## 最佳实践

1. **命令设计**：
   - 提供清晰的描述和参数提示
   - 指定所需的工具和模型能力
   - 编写详细的执行流程

2. **技能设计**：
   - 合理设置触发条件，避免过度激活
   - 提供有价值的知识库内容
   - 保持内容聚焦和相关

3. **组织结构**：
   - 按功能分类命令
   - 按领域分类技能
   - 保持文件命名一致

4. **内容语言**：
   - 使用清晰、简洁的中文
   - 提供具体的示例
   - 包含必要的说明

## 参考

- [Commands 系统文档](../../../pkg/commands/)
- [Skills 系统文档](../../../pkg/skills/)
- [使用示例](../main.go)
- [完整 README](../README.md)
