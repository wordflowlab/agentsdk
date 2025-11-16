package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/wordflowlab/agentsdk/pkg/agent"
	"github.com/wordflowlab/agentsdk/pkg/provider"
	"github.com/wordflowlab/agentsdk/pkg/sandbox"
	"github.com/wordflowlab/agentsdk/pkg/store"
	"github.com/wordflowlab/agentsdk/pkg/tools"
	"github.com/wordflowlab/agentsdk/pkg/tools/builtin"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

func main() {
	var (
		message   = flag.String("message", "", "要发送给Agent的消息")
		workspace = flag.String("workspace", "./workspace", "工作目录路径")
		debug     = flag.Bool("debug", false, "启用调试模式")
	)
	flag.Parse()

	fmt.Println("=== Agent Skills 调试模式 ===\n")
	if *debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	ctx := context.Background()

	// 创建带有 Skills 支持的 Agent
	deps := createDependencies()

	agentConfig := &types.AgentConfig{
		TemplateID: "assistant",
		ModelConfig: &types.ModelConfig{
			Provider:      "deepseek",
			Model:         "deepseek-chat",
			APIKey:        os.Getenv("DEEPSEEK_API_KEY"),
			ExecutionMode: types.ExecutionModeNonStreaming, // 🚀 非流式快速模式（小segment不会超时）
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: *workspace,
		},
		SkillsPackage: &types.SkillsPackageConfig{
			Source:      "local",
			Path:        ".", // 相对于 Sandbox.WorkDir，即 ./workspace
			CommandsDir: "commands",
			SkillsDir:   "skills",
			EnabledCommands: []string{
				"write",
				"analyze",
				"plan",
			},
			EnabledSkills: []string{
				"consistency-checker",
				"pdfmd",
				"pdf",
				"markdown-segment-translator",
			},
		},
	}

	ag, err := agent.Create(ctx, agentConfig, deps)
	if err != nil {
		log.Fatalf("创建 Agent 失败: %v", err)
	}
	defer ag.Close()

	fmt.Printf("Agent 创建成功: %s\n\n", ag.ID())

	// 使用命令行参数消息或默认PDF触发消息
	targetMessage := *message
	if targetMessage == "" {
		targetMessage = "请pdf处理2407.14333v5.pdf文档，提取其内容并转换为markdown格式"
	}

	fmt.Printf("--- 发送消息 ---\n")
	fmt.Printf("消息内容: %s\n\n", targetMessage)

	result, err := ag.Chat(ctx, targetMessage)
	if err != nil {
		log.Printf("AI 处理失败: %v", err)
	} else {
		fmt.Printf("AI 处理成功！\n")
		if result != nil && result.Text != "" {
			fmt.Printf("AI 响应: %s\n", result.Text)
		}
	}

	fmt.Println("\n=== 所有示例完成 ===")
}

// createDependencies 创建依赖（简化版本）
func createDependencies() *agent.Dependencies {
	// 创建基本的依赖项
	store, _ := store.NewJSONStore(".agentsdk-store")
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// 注册基本模板
	templateRegistry := agent.NewTemplateRegistry()
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:    "assistant",
		Model: "deepseek-chat",
		SystemPrompt: "⚠️ CRITICAL RULES ⚠️\n" +
			"1. When a skill document contains EXPLICIT instructions to use bash_run tool to execute Python scripts, you MUST follow those instructions EXACTLY.\n" +
			"2. DO NOT attempt to translate documents yourself - use bash_run to execute the translation script instead.\n" +
			"3. If skill instructions say 'use bash_run', then your FIRST tool call must be bash_run, not fs_read or fs_write.\n\n" +
			"🚀 EFFICIENCY RULES (IMPORTANT) 🚀\n" +
			"- Execute tasks as DIRECTLY as possible with MINIMAL steps\n" +
			"- When you know what to do, DO IT IMMEDIATELY without explaining first\n" +
			"- For simple tasks (read→process→write), complete them in ONE response\n" +
			"- AVOID unnecessary intermediate steps or confirmations\n" +
			"- Example: If asked to 'translate file A to B', do: fs_read→translate→fs_write in ONE go\n\n" +
			"You are a helpful assistant with access to filesystem and memory tools. " +
			"Use tools when appropriate to read/write files or manage long-term memory.",
		Tools: []interface{}{"fs_read", "fs_write", "bash_run"},
	})

	return &agent.Dependencies{
		Store:            store,
		ToolRegistry:     toolRegistry,
		SandboxFactory:   sandbox.NewFactory(),
		ProviderFactory:  provider.NewMultiProviderFactory(),
		TemplateRegistry: templateRegistry,
	}
}
