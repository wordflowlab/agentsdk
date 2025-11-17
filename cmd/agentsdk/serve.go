package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/wordflowlab/agentsdk/pkg/agent"
	"github.com/wordflowlab/agentsdk/pkg/appconfig"
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
)

// runServe 启动 HTTP Server（使用 Gin）
func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "HTTP listen address")
	workspace := fs.String("workspace", "./workspace", "Sandbox workspace directory")
	storeDir := fs.String("store", ".agentsdk", "Directory for JSON store data")
	configPath := fs.String("config", "", "Optional YAML config file")
	apiKey := fs.String("api-key", "", "API Key for authentication (empty = disabled)")
	mode := fs.String("mode", "release", "Gin mode: debug, release, test")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// 设置 Gin 模式
	gin.SetMode(*mode)

	// 创建工作目录
	if err := os.MkdirAll(*workspace, 0o755); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey == "" {
		log.Println("[WARN] ANTHROPIC_API_KEY not set")
	}

	// 加载配置
	var appCfg *appconfig.Config
	if *configPath != "" {
		cfg, err := appconfig.Load(*configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		appCfg = cfg
	}

	// 创建核心依赖
	jsonStore, err := store.NewJSONStore(*storeDir)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}

	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	sandboxFactory := sandbox.NewFactory()
	providerFactory := provider.NewMultiProviderFactory()

	templateRegistry := agent.NewTemplateRegistry()
	registerBuiltinTemplates(templateRegistry)

	// Router
	var rt router.Router
	defaultModel := &types.ModelConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5",
		APIKey:   anthropicKey,
	}
	routes := []router.StaticRouteEntry{
		{Task: "chat", Priority: router.PriorityQuality, Model: defaultModel},
	}
	rt = router.NewStaticRouter(defaultModel, routes)

	// Semantic Memory (可选)
	var semMem *memory.SemanticMemory
	if appCfg != nil && appCfg.SemanticMemory != nil && appCfg.SemanticMemory.Enabled {
		semMemCfg := memory.SemanticMemoryConfig{
			Store:    vector.NewMemoryStore(),
			Embedder: vector.NewMockEmbedder(16),
		}
		semMem = memory.NewSemanticMemory(semMemCfg)
		log.Println("[INFO] Semantic memory enabled")
	}

	deps := &agent.Dependencies{
		Store:            jsonStore,
		SandboxFactory:   sandboxFactory,
		ToolRegistry:     toolRegistry,
		ProviderFactory:  providerFactory,
		Router:           rt,
		TemplateRegistry: templateRegistry,
	}

	// 创建旧 Server（用于兼容旧 API）
	oldServer := server.New(deps)

	// 创建 Gin 路由器
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "v0.8.0"})
	})

	// 根路径
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":    "AgentSDK API",
			"version": "v0.8.0",
			"docs":    "/v1",
		})
	})

	// v1 API
	v1 := r.Group("/v1")
	{
		// API Key 认证
		if *apiKey != "" {
			v1.Use(apiKeyAuthMiddleware(*apiKey))
		}

		// Agent API (新架构)
		registerAgentRoutes(v1, deps, jsonStore)

		// Session API (新架构)
		registerSessionRoutes(v1, jsonStore)

		// Memory API (新架构)
		registerMemoryRoutes(v1, jsonStore)

		// Workflow API (新架构)
		registerWorkflowRoutes(v1, jsonStore)

		// Tool API (新架构)
		registerToolRoutes(v1, jsonStore)

		// MCP API (新架构)
		registerMCPRoutes(v1, jsonStore)

		// Middleware API (新架构)
		registerMiddlewareRoutes(v1, jsonStore)

		// Telemetry API (新架构)
		registerTelemetryRoutes(v1, jsonStore)

		// Eval API (新架构)
		registerEvalRoutes(v1, jsonStore)

		// System API (新架构)
		registerSystemRoutes(v1, jsonStore)

		// Skills API (旧实现，适配 Gin)
		v1.GET("/skills", gin.WrapH(oldServer.SkillsListOrCreateHandler()))
		v1.POST("/skills", gin.WrapH(oldServer.SkillsListOrCreateHandler()))
		v1.Any("/skills/*path", gin.WrapH(oldServer.SkillsGetOrDeleteHandler()))

		// Evals API (旧实现)
		v1.POST("/evals/text", gin.WrapH(oldServer.TextEvalHandler()))
		v1.POST("/evals/session", gin.WrapH(oldServer.SessionEvalHandler()))
		v1.POST("/evals/batch", gin.WrapH(oldServer.BatchEvalHandler()))

		// Workflow Demo (旧实现)
		v1.POST("/workflows/demo/run", gin.WrapH(oldServer.WorkflowDemoRunHandler()))
		v1.POST("/workflows/demo/run-eval", gin.WrapH(oldServer.WorkflowDemoRunEvalHandler()))
		v1.GET("/workflows/demo/runs", gin.WrapH(oldServer.WorkflowDemoGetRunHandler()))

		// Semantic Search (如果启用)
		if semMem != nil && semMem.Enabled() {
			setupSemanticSearchHandler(v1, semMem)
		}
	}

	// 打印启动信息
	printServerInfo(*addr, *apiKey != "")

	return r.Run(*addr)
}

// apiKeyAuthMiddleware API Key 认证中间件
func apiKeyAuthMiddleware(requiredKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			auth := c.GetHeader("Authorization")
			if len(auth) > 7 && auth[:7] == "Bearer " {
				apiKey = auth[7:]
			}
		}
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey != requiredKey {
			c.JSON(401, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "unauthorized",
					"message": "Invalid or missing API key",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// setupSemanticSearchHandler 设置语义搜索handler
func setupSemanticSearchHandler(v1 *gin.RouterGroup, semMem *memory.SemanticMemory) {
	v1.POST("/memory/semantic/search", func(c *gin.Context) {
		var req struct {
			Query string `json:"query" binding:"required"`
			Limit int    `json:"limit"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "bad_request",
					"message": err.Error(),
				},
			})
			return
		}

		if req.Limit <= 0 {
			req.Limit = 10
		}

		metadata := make(map[string]interface{})
		results, err := semMem.Search(c.Request.Context(), req.Query, metadata, req.Limit)
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to search: " + err.Error(),
				},
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data": gin.H{
				"query":   req.Query,
				"results": results,
				"count":   len(results),
			},
		})
	})
}

// registerBuiltinTemplates 注册内置模板
func registerBuiltinTemplates(registry *agent.TemplateRegistry) {
	registry.Register(&types.AgentTemplateDefinition{
		ID:           "assistant",
		SystemPrompt: "You are a helpful assistant.",
		Tools:        []interface{}{"filesystem", "bash"},
	})

	registry.Register(&types.AgentTemplateDefinition{
		ID:           "coder",
		SystemPrompt: "You are an expert programmer.",
		Tools:        []interface{}{"filesystem", "bash", "grep"},
	})
}

// printServerInfo 打印服务器启动信息
func printServerInfo(addr string, hasAPIKey bool) {
	fmt.Printf("🚀 AgentSDK HTTP Server started at http://localhost%s\n", addr)
	fmt.Println("\n📍 Available Endpoints:")
	fmt.Println("  GET    /                         API info")
	fmt.Println("  GET    /health                   Health check")
	fmt.Println("\n🤖 Agent API (NEW - Gin架构):")
	fmt.Println("  POST   /v1/agents                Create")
	fmt.Println("  GET    /v1/agents                List")
	fmt.Println("  GET    /v1/agents/:id            Get")
	fmt.Println("  PATCH  /v1/agents/:id            Update")
	fmt.Println("  DELETE /v1/agents/:id            Delete")
	fmt.Println("  POST   /v1/agents/:id/activate   Activate")
	fmt.Println("  GET    /v1/agents/templates      Templates")
	fmt.Println("\n🎓 Skills API:")
	fmt.Println("  GET    /v1/skills                List")
	fmt.Println("  POST   /v1/skills                Install")
	fmt.Println("\n📊 Evals API:")
	fmt.Println("  POST   /v1/evals/text            Text eval")
	fmt.Println("  POST   /v1/evals/batch           Batch eval")
	fmt.Println()

	if hasAPIKey {
		fmt.Println("🔐 API Key authentication enabled")
	} else {
		fmt.Println("⚠️  API Key authentication disabled")
	}
	fmt.Println()
}
