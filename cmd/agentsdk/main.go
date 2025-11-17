package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/wordflowlab/agentsdk/pkg/agent"
	"github.com/wordflowlab/agentsdk/pkg/appconfig"
	"github.com/wordflowlab/agentsdk/pkg/evals"
	"github.com/wordflowlab/agentsdk/pkg/mcpserver"
	"github.com/wordflowlab/agentsdk/pkg/memory"
	"github.com/wordflowlab/agentsdk/pkg/provider"
	"github.com/wordflowlab/agentsdk/pkg/router"
	"github.com/wordflowlab/agentsdk/pkg/sandbox"
	"github.com/wordflowlab/agentsdk/pkg/server"
	"github.com/wordflowlab/agentsdk/pkg/store"
	"github.com/wordflowlab/agentsdk/pkg/tools"
	"github.com/wordflowlab/agentsdk/pkg/tools/builtin"
	"github.com/wordflowlab/agentsdk/pkg/types"
	"github.com/wordflowlab/agentsdk/pkg/vector"
	pgvectorstore "github.com/wordflowlab/agentsdk/pkg/vector/pgvector"
)

// agentsdk CLI
//
// 当前提供的子命令:
//   - serve: 启动一个 HTTP Server, 暴露 /v1/agents/chat 和 /v1/agents/chat/stream
//
// 使用示例:
//   go install ./cmd/agentsdk@latest
//   agentsdk serve --addr :8080 --workspace ./workspace --store .agentsdk
//
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "serve":
		if err := runServe(os.Args[2:]); err != nil {
			log.Fatalf("agentsdk serve failed: %v", err)
		}
	case "mcp-serve":
		if err := runMCPServe(os.Args[2:]); err != nil {
			log.Fatalf("agentsdk mcp-serve failed: %v", err)
		}
	case "subagent":
		if err := runSubagent(os.Args[2:]); err != nil {
			log.Fatalf("agentsdk subagent failed: %v", err)
		}
	case "eval":
		if err := runEval(os.Args[2:]); err != nil {
			log.Fatalf("agentsdk eval failed: %v", err)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  agentsdk serve [flags]")
	fmt.Println("  agentsdk mcp-serve [flags]")
	fmt.Println("  agentsdk subagent [flags]")
	fmt.Println("  agentsdk eval [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  serve      Start an HTTP server that exposes AgentSDK chat APIs")
	fmt.Println("  mcp-serve  Start an MCP HTTP server that exposes local tools via MCP protocol")
	fmt.Println("  subagent   Run a focused sub-agent (Plan/Explore/general-purpose) once and print the result")
	fmt.Println("  eval       Run local text evals (keyword coverage, lexical similarity, etc.)")
	fmt.Println()
	fmt.Println("Use 'agentsdk <subcommand> -h' for subcommand-specific flags.")
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "HTTP listen address")
	workspace := fs.String("workspace", "./workspace", "Sandbox workspace directory")
	storeDir := fs.String("store", ".agentsdk", "Directory for JSON store data")
	templateID := fs.String("template", "assistant", "Default Agent template ID")
	modelName := fs.String("model", "claude-sonnet-4-5", "Default model name")
	configPath := fs.String("config", "", "Optional YAML config file for templates and routing")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// 可选应用配置
	var appCfg *appconfig.Config
	if *configPath != "" {
		cfg, err := appconfig.Load(*configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		appCfg = cfg
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Println("[WARN] ANTHROPIC_API_KEY not set, requests that require model calls will fail.")
	}

	if err := os.MkdirAll(*workspace, 0o755); err != nil {
		return fmt.Errorf("create workspace dir: %w", err)
	}

	ctx := context.Background()

	// 1. 工具注册表
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// 2. Sandbox 工厂
	sandboxFactory := sandbox.NewFactory()

	// 3. Provider 工厂
	providerFactory := provider.NewMultiProviderFactory()

	// 4. Store
	jsonStore, err := store.NewJSONStore(*storeDir)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}

	// 5. 模板注册表 (支持可选配置文件)
	templateRegistry := agent.NewTemplateRegistry()

	if appCfg != nil {
		for _, t := range appCfg.Templates {
			toolsList := make([]interface{}, 0, len(t.Tools))
			for _, name := range t.Tools {
				toolsList = append(toolsList, name)
			}
			templateRegistry.Register(&types.AgentTemplateDefinition{
				ID:           t.ID,
				Model:        t.Model,
				SystemPrompt: t.SystemPrompt,
				Tools:        toolsList,
			})
		}

		// 如果配置文件里包含了 templateID 对应模板, 就不再强行注册内置模板。
		if _, err := templateRegistry.Get(*templateID); err != nil {
			// 回退到内置模板
			templateRegistry.Register(&types.AgentTemplateDefinition{
				ID:    *templateID,
				Model: *modelName,
				SystemPrompt: "You are a helpful assistant with access to filesystem and memory tools. " +
					"Use tools when appropriate to read/write files or manage long-term memory.",
				Tools: []interface{}{"Read", "Write", "Bash"},
			})
		}
	} else {
		// 未指定配置文件时, 使用内置模板
		templateRegistry.Register(&types.AgentTemplateDefinition{
			ID:    *templateID,
			Model: *modelName,
			SystemPrompt: "You are a helpful assistant with access to filesystem and memory tools. " +
				"Use tools when appropriate to read/write files or manage long-term memory.",
			Tools: []interface{}{"Read", "Write", "Bash"},
		})
	}

	// 6. 配置 Router:
	//    - 如果指定了 config 且其中包含 routing 配置, 使用配置文件中的 profiles。
	//    - 否则回退到内置的 quality/cost 路由(Anthropic + Deepseek)。
	var rt router.Router

	if appCfg != nil {
		if appCfg.Routing != nil && len(appCfg.Routing.Profiles) > 0 {
			var (
				defaultModel *types.ModelConfig
				routes       []router.StaticRouteEntry
			)

			for profile, rp := range appCfg.Routing.Profiles {
				if rp.Provider == "" || rp.Model == "" {
					continue
				}

				apiKeyEnv := rp.EnvAPIKey
				if apiKeyEnv == "" {
					// 简单映射: 根据 provider 推断默认 env 名称
					switch rp.Provider {
					case "anthropic":
						apiKeyEnv = "ANTHROPIC_API_KEY"
					case "deepseek":
						apiKeyEnv = "DEEPSEEK_API_KEY"
					case "glm", "zhipu", "bigmodel":
						apiKeyEnv = "ZHIPUAI_API_KEY"
					}
				}
				apiKeyVal := ""
				if apiKeyEnv != "" {
					apiKeyVal = os.Getenv(apiKeyEnv)
				}

				modelCfg := &types.ModelConfig{
					Provider: rp.Provider,
					Model:    rp.Model,
					APIKey:   apiKeyVal,
				}

				if defaultModel == nil {
					defaultModel = modelCfg
				}

				routes = append(routes, router.StaticRouteEntry{
					Task:     "chat",
					Priority: router.Priority(profile),
					Model:    modelCfg,
				})
			}

			if defaultModel != nil {
				rt = router.NewStaticRouter(defaultModel, routes)
			}
		}
	}

	// 如果未通过配置文件成功创建 Router, 使用内置 quality/cost 路由
	if rt == nil {
		defaultModel := &types.ModelConfig{
			Provider: "anthropic",
			Model:    *modelName,
			APIKey:   apiKey,
		}

		var routes []router.StaticRouteEntry

		// quality: 默认使用 Anthropic 模型
		routes = append(routes, router.StaticRouteEntry{
			Task:     "chat",
			Priority: router.PriorityQuality,
			Model: &types.ModelConfig{
				Provider: "anthropic",
				Model:    *modelName,
				APIKey:   apiKey,
			},
		})

		// cost: 如果配置了 DEEPSEEK_API_KEY, 则使用 Deepseek 模型
		if deepseekKey := os.Getenv("DEEPSEEK_API_KEY"); deepseekKey != "" {
			deepseekModel := os.Getenv("DEEPSEEK_MODEL")
			if deepseekModel == "" {
				deepseekModel = "deepseek-chat"
			}
			routes = append(routes, router.StaticRouteEntry{
				Task:     "chat",
				Priority: router.PriorityCost,
				Model: &types.ModelConfig{
					Provider: "deepseek",
					Model:    deepseekModel,
					APIKey:   deepseekKey,
				},
			})
		} else {
			log.Println("[INFO] DEEPSEEK_API_KEY not set, cost-first routing will fall back to default anthropic model.")
		}

		rt = router.NewStaticRouter(defaultModel, routes)
	}

	// 7. 可选: 配置语义记忆 + semantic_search 工具
	var semMem *memory.SemanticMemory
	if appCfg != nil && appCfg.SemanticMemory != nil && appCfg.SemanticMemory.Enabled {
		// 解析 VectorStore
		var storeCfg *appconfig.VectorStoreConfig
		for i := range appCfg.VectorStores {
			if appCfg.VectorStores[i].Name == appCfg.SemanticMemory.Store {
				storeCfg = &appCfg.VectorStores[i]
				break
			}
		}

		var store vector.VectorStore
		if storeCfg != nil {
			switch storeCfg.Kind {
			case "", "memory":
				store = vector.NewMemoryStore()
			case "pgvector":
				if storeCfg.DSN == "" || storeCfg.Dimension <= 0 {
					log.Printf("[agentsdk serve] pgvector store %q requires dsn and dimension; falling back to memory store", storeCfg.Name)
					store = vector.NewMemoryStore()
				} else {
					pgStore, err := pgvectorstore.New(&pgvectorstore.Config{
						DSN:       storeCfg.DSN,
						Table:     storeCfg.Table,
						Dimension: storeCfg.Dimension,
						Metric:    storeCfg.Metric,
					})
					if err != nil {
						log.Printf("[agentsdk serve] Failed to create pgvector store %q: %v; falling back to memory store", storeCfg.Name, err)
						store = vector.NewMemoryStore()
					} else {
						store = pgStore
					}
				}
			default:
				log.Printf("[agentsdk serve] Unsupported vector store kind %q, semantic memory disabled", storeCfg.Kind)
			}
		}

		// 解析 Embedder
		var embedCfg *appconfig.EmbedderConfig
		for i := range appCfg.Embedders {
			if appCfg.Embedders[i].Name == appCfg.SemanticMemory.Embedder {
				embedCfg = &appCfg.Embedders[i]
				break
			}
		}

		var emb vector.Embedder
		if embedCfg != nil {
			switch embedCfg.Kind {
			case "", "mock":
				emb = vector.NewMockEmbedder(16)
			case "openai":
				apiKey := ""
				if embedCfg.EnvAPIKey != "" {
					apiKey = os.Getenv(embedCfg.EnvAPIKey)
				}
				if apiKey == "" {
					log.Printf("[agentsdk serve] OpenAI embedder %q requires API key in env %s; falling back to mock embedder", embedCfg.Name, embedCfg.EnvAPIKey)
					emb = vector.NewMockEmbedder(16)
				} else {
					emb = vector.NewOpenAIEmbedder("", apiKey, embedCfg.Model)
				}
			default:
				log.Printf("[agentsdk serve] Unsupported embedder kind %q, semantic memory disabled", embedCfg.Kind)
			}
		}

		if store != nil && emb != nil {
			semMem = memory.NewSemanticMemory(memory.SemanticMemoryConfig{
				Store:          store,
				Embedder:       emb,
				NamespaceScope: appCfg.SemanticMemory.NamespaceScope,
				TopK:           appCfg.SemanticMemory.TopK,
			})

			toolRegistry.Register("SemanticSearch", func(cfg map[string]interface{}) (tools.Tool, error) {
				if cfg == nil {
					cfg = map[string]interface{}{}
				}
				if _, ok := cfg["semantic_memory"]; !ok {
					cfg["semantic_memory"] = semMem
				}
				return builtin.NewSemanticSearchTool(cfg)
			})
			toolRegistry.Register("semantic_search", func(cfg map[string]interface{}) (tools.Tool, error) {
				if cfg == nil {
					cfg = map[string]interface{}{}
				}
				if _, ok := cfg["semantic_memory"]; !ok {
					cfg["semantic_memory"] = semMem
				}
				return builtin.NewSemanticSearchTool(cfg)
			})

			log.Printf("[agentsdk serve] Semantic memory enabled with store=%s, embedder=%s",
				appCfg.SemanticMemory.Store, appCfg.SemanticMemory.Embedder)
		} else if appCfg.SemanticMemory.Enabled {
			log.Printf("[agentsdk serve] Semantic memory config present but store or embedder could not be created; semantic_search will not be available")
		}
	}

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandboxFactory,
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		Router:           rt,
		TemplateRegistry: templateRegistry,
	}

	// 6. 创建 Server
	srv := server.New(deps)

	mux := http.NewServeMux()
	mux.Handle("/v1/agents/chat", srv.ChatHandler())
	mux.Handle("/v1/agents/chat/stream", srv.ChatStreamHandler())
	mux.Handle("/v1/evals/text", srv.TextEvalHandler())
	mux.Handle("/v1/evals/session", srv.SessionEvalHandler())
	mux.Handle("/v1/evals/batch", srv.BatchEvalHandler())
	// 语义检索 HTTP 包装层, 仅在 SemanticMemory 启用时可用。
	if semMem != nil && semMem.Enabled() {
		mux.Handle("/v1/memory/semantic/search", semanticSearchHTTPHandler(semMem))
	}
	mux.Handle("/v1/workflows/demo/run", srv.WorkflowDemoRunHandler())
	mux.Handle("/v1/workflows/demo/run-eval", srv.WorkflowDemoRunEvalHandler())
	mux.Handle("/v1/workflows/demo/runs", srv.WorkflowDemoGetRunHandler())

	// Skills 管理 API:
	// - GET    /v1/skills           列出所有 Skills
	// - POST   /v1/skills           通过 JSON body 安装/更新 Skill
	// - GET    /v1/skills/{id}      获取单个 Skill 的版本信息
	// - DELETE /v1/skills/{id}      卸载 Skill
	mux.Handle("/v1/skills", srv.SkillsListOrCreateHandler())
	mux.Handle("/v1/skills/", srv.SkillsGetOrDeleteHandler())

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Printf("agentsdk: HTTP server started at http://localhost%s\n", *addr)
	fmt.Println("  POST /v1/agents/chat         (sync chat)")
	fmt.Println("  POST /v1/agents/chat/stream  (SSE streaming chat)")
	fmt.Println("  POST /v1/evals/text          (local text evals, answer-only)")
	fmt.Println("  POST /v1/evals/session       (local text evals, from session events)")
	fmt.Println("  POST /v1/evals/batch         (batch evals with LLM-based scorers)")
	if semMem != nil && semMem.Enabled() {
		fmt.Println("  POST /v1/memory/semantic/search (semantic search proxy over SemanticMemory)")
	}
	fmt.Println("  POST /v1/workflows/demo/run  (demo workflow run API)")
	fmt.Println("  POST /v1/workflows/demo/run-eval (demo workflow run + eval API)")
	fmt.Println("  GET  /v1/workflows/demo/runs (get stored demo workflow run by id)")

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failed: %w", err)
	}

	_ = ctx
	return nil
}

// runMCPServe 启动一个 MCP HTTP Server, 将本地工具暴露为 MCP 服务。
// 当前默认注册:
// - echo        : 简单回显工具
// - docs_get    : 读取 docs 目录下的文档文件
// - docs_search : 在 docs 目录中搜索关键字
func runMCPServe(args []string) error {
	fs := flag.NewFlagSet("mcp-serve", flag.ExitOnError)
	addr := fs.String("addr", ":8090", "MCP HTTP listen address")
	docsDir := fs.String("docs", "./docs/content", "Base directory for docs_get/docs_search tools")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// 1. 注册工具
	registry := tools.NewRegistry()

	// echo 工具
	registry.Register("echo", func(config map[string]interface{}) (tools.Tool, error) {
		return &EchoTool{}, nil
	})

	// 文档工具
	if docsGet, err := mcpserver.NewDocsGetTool(*docsDir); err == nil {
		registry.Register(docsGet.Name(), func(cfg map[string]interface{}) (tools.Tool, error) {
			return docsGet, nil
		})
	} else {
		log.Printf("[agentsdk mcp-serve] docs_get disabled: %v", err)
	}

	if docsSearch, err := mcpserver.NewDocsSearchTool(*docsDir); err == nil {
		registry.Register(docsSearch.Name(), func(cfg map[string]interface{}) (tools.Tool, error) {
			return docsSearch, nil
		})
	} else {
		log.Printf("[agentsdk mcp-serve] docs_search disabled: %v", err)
	}

	// 2. 创建 MCP Server
	srv, err := mcpserver.New(&mcpserver.Config{
		Registry: registry,
	})
	if err != nil {
		return fmt.Errorf("create MCP server: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", srv.Handler())

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Printf("agentsdk: MCP HTTP server started at http://localhost%s/mcp\n", *addr)
	fmt.Println("  tools:")
	for _, name := range registry.List() {
		fmt.Printf("    - %s\n", name)
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("mcp http server failed: %w", err)
	}
	return nil
}

// EchoTool 与 examples/mcp/server 中的示例一致, 用于测试 MCP 工具调用。
type EchoTool struct{}

func (t *EchoTool) Name() string        { return "echo" }
func (t *EchoTool) Description() string { return "Echo the input text with an optional prefix." }

func (t *EchoTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Text to echo.",
			},
			"prefix": map[string]interface{}{
				"type":        "string",
				"description": "Optional prefix to add before the text.",
			},
		},
		"required": []string{"text"},
	}
}

func (t *EchoTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	text, _ := input["text"].(string)
	prefix, _ := input["prefix"].(string)
	if prefix != "" {
		return fmt.Sprintf("%s%s", prefix, text), nil
	}
	return text, nil
}

func (t *EchoTool) Prompt() string {
	return "Use this tool to echo back user-provided text, optionally with a prefix."
}

// runSubagent 启动一个专门子代理进程并执行一次性任务。
//
// 这个子命令主要用于被 Task 内置工具调用:
//   agentsdk subagent --type=Plan --prompt='分析builtin工具测试需求'
//
// 也可以直接在命令行手动使用。
func runSubagent(args []string) error {
	fs := flag.NewFlagSet("subagent", flag.ExitOnError)

	subagentType := fs.String("type", "general-purpose", "Subagent type: general-purpose, Explore, Plan, statusline-setup")
	prompt := fs.String("prompt", "", "Natural language task description for this subagent")
	modelName := fs.String("model", "claude-sonnet-4-5", "Model name to use for the subagent")

	timeout := fs.Duration("timeout", 0, "Optional overall timeout for this subagent run (e.g. 30m)")
	_ = fs.Int("max-tokens", 0, "Maximum tokens for model responses (currently informational)")
	_ = fs.Float64("temperature", 0, "Sampling temperature (currently informational)")

	workspace := fs.String("workspace", ".", "Sandbox workspace directory for the subagent")
	storeDir := fs.String("store", ".agentsdk-subagent", "Directory for JSON store data for subagents")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*prompt) == "" {
		return fmt.Errorf("prompt is required")
	}

	ctx := context.Background()
	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	sandboxFactory := sandbox.NewFactory()

	providerFactory := provider.NewMultiProviderFactory()

	jsonStore, err := store.NewJSONStore(*storeDir)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}

	templateRegistry := agent.NewTemplateRegistry()

	templateID, template := buildSubagentTemplate(*subagentType, *modelName)
	templateRegistry.Register(template)

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandboxFactory,
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		TemplateRegistry: templateRegistry,
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Println("[WARN] ANTHROPIC_API_KEY not set, subagent model calls may fail.")
	}

	modelCfg := &types.ModelConfig{
		Provider: "anthropic",
		Model:    *modelName,
		APIKey:   apiKey,
	}

	sandboxCfg := &types.SandboxConfig{
		Kind:    types.SandboxKindLocal,
		WorkDir: *workspace,
	}

	agentConfig := &types.AgentConfig{
		TemplateID:  templateID,
		ModelConfig: modelCfg,
		Sandbox:     sandboxCfg,
		Middlewares: subagentMiddlewaresForType(*subagentType),
	}

	ag, err := agent.Create(ctx, agentConfig, deps)
	if err != nil {
		return fmt.Errorf("create subagent: %w", err)
	}
	defer ag.Close()

	result, err := ag.Chat(ctx, *prompt)
	if err != nil {
		return fmt.Errorf("subagent chat failed: %w", err)
	}

	text := strings.TrimSpace(result.Text)
	if text != "" {
		fmt.Println(text)
	}

	return nil
}

func buildSubagentTemplate(subagentType, modelName string) (string, *types.AgentTemplateDefinition) {
	var (
		id         string
		systemText string
		toolNames  []string
	)

	switch subagentType {
	case "Explore":
		id = "subagent-explore"
		systemText = "You are a focused code exploration sub-agent. " +
			"Use filesystem tools (Read, Glob, Grep) and safe shell commands (Bash) to understand the current project state. " +
			"Favor reading and searching over making changes. " +
			"Summarize your findings clearly for the parent agent."
		toolNames = []string{"Read", "Glob", "Grep", "Bash", "TodoWrite", "ExitPlanMode"}
	case "Plan":
		id = "subagent-plan"
		systemText = "You are a planning sub-agent. " +
			"Your job is to analyze the user's goal and the existing code, then propose a concrete, step-by-step implementation plan. " +
			"Use Read/Glob/Grep to gather context, and use TodoWrite to maintain a structured task list. " +
			"When the plan is ready, call ExitPlanMode with a clear Markdown plan for the parent agent to execute."
		toolNames = []string{"Read", "Glob", "Grep", "TodoWrite", "ExitPlanMode"}
	case "statusline-setup":
		id = "subagent-statusline-setup"
		systemText = "You are a small configuration sub-agent focused on setting up editor or CLI status lines. " +
			"Inspect config files, suggest minimal safe edits, and keep changes localized."
		fsTools := builtin.FileSystemTools()
		execTools := builtin.ExecutionTools()
		toolNames = append(fsTools, execTools...)
	default:
		id = "subagent-general-purpose"
		systemText = "You are a general-purpose coding sub-agent. " +
			"Work independently on the delegated task using the available tools. " +
			"Plan your work, use filesystem and shell tools when needed, and return a concise summary of what you did and what you found."
		toolNames = builtin.AllTools()
	}

	toolsInterface := make([]interface{}, 0, len(toolNames))
	for _, name := range toolNames {
		toolsInterface = append(toolsInterface, name)
	}

	template := &types.AgentTemplateDefinition{
		ID:           id,
		Model:        modelName,
		SystemPrompt: systemText,
		Tools:        toolsInterface,
	}

	return id, template
}

func subagentMiddlewaresForType(subagentType string) []string {
	switch subagentType {
	case "Plan":
		return []string{"filesystem", "todolist"}
	case "Explore":
		return []string{"filesystem"}
	default:
		return []string{"filesystem", "summarization"}
	}
}

// runEval 运行本地文本评估, 不依赖外部 LLM。
//
// 示例:
//   echo "Paris is the capital of France." | agentsdk eval -reference "Paris is the capital city of France, a country in Europe." -keywords "paris,capital,france,europe"
//
//   agentsdk eval -answer "..." -reference "..." -keywords "a,b,c" -json
//
//   agentsdk eval -file cases.jsonl -json
func runEval(args []string) error {
	fs := flag.NewFlagSet("eval", flag.ExitOnError)
	answer := fs.String("answer", "", "Answer text to evaluate. If empty, read from stdin.")
	reference := fs.String("reference", "", "Optional reference text for lexical similarity scorer.")
	keywordsStr := fs.String("keywords", "", "Comma-separated keywords for keyword coverage scorer.")
	minTokenLen := fs.Int("min-token-length", 2, "Minimum token length for lexical similarity scorer.")
	disableKeyword := fs.Bool("no-keywords", false, "Disable keyword coverage scorer.")
	disableSimilarity := fs.Bool("no-similarity", false, "Disable lexical similarity scorer.")
	jsonOut := fs.Bool("json", false, "Output results as pretty-printed JSON.")
	filePath := fs.String("file", "", "Optional JSONL file with eval cases. Each line: {\"answer\": \"...\", \"reference\": \"...\", \"keywords\": [\"...\"]}")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// 批量模式: 从 JSONL 文件读取多条样本
	if *filePath != "" {
		return runEvalFromFile(*filePath, *minTokenLen, *disableKeyword, *disableSimilarity, *jsonOut)
	}

	text := strings.TrimSpace(*answer)
	if text == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		text = strings.TrimSpace(string(data))
	}
	if text == "" {
		return fmt.Errorf("answer text is empty; provide -answer or pipe text via stdin")
	}

	input := &evals.TextEvalInput{
		Answer:    text,
		Reference: *reference,
	}

	ctx := context.Background()
	results := make([]*evals.ScoreResult, 0, 2)

	if !*disableKeyword {
		var keywords []string
		if *keywordsStr != "" {
			parts := strings.Split(*keywordsStr, ",")
			for _, p := range parts {
				if trimmed := strings.TrimSpace(p); trimmed != "" {
					keywords = append(keywords, trimmed)
				}
			}
		}
		scorer := evals.NewKeywordCoverageScorer(evals.KeywordCoverageConfig{
			Keywords:        keywords,
			CaseInsensitive: true,
		})
		score, err := scorer.Score(ctx, input)
		if err != nil {
			return fmt.Errorf("keyword_coverage scorer error: %w", err)
		}
		results = append(results, score)
	}

	if !*disableSimilarity {
		scorer := evals.NewLexicalSimilarityScorer(evals.LexicalSimilarityConfig{
			MinTokenLength: *minTokenLen,
		})
		score, err := scorer.Score(ctx, input)
		if err != nil {
			return fmt.Errorf("lexical_similarity scorer error: %w", err)
		}
		results = append(results, score)
	}

	if *jsonOut {
		flat := make([]evals.ScoreResult, 0, len(results))
		for _, r := range results {
			if r != nil {
				flat = append(flat, *r)
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(flat); err != nil {
			return fmt.Errorf("encode JSON: %w", err)
		}
		return nil
	}

	for _, r := range results {
		if r == nil {
			continue
		}
		fmt.Printf("%s: %.4f\n", r.Name, r.Value)
		if len(r.Details) > 0 {
			fmt.Printf("  details: %+v\n", r.Details)
		}
	}

	return nil
}

// semanticSearchHTTPHandler 将 SemanticMemory 暴露为简单的 HTTP 接口:
// POST /v1/memory/semantic/search
// 请求: { "query": "...", "top_k": 5, "metadata": {...} }
// 响应: { "hits": [ { "id": "...", "score": 0.87, "metadata": {...} }, ... ] }
func semanticSearchHTTPHandler(sm *memory.SemanticMemory) http.Handler {
	type request struct {
		Query    string                 `json:"query"`
		TopK     int                    `json:"top_k,omitempty"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}
	type hit struct {
		ID       string                 `json:"id"`
		Score    float64                `json:"score"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}
	type response struct {
		Hits         []hit  `json:"hits"`
		ErrorMessage string `json:"error_message,omitempty"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if sm == nil || !sm.Enabled() {
			http.Error(w, "semantic memory not configured", http.StatusServiceUnavailable)
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Query) == "" {
			http.Error(w, "query is required", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		hitsRaw, err := sm.Search(ctx, req.Query, req.Metadata, req.TopK)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(&response{
				ErrorMessage: err.Error(),
			})
			return
		}

		out := make([]hit, 0, len(hitsRaw))
		for _, h := range hitsRaw {
			out = append(out, hit{
				ID:       h.ID,
				Score:    h.Score,
				Metadata: h.Metadata,
			})
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(&response{Hits: out})
	})
}

// evalCase 用于解析 JSONL 文件中的单条评估样本。
type evalCase struct {
	Answer    string   `json:"answer"`
	Reference string   `json:"reference,omitempty"`
	Keywords  []string `json:"keywords,omitempty"`
}

// runEvalFromFile 从 JSONL 文件中读取多条样本并逐条评估。
func runEvalFromFile(path string, minTokenLen int, disableKeyword, disableSimilarity, jsonOut bool) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 支持较大行

	ctx := context.Background()
	lineNo := 0

	enc := json.NewEncoder(os.Stdout)
	if jsonOut {
		enc.SetIndent("", "  ")
	}

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var c evalCase
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			return fmt.Errorf("parse JSONL line %d: %w", lineNo, err)
		}
		if strings.TrimSpace(c.Answer) == "" {
			return fmt.Errorf("line %d: answer is empty", lineNo)
		}

		input := &evals.TextEvalInput{
			Answer:    c.Answer,
			Reference: c.Reference,
		}

		results := make([]*evals.ScoreResult, 0, 2)

		if !disableKeyword {
			scorer := evals.NewKeywordCoverageScorer(evals.KeywordCoverageConfig{
				Keywords:        c.Keywords,
				CaseInsensitive: true,
			})
			score, err := scorer.Score(ctx, input)
			if err != nil {
				return fmt.Errorf("line %d: keyword_coverage scorer error: %w", lineNo, err)
			}
			results = append(results, score)
		}

		if !disableSimilarity {
			scorer := evals.NewLexicalSimilarityScorer(evals.LexicalSimilarityConfig{
				MinTokenLength: minTokenLen,
			})
			score, err := scorer.Score(ctx, input)
			if err != nil {
				return fmt.Errorf("line %d: lexical_similarity scorer error: %w", lineNo, err)
			}
			results = append(results, score)
		}

		if jsonOut {
			flat := make([]evals.ScoreResult, 0, len(results))
			for _, r := range results {
				if r != nil {
					flat = append(flat, *r)
				}
			}
			wrapped := map[string]interface{}{
				"line":    lineNo,
				"answer":  c.Answer,
				"scores":  flat,
				"keywords": c.Keywords,
			}
			if err := enc.Encode(wrapped); err != nil {
				return fmt.Errorf("encode JSON for line %d: %w", lineNo, err)
			}
		} else {
			fmt.Printf("Line %d:\n", lineNo)
			for _, r := range results {
				if r == nil {
					continue
				}
				fmt.Printf("  %s: %.4f\n", r.Name, r.Value)
				if len(r.Details) > 0 {
					fmt.Printf("    details: %+v\n", r.Details)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan file: %w", err)
	}

	return nil
}
