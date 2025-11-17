package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/wordflowlab/agentsdk/pkg/agent"
	"github.com/wordflowlab/agentsdk/pkg/backends"
	"github.com/wordflowlab/agentsdk/pkg/middleware"
	"github.com/wordflowlab/agentsdk/pkg/provider"
	"github.com/wordflowlab/agentsdk/pkg/tools"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

func main() {
	ctx := context.Background()

	// 1. 创建 HITL 中间件
	hitlMW, err := createHITLMiddleware()
	if err != nil {
		log.Fatalf("Failed to create HITL middleware: %v", err)
	}

	// 2. 创建文件系统中间件（提供文件操作工具）
	backend := backends.NewStateBackend()
	filesMW := middleware.NewFilesystemMiddleware(&middleware.FilesystemMiddlewareConfig{
		Backend:    backend,
		TokenLimit: 20000,
	})

	// 3. 注册中间件
	stack := middleware.NewStack()
	stack.Use(hitlMW)
	stack.Use(filesMW)

	// 4. 创建 LLM Provider
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	llm, err := provider.NewOpenAIProvider(&provider.OpenAIProviderConfig{
		APIKey: apiKey,
		Model:  "gpt-4",
	})
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// 5. 创建 Agent 配置
	config := &agent.Config{
		Name:         "HITL-Demo-Agent",
		Description:  "演示 Human-in-the-Loop 功能的 Agent",
		SystemPrompt: buildSystemPrompt(),
		Tools:        []tools.Tool{
			// 添加一个危险的 shell 工具用于演示
			&tools.ShellTool{
				Name:        "Bash",
				Description: "Execute shell commands",
			},
		},
	}

	// 6. 创建 Agent
	ag, err := agent.Create(ctx, config, &agent.Dependencies{
		Provider:        llm,
		Backend:         backend,
		MiddlewareStack: stack,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer ag.Close()

	// 7. 运行演示场景
	runDemo(ctx, ag)
}

func createHITLMiddleware() (*middleware.HumanInTheLoopMiddleware, error) {
	return middleware.NewHumanInTheLoopMiddleware(&middleware.HumanInTheLoopMiddlewareConfig{
		// 配置需要审核的工具
		InterruptOn: map[string]interface{}{
			"Bash": map[string]interface{}{
				"message":           "⚠️  Shell 命令执行需要审核，请确认命令安全性",
				"allowed_decisions": []string{"approve", "reject", "edit"},
			},
			"Write": map[string]interface{}{
				"message":           "📝 文件写入操作需要审核",
				"allowed_decisions": []string{"approve", "reject", "edit"},
			},
			"fs_delete": map[string]interface{}{
				"message":           "🗑️  文件删除操作需要审核",
				"allowed_decisions": []string{"approve", "reject"},
			},
		},
		// 智能审核处理器
		ApprovalHandler: smartApprovalHandler,
	})
}

func smartApprovalHandler(ctx context.Context, req *middleware.ReviewRequest) ([]middleware.Decision, error) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🚨 HUMAN-IN-THE-LOOP 审核请求")
	fmt.Println(strings.Repeat("=", 70))

	for i, action := range req.ActionRequests {
		fmt.Printf("\n【操作 %d/%d】\n", i+1, len(req.ActionRequests))
		fmt.Printf("工具名称: %s\n", action.ToolName)
		fmt.Printf("工具参数:\n")
		for key, value := range action.Input {
			fmt.Printf("  %s: %v\n", key, value)
		}
		fmt.Printf("\n%s\n", action.Message)

		// 基于风险评估
		risk := assessRisk(action)
		fmt.Printf("\n风险级别: %s\n", getRiskLabel(risk))

		// 根据风险级别决定审核策略
		switch risk {
		case RiskLow:
			fmt.Println("✅ 低风险操作，自动批准")
			return []middleware.Decision{{
				Type:   middleware.DecisionApprove,
				Reason: "低风险操作自动批准",
			}}, nil

		case RiskMedium:
			fmt.Println("\n⚠️  中风险操作，需要确认")
			return promptForDecision(action, req.ReviewConfigs[i])

		case RiskHigh:
			fmt.Println("\n🚨 高风险操作，需要明确确认")
			return promptForHighRiskDecision(action)
		}
	}

	return nil, fmt.Errorf("no decision made")
}

type RiskLevel int

const (
	RiskLow    RiskLevel = 1
	RiskMedium RiskLevel = 2
	RiskHigh   RiskLevel = 3
)

func assessRisk(action middleware.ActionRequest) RiskLevel {
	switch action.ToolName {
	case "Bash":
		if cmd, ok := action.Input["command"].(string); ok {
			// 高风险命令
			highRiskPatterns := []string{"rm -rf", "mkfs", "dd if=", "format", "> /dev/"}
			for _, pattern := range highRiskPatterns {
				if strings.Contains(cmd, pattern) {
					return RiskHigh
				}
			}

			// 中风险命令
			mediumRiskPatterns := []string{"rm ", "mv ", "chmod", "chown", "kill", "pkill"}
			for _, pattern := range mediumRiskPatterns {
				if strings.Contains(cmd, pattern) {
					return RiskMedium
				}
			}

			// 低风险命令
			return RiskLow
		}

	case "fs_delete":
		return RiskHigh

	case "Write":
		if path, ok := action.Input["path"].(string); ok {
			systemPaths := []string{"/etc", "/usr", "/bin", "/boot", "/sys"}
			for _, sp := range systemPaths {
				if strings.HasPrefix(path, sp) {
					return RiskHigh
				}
			}
			return RiskMedium
		}
	}

	return RiskLow
}

func getRiskLabel(risk RiskLevel) string {
	switch risk {
	case RiskLow:
		return "🟢 低"
	case RiskMedium:
		return "🟡 中"
	case RiskHigh:
		return "🔴 高"
	default:
		return "❓ 未知"
	}
}

func promptForDecision(action middleware.ActionRequest, config middleware.InterruptConfig) ([]middleware.Decision, error) {
	fmt.Println("\n可用操作:")
	hasEdit := false
	for _, decision := range config.AllowedDecisions {
		switch decision {
		case middleware.DecisionApprove:
			fmt.Println("  [a] approve - 批准执行")
		case middleware.DecisionReject:
			fmt.Println("  [r] reject  - 拒绝执行")
		case middleware.DecisionEdit:
			fmt.Println("  [e] edit    - 编辑参数后执行")
			hasEdit = true
		}
	}

	fmt.Print("\n你的选择: ")
	var choice string
	fmt.Scanln(&choice)

	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "a", "approve":
		return []middleware.Decision{{
			Type:   middleware.DecisionApprove,
			Reason: "用户批准执行",
		}}, nil

	case "r", "reject":
		fmt.Print("拒绝原因 (可选): ")
		var reason string
		fmt.Scanln(&reason)
		if reason == "" {
			reason = "用户拒绝"
		}
		return []middleware.Decision{{
			Type:   middleware.DecisionReject,
			Reason: reason,
		}}, nil

	case "e", "edit":
		if !hasEdit {
			fmt.Println("❌ 此操作不支持编辑")
			return promptForDecision(action, config)
		}
		return promptForEdit(action)

	default:
		fmt.Println("❌ 无效的选择，请重新输入")
		return promptForDecision(action, config)
	}
}

func promptForHighRiskDecision(action middleware.ActionRequest) ([]middleware.Decision, error) {
	fmt.Println("\n⚠️  这是一个高风险操作！")
	fmt.Println("如果你确定要执行，请输入 'CONFIRM'")
	fmt.Println("否则输入任何其他内容拒绝")

	fmt.Print("\n确认: ")
	var confirm string
	fmt.Scanln(&confirm)

	if confirm == "CONFIRM" {
		return []middleware.Decision{{
			Type:   middleware.DecisionApprove,
			Reason: "用户明确确认高风险操作",
		}}, nil
	}

	return []middleware.Decision{{
		Type:   middleware.DecisionReject,
		Reason: "高风险操作未通过确认",
	}}, nil
}

func promptForEdit(action middleware.ActionRequest) ([]middleware.Decision, error) {
	fmt.Println("\n✏️  编辑参数:")
	editedInput := make(map[string]interface{})

	for key, value := range action.Input {
		fmt.Printf("\n%s (当前值: %v)\n", key, value)
		fmt.Print("新值 (按回车保持不变): ")

		var newValue string
		fmt.Scanln(&newValue)

		if newValue != "" {
			editedInput[key] = newValue
		} else {
			editedInput[key] = value
		}
	}

	return []middleware.Decision{{
		Type:        middleware.DecisionEdit,
		EditedInput: editedInput,
		Reason:      "用户编辑参数后执行",
	}}, nil
}

func buildSystemPrompt() string {
	return `你是一个演示 Human-in-the-Loop (HITL) 功能的 AI Agent。

你的任务是帮助用户完成各种操作，但某些敏感操作需要人工审核。

## 审核机制

以下工具调用需要人工审核：
- Bash: 执行 Shell 命令
- Write: 写入文件
- fs_delete: 删除文件

当你调用这些工具时：
1. 系统会暂停执行，等待人工审核
2. 审核员可以批准、拒绝或修改参数
3. 如果被拒绝，你应该尝试其他方法或向用户说明情况

## 行为准则

1. 清楚解释为什么需要执行某个操作
2. 提供足够的上下文帮助审核
3. 尊重人工决策，不要重复尝试被拒绝的操作
4. 如果操作被拒绝，向用户解释并提供替代方案

` + middleware.HITL_SYSTEM_PROMPT
}

func runDemo(ctx context.Context, ag *agent.Agent) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🎯 Human-in-the-Loop (HITL) 功能演示")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("\n本演示将展示 HITL 中间件如何拦截和审核敏感操作。")
	fmt.Println("你将看到不同风险级别的操作如何被处理。\n")

	scenarios := []struct {
		name    string
		message string
	}{
		{
			name:    "低风险操作",
			message: "请列出当前目录的文件",
		},
		{
			name:    "中风险操作",
			message: "请删除 /tmp/test.txt 文件",
		},
		{
			name:    "高风险操作",
			message: "请执行 rm -rf /tmp/* 命令",
		},
	}

	for i, scenario := range scenarios {
		fmt.Printf("\n【场景 %d: %s】\n", i+1, scenario.name)
		fmt.Printf("用户请求: %s\n", scenario.message)
		fmt.Println(strings.Repeat("-", 70))

		result, err := ag.Chat(ctx, scenario.message, &types.ChatOptions{
			MaxIterations: 3,
		})

		if err != nil {
			fmt.Printf("❌ 错误: %v\n", err)
			continue
		}

		fmt.Printf("\n✅ Agent 响应: %s\n", result.Text)
		fmt.Println(strings.Repeat("=", 70))

		// 询问是否继续
		if i < len(scenarios)-1 {
			fmt.Print("\n按回车继续下一个场景...")
			fmt.Scanln()
		}
	}

	fmt.Println("\n✨ 演示完成！")
}
