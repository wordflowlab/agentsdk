---
title: Skills API
description: Skills 系统完整 API 参考文档
---

# Skills API 参考

本文档提供 Skills 系统的完整 API 参考，包括 Skill 定义、加载器、注入器等核心组件。

## 目录

- [Skill 定义](#skill-定义)
- [SkillLoader 加载器](#skillloader-加载器)
- [Injector 注入器](#injector-注入器)
- [配置类型](#配置类型)
- [触发器类型](#触发器类型)

---

## Skill 定义

### types.SkillDefinition

Skill 的完整定义结构。

```go
type SkillDefinition struct {
    // 元数据
    Name        string   `yaml:"name"`        // Skill 唯一标识符
    Description string   `yaml:"description"` // Skill 功能描述
    AllowedTools []string `yaml:"allowed-tools,omitempty"` // 允许使用的工具列表

    // 触发配置
    Triggers []Trigger `yaml:"triggers"` // 触发条件列表

    // 知识库内容
    Content string `yaml:"-"` // Markdown 内容（不包含 frontmatter）
}
```

**字段说明**：

- `Name`: Skill 的唯一标识符，必须符合命名规范（字母、数字、连字符）
- `Description`: 简短描述 Skill 的功能和用途
- `AllowedTools`: 可选字段，限制 Skill 可以使用的工具
- `Triggers`: 触发条件数组，至少需要一个触发器
- `Content`: Skill 的知识库内容，使用 Markdown 格式

**示例**：

```go
skill := &types.SkillDefinition{
    Name:        "consistency-checker",
    Description: "检查写作内容的一致性",
    AllowedTools: []string{"Read", "Grep"},
    Triggers: []types.Trigger{
        {
            Type:     types.TriggerTypeKeyword,
            Keywords: []string{"一致性", "检查"},
        },
    },
    Content: "# 一致性检查指南\n\n...",
}
```

---

## SkillLoader 加载器

### skills.NewLoader

创建 Skill 加载器实例。

```go
func NewLoader(config *types.SkillsPackageConfig) (*SkillLoader, error)
```

**参数**：
- `config`: Skills Package 配置，指定加载路径和选项

**返回**：
- `*SkillLoader`: 加载器实例
- `error`: 创建失败时返回错误

**示例**：

```go
loader, err := skills.NewLoader(&types.SkillsPackageConfig{
    Path: "./workspace/.claude/skills",
})
if err != nil {
    log.Fatal(err)
}
```

### loader.LoadAll

从指定路径加载所有 Skills。

```go
func (l *SkillLoader) LoadAll(ctx context.Context) ([]*types.SkillDefinition, error)
```

**参数**：
- `ctx`: 上下文，用于控制加载超时

**返回**：
- `[]*types.SkillDefinition`: 加载的 Skill 列表
- `error`: 加载失败时返回错误

**示例**：

```go
skills, err := loader.LoadAll(ctx)
if err != nil {
    log.Printf("加载 Skills 失败: %v", err)
    return
}

fmt.Printf("成功加载 %d 个 Skills\n", len(skills))
for _, skill := range skills {
    fmt.Printf("- %s: %s\n", skill.Name, skill.Description)
}
```

### loader.LoadByName

根据名称加载单个 Skill。

```go
func (l *SkillLoader) LoadByName(ctx context.Context, name string) (*types.SkillDefinition, error)
```

**参数**：
- `ctx`: 上下文
- `name`: Skill 名称（不包含 .md 扩展名）

**返回**：
- `*types.SkillDefinition`: 加载的 Skill
- `error`: 加载失败时返回错误

**示例**：

```go
skill, err := loader.LoadByName(ctx, "consistency-checker")
if err != nil {
    log.Printf("加载 Skill 失败: %v", err)
    return
}
```

### loader.Parse

解析 Skill Markdown 文件内容。

```go
func (l *SkillLoader) Parse(content []byte) (*types.SkillDefinition, error)
```

**参数**：
- `content`: Skill Markdown 文件的原始内容

**返回**：
- `*types.SkillDefinition`: 解析后的 Skill 定义
- `error`: 解析失败时返回错误

**示例**：

```go
content, err := os.ReadFile("consistency-checker.md")
if err != nil {
    log.Fatal(err)
}

skill, err := loader.Parse(content)
if err != nil {
    log.Printf("解析失败: %v", err)
    return
}
```

---

## Injector 注入器

### skills.NewInjector

创建 Skill 注入器实例。

```go
func NewInjector(loader *SkillLoader) *Injector
```

**参数**：
- `loader`: Skill 加载器实例

**返回**：
- `*Injector`: 注入器实例

**示例**：

```go
loader, _ := skills.NewLoader(config)
injector := skills.NewInjector(loader)
```

### injector.ActivateSkills

根据触发条件激活相关 Skills。

```go
func (inj *Injector) ActivateSkills(
    ctx context.Context,
    userMessage string,
    currentContext *types.ExecutionContext,
) ([]*types.SkillDefinition, error)
```

**参数**：
- `ctx`: 上下文
- `userMessage`: 用户消息内容
- `currentContext`: 当前执行上下文（包含命令、文件路径等）

**返回**：
- `[]*types.SkillDefinition`: 激活的 Skill 列表
- `error`: 错误信息

**示例**：

```go
activated, err := injector.ActivateSkills(ctx, "帮我检查一下一致性", &types.ExecutionContext{
    Command: "/write",
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("激活了 %d 个 Skills\n", len(activated))
```

### injector.InjectToSystemPrompt

将激活的 Skills 注入到系统提示词。

```go
func (inj *Injector) InjectToSystemPrompt(
    basePrompt string,
    activatedSkills []*types.SkillDefinition,
) string
```

**参数**：
- `basePrompt`: 基础系统提示词
- `activatedSkills`: 激活的 Skill 列表

**返回**：
- `string`: 注入后的系统提示词

**示例**：

```go
activated, _ := injector.ActivateSkills(ctx, userMsg, execCtx)

enhancedPrompt := injector.InjectToSystemPrompt(
    "You are a helpful AI assistant.",
    activated,
)

// enhancedPrompt 现在包含:
// You are a helpful AI assistant.
//
// ## Activated Skills
//
// ### consistency-checker
// [Skill 内容]
```

### injector.InjectToUserMessage

将激活的 Skills 注入到用户消息前缀。

```go
func (inj *Injector) InjectToUserMessage(
    userMessage string,
    activatedSkills []*types.SkillDefinition,
) string
```

**参数**：
- `userMessage`: 原始用户消息
- `activatedSkills`: 激活的 Skill 列表

**返回**：
- `string`: 注入后的用户消息

**示例**：

```go
activated, _ := injector.ActivateSkills(ctx, userMsg, execCtx)

enhancedMessage := injector.InjectToUserMessage(
    "帮我检查一致性",
    activated,
)

// enhancedMessage 现在包含:
// ## Knowledge Base
//
// ### consistency-checker
// [Skill 内容]
//
// ---
//
// 帮我检查一致性
```

---

## 配置类型

### types.SkillsPackageConfig

Skills Package 配置。

```go
type SkillsPackageConfig struct {
    // Skills 包路径
    // 支持本地文件系统、OSS、S3、HTTP(S)
    Path string `json:"path" yaml:"path"`

    // 可选：缓存配置
    CacheEnabled bool          `json:"cache_enabled,omitempty" yaml:"cache_enabled,omitempty"`
    CacheTTL     time.Duration `json:"cache_ttl,omitempty" yaml:"cache_ttl,omitempty"`

    // 可选：OSS/S3 认证信息
    Credentials *StorageCredentials `json:"credentials,omitempty" yaml:"credentials,omitempty"`
}
```

**字段说明**：

- `Path`: Skills 包的加载路径
  - 本地路径: `./workspace/.claude/skills`
  - OSS: `oss://bucket-name/skills/`
  - S3: `s3://bucket-name/skills/`
  - HTTP(S): `https://cdn.example.com/skills/`

- `CacheEnabled`: 是否启用 Skill 缓存（默认 false）
- `CacheTTL`: 缓存过期时间（默认 0，永不过期）
- `Credentials`: 云存储认证信息（可选）

**示例**：

```go
// 本地文件系统
config := &types.SkillsPackageConfig{
    Path: "./workspace/.claude/skills",
}

// OSS 存储
config := &types.SkillsPackageConfig{
    Path: "oss://my-bucket/skills/",
    Credentials: &types.StorageCredentials{
        AccessKeyID:     os.Getenv("OSS_ACCESS_KEY_ID"),
        AccessKeySecret: os.Getenv("OSS_ACCESS_KEY_SECRET"),
        Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
    },
    CacheEnabled: true,
    CacheTTL:     30 * time.Minute,
}
```

---

## 触发器类型

### types.Trigger

触发条件定义。

```go
type Trigger struct {
    Type TriggerType `yaml:"type"` // 触发类型

    // 关键词触发专用
    Keywords []string `yaml:"keywords,omitempty"`

    // 上下文触发专用
    Condition string `yaml:"condition,omitempty"`

    // 文件模式触发专用
    Pattern string `yaml:"pattern,omitempty"`
}
```

### types.TriggerType

触发类型枚举。

```go
type TriggerType string

const (
    TriggerTypeKeyword     TriggerType = "keyword"      // 关键词触发
    TriggerTypeContext     TriggerType = "context"      // 上下文触发
    TriggerTypeAlways      TriggerType = "always"       // 总是激活
    TriggerTypeFilePattern TriggerType = "file_pattern" // 文件模式触发
)
```

**触发类型说明**：

| 类型 | 说明 | 配置字段 | 示例 |
|------|------|---------|------|
| `keyword` | 用户消息包含关键词时触发 | `keywords` | `keywords: ["测试", "test"]` |
| `context` | 特定上下文条件满足时触发 | `condition` | `condition: "during /write"` |
| `always` | 无条件总是激活 | 无 | `type: always` |
| `file_pattern` | 操作的文件路径匹配模式时触发 | `pattern` | `pattern: "**/*.go"` |

**示例**：

```yaml
# 关键词触发
triggers:
  - type: keyword
    keywords: ["一致性", "检查", "consistency"]

# 上下文触发
triggers:
  - type: context
    condition: "during /write"

# 总是激活
triggers:
  - type: always

# 文件模式触发
triggers:
  - type: file_pattern
    pattern: "src/**/*.go"

# 组合触发
triggers:
  - type: keyword
    keywords: ["审查", "review"]
  - type: file_pattern
    pattern: "**/*.go"
  - type: context
    condition: "during /review"
```

---

## 完整使用示例

### 基础用法

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/wordflowlab/agentsdk/pkg/agent"
    "github.com/wordflowlab/agentsdk/pkg/skills"
    "github.com/wordflowlab/agentsdk/pkg/types"
)

func main() {
    ctx := context.Background()

    // 1. 创建 Skill 加载器
    loader, err := skills.NewLoader(&types.SkillsPackageConfig{
        Path: "./workspace/.claude/skills",
    })
    if err != nil {
        log.Fatal(err)
    }

    // 2. 加载所有 Skills
    allSkills, err := loader.LoadAll(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("加载了 %d 个 Skills\n", len(allSkills))

    // 3. 创建 Agent 并启用 Skills
    ag, err := agent.Create(ctx, &types.AgentConfig{
        TemplateID: "assistant",
        ModelConfig: &types.ModelConfig{
            Provider: "anthropic",
            Model:    "claude-sonnet-4-5",
            APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
        },
        SkillsPackageConfig: &types.SkillsPackageConfig{
            Path: "./workspace/.claude/skills",
        },
    }, deps)
    if err != nil {
        log.Fatal(err)
    }
    defer ag.Close()

    // 4. 使用 Agent（Skills 会自动激活）
    result, err := ag.Chat(ctx, "帮我检查一下代码一致性")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Text)
}
```

### 手动控制 Skill 激活

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/wordflowlab/agentsdk/pkg/skills"
    "github.com/wordflowlab/agentsdk/pkg/types"
)

func main() {
    ctx := context.Background()

    // 1. 创建加载器和注入器
    loader, _ := skills.NewLoader(&types.SkillsPackageConfig{
        Path: "./workspace/.claude/skills",
    })
    injector := skills.NewInjector(loader)

    // 2. 手动激活 Skills
    userMsg := "帮我检查一致性"
    execCtx := &types.ExecutionContext{
        Command: "/write",
    }

    activated, err := injector.ActivateSkills(ctx, userMsg, execCtx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("激活了 %d 个 Skills:\n", len(activated))
    for _, skill := range activated {
        fmt.Printf("- %s: %s\n", skill.Name, skill.Description)
    }

    // 3. 注入到提示词
    basePrompt := "You are a helpful AI assistant."
    enhancedPrompt := injector.InjectToSystemPrompt(basePrompt, activated)

    fmt.Println("\n增强后的提示词:")
    fmt.Println(enhancedPrompt)
}
```

### 动态加载 Skill

```go
package main

import (
    "context"
    "os"
    "log"

    "github.com/wordflowlab/agentsdk/pkg/skills"
    "github.com/wordflowlab/agentsdk/pkg/types"
)

func main() {
    ctx := context.Background()

    loader, _ := skills.NewLoader(&types.SkillsPackageConfig{
        Path: "./workspace/.claude/skills",
    })

    // 1. 加载特定 Skill
    skill, err := loader.LoadByName(ctx, "consistency-checker")
    if err != nil {
        log.Fatal(err)
    }

    // 2. 或从文件内容解析
    content, _ := os.ReadFile("custom-skill.md")
    customSkill, err := loader.Parse(content)
    if err != nil {
        log.Fatal(err)
    }

    // 3. 检查 Skill 元数据
    log.Printf("Skill: %s", skill.Name)
    log.Printf("描述: %s", skill.Description)
    log.Printf("触发器数量: %d", len(skill.Triggers))
    log.Printf("允许的工具: %v", skill.AllowedTools)
}
```

---

## 错误处理

### 常见错误

| 错误 | 原因 | 解决方案 |
|------|------|---------|
| `skill not found` | Skill 文件不存在 | 检查文件路径和名称 |
| `invalid yaml frontmatter` | YAML 格式错误 | 验证 YAML 语法 |
| `missing required field: name` | 缺少必需字段 | 添加 name 字段 |
| `no triggers defined` | 没有定义触发器 | 至少添加一个触发器 |
| `invalid trigger type` | 触发器类型不支持 | 使用支持的类型 |

### 错误处理示例

```go
loader, err := skills.NewLoader(config)
if err != nil {
    switch {
    case errors.Is(err, skills.ErrInvalidPath):
        log.Fatal("Skills 路径无效")
    case errors.Is(err, skills.ErrAccessDenied):
        log.Fatal("没有访问权限")
    default:
        log.Fatalf("创建加载器失败: %v", err)
    }
}

skills, err := loader.LoadAll(ctx)
if err != nil {
    // 部分加载失败，记录日志但继续
    log.Printf("警告: 部分 Skills 加载失败: %v", err)
}
```

---

## 性能优化

### 缓存策略

```go
config := &types.SkillsPackageConfig{
    Path: "oss://my-bucket/skills/",
    CacheEnabled: true,
    CacheTTL:     30 * time.Minute, // 30分钟缓存
}
```

### 延迟加载

```go
// 不要一次性加载所有 Skills
// loader.LoadAll(ctx)

// 而是按需加载
skill, err := loader.LoadByName(ctx, "needed-skill")
```

### 预加载常用 Skills

```go
// 在应用启动时预加载
commonSkills := []string{
    "coding-standards",
    "security-checklist",
}

for _, name := range commonSkills {
    _, err := loader.LoadByName(ctx, name)
    if err != nil {
        log.Printf("预加载 %s 失败: %v", name, err)
    }
}
```

---

## 相关资源

- [Skills 核心概念](/core-concepts/skills-system) - Skills 系统设计和原理
- [自定义工具文档](/tools/builtin/custom) - 完整使用指南
- [Skills 示例](/examples/skills) - 实际应用案例
- [最佳实践](/best-practices/skills) - 高级技巧

---

**包路径**: `github.com/wordflowlab/agentsdk/pkg/skills`
**版本**: v0.4.0+
