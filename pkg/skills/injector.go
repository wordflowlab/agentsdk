package skills

import (
	"context"
	"fmt"
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
	// 获取应该激活的技能
	activeSkills := i.getActiveSkills(skillContext)

	if len(activeSkills) == 0 {
		return basePrompt
	}

	// 根据模型能力选择注入方式
	if i.capabilities.SupportSystemPrompt {
		return i.injectToSystemPrompt(basePrompt, activeSkills)
	}

	// 不支持 system prompt，返回原始提示词
	return basePrompt
}

// PrepareUserMessage 准备用户消息（为不支持 system prompt 的模型）
func (i *Injector) PrepareUserMessage(message string, skillContext SkillContext) string {
	activeSkills := i.getActiveSkills(skillContext)

	if len(activeSkills) == 0 {
		return message
	}

	// 对于不支持 system prompt 的模型，在 user message 中添加提示
	if !i.capabilities.SupportSystemPrompt {
		var prefix strings.Builder
		prefix.WriteString("[Active Skills: ")
		for idx, skill := range activeSkills {
			if idx > 0 {
				prefix.WriteString(", ")
			}
			prefix.WriteString(skill.Name)
		}
		prefix.WriteString("]\n\n")
		prefix.WriteString(message)
		return prefix.String()
	}

	return message
}

// injectToSystemPrompt 注入到系统提示词
func (i *Injector) injectToSystemPrompt(basePrompt string, skills []*SkillDefinition) string {
	var builder strings.Builder
	builder.WriteString(basePrompt)
	builder.WriteString("\n\n## Active Skills (Auto-loaded Knowledge)\n\n")
	builder.WriteString("The following skills are automatically activated based on context:\n\n")

	for _, skill := range skills {
		builder.WriteString(fmt.Sprintf("### %s\n\n", skill.Name))
		if skill.Description != "" {
			builder.WriteString(fmt.Sprintf("**Description**: %s\n\n", skill.Description))
		}
		builder.WriteString(skill.KnowledgeBase)
		builder.WriteString("\n\n---\n\n")
	}

	return builder.String()
}

// getActiveSkills 获取应该激活的技能
func (i *Injector) getActiveSkills(context SkillContext) []*SkillDefinition {
	var activeSkills []*SkillDefinition

	for name, skill := range i.skills {
		// 检查是否启用
		if !i.enabledSkills[name] {
			continue
		}

		// 检查触发条件
		if i.shouldActivate(skill, context) {
			activeSkills = append(activeSkills, skill)
		}
	}

	return activeSkills
}

// shouldActivate 检查是否应该激活技能
func (i *Injector) shouldActivate(skill *SkillDefinition, context SkillContext) bool {
	// 如果没有触发条件，默认总是激活
	if len(skill.Triggers) == 0 {
		return true
	}

	for _, trigger := range skill.Triggers {
		switch trigger.Type {
		case "always":
			return true

		case "keyword":
			// 检查关键词
			for _, keyword := range trigger.Keywords {
				if strings.Contains(strings.ToLower(context.UserMessage), strings.ToLower(keyword)) {
					return true
				}
			}

		case "context":
			// 检查上下文条件
			if trigger.Condition != "" {
				if i.matchCondition(trigger.Condition, context) {
					return true
				}
			}
		}
	}

	return false
}

// matchCondition 匹配条件
func (i *Injector) matchCondition(condition string, context SkillContext) bool {
	// 简单的条件匹配
	switch {
	case strings.Contains(condition, "during /write"):
		return context.Command == "/write"
	case strings.Contains(condition, "during /analyze"):
		return context.Command == "/analyze"
	case strings.Contains(condition, "writing"):
		return context.Command == "/write" || strings.Contains(context.UserMessage, "write") || strings.Contains(context.UserMessage, "写")
	default:
		return false
	}
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
