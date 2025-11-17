package skills

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/wordflowlab/agentsdk/pkg/provider"
)

// InjectorConfig 注入器配置
type InjectorConfig struct {
	Loader        *SkillLoader
	EnabledSkills []string
	Provider      provider.Provider
	Capabilities  provider.ProviderCapabilities
}

// Injector 技能注入器
type Injector struct {
	loader        *SkillLoader
	skills        map[string]*SkillDefinition
	enabledSkills map[string]bool
	provider      provider.Provider
	capabilities  provider.ProviderCapabilities
}

// NewInjector 创建注入器
func NewInjector(ctx context.Context, config *InjectorConfig) (*Injector, error) {
	injector := &Injector{
		loader:        config.Loader,
		skills:        make(map[string]*SkillDefinition),
		enabledSkills: make(map[string]bool),
		provider:      config.Provider,
		capabilities:  config.Capabilities,
	}

	// 加载启用的技能
	if len(config.EnabledSkills) > 0 {
		skills, err := config.Loader.LoadMultiple(ctx, config.EnabledSkills)
		if err != nil {
			return nil, fmt.Errorf("load skills: %w", err)
		}
		injector.skills = skills

		for _, name := range config.EnabledSkills {
			injector.enabledSkills[name] = true
		}
	}

	return injector, nil
}

// EnhanceSystemPrompt 增强系统提示词
func (i *Injector) EnhanceSystemPrompt(ctx context.Context, basePrompt string, skillContext SkillContext) string {
	// 获取当前 Agent 已启用的技能集合。
	// 而是将所有启用的技能以元数据形式暴露给模型，由模型根据描述自行判断何时使用。
	activeSkills := i.getActiveSkills(skillContext)

	log.Printf("[Skills] Checking skills for message: %q", skillContext.UserMessage)
	log.Printf("[Skills] Enabled skills available to inject: %d", len(activeSkills))
	for _, skill := range activeSkills {
		log.Printf("[Skills] - Enabled: %s (%s)", skill.Name, skill.Description)
	}

	if len(activeSkills) == 0 {
		log.Printf("[Skills] No skills activated, returning base prompt")
		return basePrompt
	}

	// 根据模型能力选择注入方式
	if i.capabilities.SupportSystemPrompt {
		enhanced := i.injectToSystemPrompt(basePrompt, activeSkills)
		log.Printf("[Skills] Enhanced system prompt length: %d -> %d", len(basePrompt), len(enhanced))
		return enhanced
	}

	// 不支持 system prompt，返回原始提示词
	log.Printf("[Skills] Provider doesn't support system prompt, returning base")
	return basePrompt
}

// ActivateSkills 根据上下文返回应当激活的 Skill 列表
// 这是对内部 getActiveSkills 的公开包装，方便在自定义流程中手动控制注入。
func (i *Injector) ActivateSkills(ctx context.Context, skillContext SkillContext) []*SkillDefinition {
	return i.getActiveSkills(skillContext)
}

// InjectToSystemPrompt 将给定的 Skills 注入到 System Prompt。
// 与 EnhanceSystemPrompt 不同，这里假设调用方已经决定了要注入哪些 Skills。
func (i *Injector) InjectToSystemPrompt(basePrompt string, skills []*SkillDefinition) string {
	return i.injectToSystemPrompt(basePrompt, skills)
}

// InjectToUserMessage 将激活的 Skills 作为知识库注入到用户消息前。
// 这主要用于不支持独立 system prompt 的模型。
// 为了符合「渐进式加载」的设计，这里只注入 Skill 元数据，
// 具体的 SKILL.md 内容仍然通过文件系统工具按需读取。
func (i *Injector) InjectToUserMessage(userMessage string, skills []*SkillDefinition) string {
	if len(skills) == 0 {
		return userMessage
	}

	var b strings.Builder
	b.WriteString("## Skills Overview\n\n")
	b.WriteString("The following skills are available for this task. ")
	b.WriteString("Each skill's detailed instructions are stored on disk in its `SKILL.md` file. ")
	b.WriteString("When a skill is relevant, use filesystem tools (for example the `Read` or `Bash` tools) ")
	b.WriteString("to open the corresponding `SKILL.md`, then follow the instructions and any referenced scripts or resources.\n\n")

	for _, skill := range skills {
		if skill == nil {
			continue
		}

		// 生成 SKILL.md 提示路径（相对于沙箱工作目录）
		skillFileHint := ""
		path := strings.Trim(skill.Path, "/")
		baseDir := strings.Trim(skill.BaseDir, "/")
		if path != "" {
			if baseDir != "" {
				skillFileHint = fmt.Sprintf("%s/%s/SKILL.md", baseDir, path)
			} else {
				skillFileHint = fmt.Sprintf("%s/SKILL.md", path)
			}
		}

		b.WriteString(fmt.Sprintf("- `%s`", skill.Name))
		if skill.Description != "" {
			b.WriteString(": ")
			b.WriteString(skill.Description)
		}
		if skillFileHint != "" {
			b.WriteString(fmt.Sprintf(" (SKILL file: `%s`)", skillFileHint))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n---\n\n")
	b.WriteString(userMessage)
	return b.String()
}

// PrepareUserMessage 准备用户消息（为不支持 system prompt 的模型）
func (i *Injector) PrepareUserMessage(message string, skillContext SkillContext) string {
	activeSkills := i.getActiveSkills(skillContext)

	if len(activeSkills) == 0 {
		return message
	}

	// 对于不支持 system prompt 的模型，在 user message 中添加提示
	if !i.capabilities.SupportSystemPrompt {
		return i.InjectToUserMessage(message, activeSkills)
	}

	return message
}

// injectToSystemPrompt 注入到系统提示词
func (i *Injector) injectToSystemPrompt(basePrompt string, skills []*SkillDefinition) string {
	var builder strings.Builder
	builder.WriteString(basePrompt)
	builder.WriteString("\n\n## Active Skills\n\n")
	builder.WriteString("The following skills are installed and enabled for this agent. ")
	builder.WriteString("Each skill's detailed instructions are stored on disk in its `SKILL.md` file under the skills directory. ")
	builder.WriteString("When a skill is relevant, FIRST use filesystem tools (for example the `Read` or `Bash` tools) ")
	builder.WriteString("to open its `SKILL.md`, then follow the instructions and any referenced scripts or resources.\n\n")

	for _, skill := range skills {
		if skill == nil {
			continue
		}

		// 生成 SKILL.md 提示路径（相对于沙箱工作目录）
		skillFileHint := ""
		path := strings.Trim(skill.Path, "/")
		baseDir := strings.Trim(skill.BaseDir, "/")
		if path != "" {
			if baseDir != "" {
				skillFileHint = fmt.Sprintf("%s/%s/SKILL.md", baseDir, path)
			} else {
				skillFileHint = fmt.Sprintf("%s/SKILL.md", path)
			}
		}

		builder.WriteString(fmt.Sprintf("- `%s`", skill.Name))
		if skill.Description != "" {
			builder.WriteString(": ")
			builder.WriteString(skill.Description)
		}
		if skillFileHint != "" {
			builder.WriteString(fmt.Sprintf(" (SKILL file: `%s`)", skillFileHint))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// getActiveSkills 获取应该激活的技能
func (i *Injector) getActiveSkills(context SkillContext) []*SkillDefinition {
	var activeSkills []*SkillDefinition

	log.Printf("[Skills] Total skills loaded: %d", len(i.skills))
	log.Printf("[Skills] Enabled skills map: %v", i.enabledSkills)

	for name, skill := range i.skills {
		enabled := i.enabledSkills[name]
		log.Printf("[Skills] Checking skill: %s (enabled: %v)", name, enabled)
		if !enabled {
			continue
		}
		activeSkills = append(activeSkills, skill)
	}

	return activeSkills
}

// GetActiveSkillNames 获取激活的技能名称列表
func (i *Injector) GetActiveSkillNames(context SkillContext) []string {
	skills := i.getActiveSkills(context)
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		names = append(names, skill.Name)
	}
	return names
}
