package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/commands"
	"github.com/wordflowlab/agentsdk/pkg/events"
	"github.com/wordflowlab/agentsdk/pkg/provider"
	"github.com/wordflowlab/agentsdk/pkg/sandbox"
	"github.com/wordflowlab/agentsdk/pkg/skills"
	"github.com/wordflowlab/agentsdk/pkg/tools"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// Agent AI代理
type Agent struct {
	// 基础配置
	id       string
	template *types.AgentTemplateDefinition
	config   *types.AgentConfig
	deps     *Dependencies

	// 核心组件
	eventBus *events.EventBus
	provider provider.Provider
	sandbox  sandbox.Sandbox
	executor *tools.Executor
	toolMap  map[string]tools.Tool

	// Slash Commands & Skills 支持
	commandExecutor *commands.Executor
	skillInjector   *skills.Injector

	// 状态管理
	mu           sync.RWMutex
	state        types.AgentRuntimeState
	breakpoint   types.BreakpointState
	messages     []types.Message
	toolRecords  map[string]*types.ToolCallRecord
	stepCount    int
	lastSfpIndex int
	lastBookmark *types.Bookmark
	createdAt    time.Time

	// 权限管理
	pendingPermissions map[string]chan string // callID -> decision channel

	// 控制信号
	stopCh chan struct{}
}

// Create 创建新Agent
func Create(ctx context.Context, config *types.AgentConfig, deps *Dependencies) (*Agent, error) {
	// 生成AgentID
	if config.AgentID == "" {
		config.AgentID = generateAgentID()
	}

	// 获取模板
	template, err := deps.TemplateRegistry.Get(config.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	// 创建Provider
	modelConfig := config.ModelConfig
	if modelConfig == nil && template.Model != "" {
		modelConfig = &types.ModelConfig{
			Provider: "anthropic",
			Model:    template.Model,
		}
	}
	if modelConfig == nil {
		return nil, fmt.Errorf("model config is required")
	}

	prov, err := deps.ProviderFactory.Create(modelConfig)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}

	// 创建Sandbox
	sandboxConfig := config.Sandbox
	if sandboxConfig == nil {
		sandboxConfig = &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: ".",
		}
	}

	sb, err := deps.SandboxFactory.Create(sandboxConfig)
	if err != nil {
		return nil, fmt.Errorf("create sandbox: %w", err)
	}

	// 创建工具执行器
	executor := tools.NewExecutor(tools.ExecutorConfig{
		MaxConcurrency: 3,
		DefaultTimeout: 60 * time.Second,
	})

	// 解析工具列表
	toolNames := config.Tools
	if toolNames == nil {
		// 使用模板的工具列表
		if toolsVal, ok := template.Tools.([]string); ok {
			toolNames = toolsVal
		} else if template.Tools == "*" {
			toolNames = deps.ToolRegistry.List()
		}
	}

	// 创建工具实例
	toolMap := make(map[string]tools.Tool)
	for _, name := range toolNames {
		tool, err := deps.ToolRegistry.Create(name, nil)
		if err != nil {
			continue // 忽略未注册的工具
		}
		toolMap[name] = tool
	}

	// 初始化 Slash Commands & Skills（如果配置了）
	var cmdExecutor *commands.Executor
	var skillInjector *skills.Injector

	if config.SkillsPackage != nil {
		// 初始化命令执行器
		commandsDir := config.SkillsPackage.CommandsDir
		if commandsDir == "" {
			commandsDir = "commands"
		}
		commandLoader := commands.NewLoader(commandsDir, sb.FS())
		cmdExecutor = commands.NewExecutor(&commands.ExecutorConfig{
			Loader:       commandLoader,
			Sandbox:      sb,
			Provider:     prov,
			Capabilities: prov.Capabilities(),
		})

		// 初始化技能注入器
		skillsDir := config.SkillsPackage.SkillsDir
		if skillsDir == "" {
			skillsDir = "skills"
		}
		skillLoader := skills.NewLoader(skillsDir, sb.FS())
		skillInjector, err = skills.NewInjector(ctx, &skills.InjectorConfig{
			Loader:        skillLoader,
			EnabledSkills: config.SkillsPackage.EnabledSkills,
			Provider:      prov,
			Capabilities:  prov.Capabilities(),
		})
		if err != nil {
			return nil, fmt.Errorf("create skill injector: %w", err)
		}
	}

	// 创建Agent
	agent := &Agent{
		id:                 config.AgentID,
		template:           template,
		config:             config,
		deps:               deps,
		eventBus:           events.NewEventBus(),
		provider:           prov,
		sandbox:            sb,
		executor:           executor,
		toolMap:            toolMap,
		commandExecutor:    cmdExecutor,
		skillInjector:      skillInjector,
		state:              types.AgentStateReady,
		breakpoint:         types.BreakpointReady,
		messages:           []types.Message{},
		toolRecords:        make(map[string]*types.ToolCallRecord),
		pendingPermissions: make(map[string]chan string),
		createdAt:          time.Now(),
		stopCh:             make(chan struct{}),
	}

	// 初始化
	if err := agent.initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize agent: %w", err)
	}

	return agent, nil
}

// initialize 初始化Agent
func (a *Agent) initialize(ctx context.Context) error {
	// 从Store加载状态
	messages, err := a.deps.Store.LoadMessages(ctx, a.id)
	if err == nil && len(messages) > 0 {
		a.messages = messages
	}

	toolRecords, err := a.deps.Store.LoadToolCallRecords(ctx, a.id)
	if err == nil {
		for _, record := range toolRecords {
			a.toolRecords[record.ID] = &record
		}
	}

	// 保存Agent信息
	info := types.AgentInfo{
		AgentID:       a.id,
		TemplateID:    a.template.ID,
		CreatedAt:     a.createdAt,
		Lineage:       []string{},
		ConfigVersion: "v1.0.0",
		MessageCount:  len(a.messages),
	}

	return a.deps.Store.SaveInfo(ctx, a.id, info)
}

// ID 返回AgentID
func (a *Agent) ID() string {
	return a.id
}

// Send 发送消息
func (a *Agent) Send(ctx context.Context, text string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 检测 slash command
	if strings.HasPrefix(text, "/") {
		return a.handleSlashCommand(ctx, text)
	}

	// 准备消息内容
	messageText := text

	// 如果启用了 skills，增强消息
	if a.skillInjector != nil {
		skillContext := skills.SkillContext{
			UserMessage: text,
			Files:       a.getRecentFiles(),
			Metadata:    make(map[string]interface{}),
		}

		// 增强 system prompt（对于支持的模型）
		caps := a.provider.Capabilities()
		if caps.SupportSystemPrompt {
			enhancedSysPrompt := a.skillInjector.EnhanceSystemPrompt(
				ctx,
				a.template.SystemPrompt,
				skillContext,
			)
			a.provider.SetSystemPrompt(enhancedSysPrompt)
		} else {
			// 不支持 system prompt，增强 user message
			messageText = a.skillInjector.PrepareUserMessage(text, skillContext)
		}
	}

	// 创建用户消息
	message := types.Message{
		Role: types.MessageRoleUser,
		Content: []types.ContentBlock{
			&types.TextBlock{Text: messageText},
		},
	}

	a.messages = append(a.messages, message)
	a.stepCount++

	// 持久化
	if err := a.deps.Store.SaveMessages(ctx, a.id, a.messages); err != nil {
		return fmt.Errorf("save messages: %w", err)
	}

	// 触发处理
	go a.processMessages(ctx)

	return nil
}

// Chat 同步对话(阻塞式)
func (a *Agent) Chat(ctx context.Context, text string) (*types.CompleteResult, error) {
	// 发送消息
	if err := a.Send(ctx, text); err != nil {
		return nil, err
	}

	// 等待完成
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			a.mu.RLock()
			state := a.state
			a.mu.RUnlock()

			if state == types.AgentStateReady {
				// 提取最后的助手回复
				a.mu.RLock()
				defer a.mu.RUnlock()

				var text string
				for i := len(a.messages) - 1; i >= 0; i-- {
					if a.messages[i].Role == types.MessageRoleAssistant {
						for _, block := range a.messages[i].Content {
							if tb, ok := block.(*types.TextBlock); ok {
								text = tb.Text
								break
							}
						}
						break
					}
				}

				return &types.CompleteResult{
					Status: "ok",
					Text:   text,
					Last:   a.lastBookmark,
				}, nil
			}
		}
	}
}

// Subscribe 订阅事件
func (a *Agent) Subscribe(channels []types.AgentChannel, opts *types.SubscribeOptions) <-chan types.AgentEventEnvelope {
	return a.eventBus.Subscribe(channels, opts)
}

// Status 获取状态
func (a *Agent) Status() *types.AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return &types.AgentStatus{
		AgentID:      a.id,
		State:        a.state,
		StepCount:    a.stepCount,
		LastSfpIndex: a.lastSfpIndex,
		LastBookmark: a.lastBookmark,
		Cursor:       a.eventBus.GetCursor(),
		Breakpoint:   a.breakpoint,
	}
}

// Close 关闭Agent
func (a *Agent) Close() error {
	close(a.stopCh)

	if err := a.sandbox.Dispose(); err != nil {
		return err
	}

	return a.provider.Close()
}

// handleSlashCommand 处理 slash command
func (a *Agent) handleSlashCommand(ctx context.Context, text string) error {
	if a.commandExecutor == nil {
		return fmt.Errorf("slash commands not enabled")
	}

	// 解析命令和参数
	parts := strings.Fields(text)
	commandName := strings.TrimPrefix(parts[0], "/")

	args := make(map[string]string)
	if len(parts) > 1 {
		args["argument"] = strings.Join(parts[1:], " ")
	}

	// 执行命令并获取消息
	message, err := a.commandExecutor.Execute(ctx, commandName, args)
	if err != nil {
		return fmt.Errorf("execute command: %w", err)
	}

	// 将命令消息作为用户消息发送
	userMessage := types.Message{
		Role: types.MessageRoleUser,
		Content: []types.ContentBlock{
			&types.TextBlock{Text: message},
		},
	}

	a.messages = append(a.messages, userMessage)
	a.stepCount++

	// 持久化
	if err := a.deps.Store.SaveMessages(ctx, a.id, a.messages); err != nil {
		return fmt.Errorf("save messages: %w", err)
	}

	// 触发处理
	go a.processMessages(ctx)

	return nil
}

// getRecentFiles 获取最近访问的文件列表
func (a *Agent) getRecentFiles() []string {
	// TODO: 实现文件追踪逻辑
	// 可以从 toolRecords 中提取最近读写的文件
	return []string{}
}

// generateAgentID 生成AgentID
func generateAgentID() string {
	return "agt:" + uuid.New().String()
}
