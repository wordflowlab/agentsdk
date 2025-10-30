package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/wordflowlab/agentsdk/pkg/agent"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

func main() {
	fmt.Println("=== Slash Commands & Agent Skills 示例 ===\n")

	ctx := context.Background()

	// 创建带有 Skills 支持的 Agent
	deps := createDependencies()

	agentConfig := &types.AgentConfig{
		TemplateID: "novel-writer",
		ModelConfig: &types.ModelConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4",
			APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: "./workspace",
		},
		SkillsPackage: &types.SkillsPackageConfig{
			Source:      "local",
			Path:        "./skills-package",
			Version:     "v1.0.0",
			CommandsDir: "commands",
			SkillsDir:   "skills",
			EnabledCommands: []string{
				"write",
				"analyze",
				"plan",
			},
			EnabledSkills: []string{
				"consistency-checker",
				"workflow-guide",
			},
		},
	}

	ag, err := agent.Create(ctx, agentConfig, deps)
	if err != nil {
		log.Fatalf("创建 Agent 失败: %v", err)
	}
	defer ag.Close()

	fmt.Printf("Agent 创建成功: %s\n\n", ag.ID())

	// 示例 1: 使用 Slash Command
	fmt.Println("--- 示例 1: 使用 Slash Command ---")
	err = ag.Send(ctx, "/write 第1章")
	if err != nil {
		log.Printf("执行命令失败: %v", err)
	} else {
		fmt.Println("命令已发送，等待 AI 处理...")
	}

	// 等待一段时间
	fmt.Println()

	// 示例 2: 普通对话（自动激活 Skills）
	fmt.Println("--- 示例 2: 普通对话（自动激活 Skills）---")
	err = ag.Send(ctx, "帮我检查第1章的一致性问题")
	if err != nil {
		log.Printf("发送消息失败: %v", err)
	} else {
		fmt.Println("消息已发送，consistency-checker skill 将自动激活...")
	}

	fmt.Println()

	// 示例 3: 使用不同模型
	fmt.Println("--- 示例 3: 使用 GPT-4 ---")

	gptConfig := &types.AgentConfig{
		TemplateID: "novel-writer",
		ModelConfig: &types.ModelConfig{
			Provider: "openai",
			Model:    "gpt-4-turbo",
			APIKey:   os.Getenv("OPENAI_API_KEY"),
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: "./workspace",
		},
		SkillsPackage: &types.SkillsPackageConfig{
			Source:          "local",
			Path:            "./skills-package",
			EnabledCommands: []string{"write", "analyze"},
			EnabledSkills:   []string{"consistency-checker"},
		},
	}

	gptAgent, err := agent.Create(ctx, gptConfig, deps)
	if err != nil {
		log.Fatalf("创建 GPT Agent 失败: %v", err)
	}
	defer gptAgent.Close()

	fmt.Printf("GPT Agent 创建成功: %s\n", gptAgent.ID())
	err = gptAgent.Send(ctx, "/write 第1章")
	if err != nil {
		log.Printf("执行命令失败: %v", err)
	} else {
		fmt.Println("GPT Agent 正在处理...")
	}

	fmt.Println("\n=== 所有示例完成 ===")
}

// createDependencies 创建依赖（简化版本）
func createDependencies() *agent.Dependencies {
	// 这里应该创建实际的依赖
	// 为了示例简洁，返回 nil
	// 实际使用时需要提供完整的依赖
	return nil
}
