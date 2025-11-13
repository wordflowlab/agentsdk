---
title: 切换 Provider
description: 在不同 AI Provider 之间轻松切换
---

# 切换 Provider 示例

展示如何在不同的 AI Provider 之间快速切换。

## 完整代码

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/wordflowlab/agentsdk/pkg/provider"
    "github.com/wordflowlab/agentsdk/pkg/types"
)

func main() {
    ctx := context.Background()
    factory := provider.NewMultiProviderFactory()

    // 测试不同的 Providers
    providers := []struct {
        name     string
        config   *types.ModelConfig
    }{
        {
            name: "OpenAI GPT-4o",
            config: &types.ModelConfig{
                Provider: "openai",
                Model:    "gpt-4o",
                APIKey:   os.Getenv("OPENAI_API_KEY"),
            },
        },
        {
            name: "Anthropic Claude",
            config: &types.ModelConfig{
                Provider: "anthropic",
                Model:    "claude-sonnet-4-5",
                APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
            },
        },
        {
            name: "DeepSeek Chat",
            config: &types.ModelConfig{
                Provider: "deepseek",
                Model:    "deepseek-chat",
                APIKey:   os.Getenv("DEEPSEEK_API_KEY"),
            },
        },
        {
            name: "Groq Llama",
            config: &types.ModelConfig{
                Provider: "groq",
                Model:    "llama-3.3-70b-versatile",
                APIKey:   os.Getenv("GROQ_API_KEY"),
            },
        },
        {
            name: "Ollama (本地)",
            config: &types.ModelConfig{
                Provider: "ollama",
                Model:    "llama3.2",
                BaseURL:  "http://localhost:11434/v1",
            },
        },
    }

    prompt := "用一句话介绍你自己"

    // 测试每个 Provider
    for _, p := range providers {
        fmt.Printf("\n========== %s ==========\n", p.name)

        prov, err := factory.Create(p.config)
        if err != nil {
            log.Printf("创建失败: %v", err)
            continue
        }
        defer prov.Close()

        // 发送相同的请求
        messages := []types.Message{
            {Role: types.RoleUser, Content: prompt},
        }

        response, err := prov.Complete(ctx, messages, &provider.StreamOptions{
            MaxTokens: 200,
        })
        if err != nil {
            log.Printf("请求失败: %v", err)
            continue
        }

        // 打印响应
        fmt.Printf("回复: %s\n", response.Message.Content)
        fmt.Printf("Token: %d\n", response.Usage.TotalTokens)
    }
}
```

## 运行示例

```bash
# 设置 API Keys
export OPENAI_API_KEY="sk-xxx"
export ANTHROPIC_API_KEY="sk-ant-xxx"
export DEEPSEEK_API_KEY="sk-xxx"
export GROQ_API_KEY="gsk-xxx"

# 启动 Ollama（如果需要本地测试）
ollama serve

# 运行
go run main.go
```

## 输出示例

```
========== OpenAI GPT-4o ==========
回复: 我是 GPT-4o，OpenAI 开发的大型语言模型，可以帮助你解答问题、创作内容和编写代码。
Token: 89

========== Anthropic Claude ==========
回复: 我是 Claude，由 Anthropic 开发的 AI 助手，擅长对话、分析和协助各类任务。
Token: 76

========== DeepSeek Chat ==========
回复: 我是 DeepSeek，一个由深度求索研发的智能对话助手。
Token: 52

========== Groq Llama ==========
回复: 我是基于 Llama 架构的开源语言模型，运行在 Groq 的高性能硬件上。
Token: 68

========== Ollama (本地) ==========
回复: 我是在你本地运行的 Llama 3.2 模型，无需联网即可使用。
Token: 45
```

## 在 Agent 中切换

```go
// 方法 1: 创建时指定
ag, err := agent.Create(ctx, &types.AgentConfig{
    TemplateID: "assistant",
    ModelConfig: &types.ModelConfig{
        Provider: "openai",  // 切换到 OpenAI
        Model:    "gpt-4o",
        APIKey:   os.Getenv("OPENAI_API_KEY"),
    },
}, deps)

// 方法 2: 使用环境变量动态选择
providerName := os.Getenv("AI_PROVIDER") // "openai", "anthropic", "deepseek"
modelName := os.Getenv("AI_MODEL")

ag, err := agent.Create(ctx, &types.AgentConfig{
    TemplateID: "assistant",
    ModelConfig: &types.ModelConfig{
        Provider: providerName,
        Model:    modelName,
        APIKey:   os.Getenv(strings.ToUpper(providerName) + "_API_KEY"),
    },
}, deps)
```

## Provider 特点对比

### 速度优先 - Groq

```go
config := &types.ModelConfig{
    Provider: "groq",
    Model:    "llama-3.3-70b-versatile",  // 业界最快
    APIKey:   os.Getenv("GROQ_API_KEY"),
}
```

### 质量优先 - Claude/GPT-4

```go
// Claude Sonnet 4.5
config := &types.ModelConfig{
    Provider: "anthropic",
    Model:    "claude-sonnet-4-5",
    APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
}

// GPT-4o
config := &types.ModelConfig{
    Provider: "openai",
    Model:    "gpt-4o",
    APIKey:   os.Getenv("OPENAI_API_KEY"),
}
```

### 成本优先 - DeepSeek/Ollama

```go
// DeepSeek（超低价格）
config := &types.ModelConfig{
    Provider: "deepseek",
    Model:    "deepseek-chat",
    APIKey:   os.Getenv("DEEPSEEK_API_KEY"),
}

// Ollama（完全免费，本地运行）
config := &types.ModelConfig{
    Provider: "ollama",
    Model:    "llama3.2",
    BaseURL:  "http://localhost:11434/v1",
}
```

### 长上下文 - Gemini

```go
config := &types.ModelConfig{
    Provider: "gemini",
    Model:    "gemini-1.5-pro",  // 2M tokens 上下文
    APIKey:   os.Getenv("GEMINI_API_KEY"),
}
```

## 故障转移

自动切换到备用 Provider：

```go
func createProviderWithFallback(ctx context.Context) (provider.Provider, error) {
    factory := provider.NewMultiProviderFactory()

    // 主 Provider
    primary := &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-sonnet-4-5",
        APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
    }

    p, err := factory.Create(primary)
    if err == nil {
        return p, nil
    }

    log.Printf("主 Provider 失败，切换到备用: %v", err)

    // 备用 Provider
    fallback := &types.ModelConfig{
        Provider: "groq",
        Model:    "llama-3.3-70b-versatile",
        APIKey:   os.Getenv("GROQ_API_KEY"),
    }

    return factory.Create(fallback)
}
```

## 相关资源

- [Provider 总览](../../providers/overview)
- [Provider API 文档](../../api-reference/provider-api)
- [OpenAI Provider](../../providers/openai)
- [Anthropic Provider](../../providers/anthropic)
- [Gemini Provider](../../providers/gemini)
