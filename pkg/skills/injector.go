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
	// 获取应该激活的技能
	activeSkills := i.getActiveSkills(skillContext)

	log.Printf("[Skills] Checking activation for message: %q", skillContext.UserMessage)
	log.Printf("[Skills] Found %d active skills", len(activeSkills))
	for _, skill := range activeSkills {
		log.Printf("[Skills] - Activated: %s (%s)", skill.Name, skill.Description)
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
func (i *Injector) InjectToUserMessage(userMessage string, skills []*SkillDefinition) string {
	if len(skills) == 0 {
		return userMessage
	}

	var b strings.Builder
	b.WriteString("## Knowledge Base\n\n")

	for _, skill := range skills {
		b.WriteString(fmt.Sprintf("### %s\n\n", skill.Name))
		if skill.Description != "" {
			b.WriteString(fmt.Sprintf("**Description**: %s\n\n", skill.Description))
		}
		if skill.KnowledgeBase != "" {
			b.WriteString(skill.KnowledgeBase)
			b.WriteString("\n\n")
		}
	}

	b.WriteString("---\n\n")
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

	log.Printf("[Skills] Total skills loaded: %d", len(i.skills))
	log.Printf("[Skills] Enabled skills: %v", i.enabledSkills)

	for name, skill := range i.skills {
		log.Printf("[Skills] Checking skill: %s (enabled: %v)", name, i.enabledSkills[name])

		// 检查是否启用
		if !i.enabledSkills[name] {
			log.Printf("[Skills] - Skipping %s: not enabled", name)
			continue
		}

		// 检查触发条件
		if i.shouldActivate(skill, context) {
			log.Printf("[Skills] - Activating %s: trigger matched", name)
			activeSkills = append(activeSkills, skill)
		} else {
			log.Printf("[Skills] - Skipping %s: no trigger matched", name)
		}
	}

	return activeSkills
}

// shouldActivate 检查是否应该激活技能
func (i *Injector) shouldActivate(skill *SkillDefinition, context SkillContext) bool {
	// 如果没有触发条件，默认总是激活
	if len(skill.Triggers) == 0 {
		log.Printf("[Skills] - Skill %s: no triggers, always activating", skill.Name)
		return true
	}

	log.Printf("[Skills] - Skill %s: checking %d triggers", skill.Name, len(skill.Triggers))
	for triggerIndex, trigger := range skill.Triggers {
		log.Printf("[Skills] - Trigger %d: type=%s", triggerIndex, trigger.Type)
		switch trigger.Type {
		case "always":
			log.Printf("[Skills] - Always trigger matched for %s", skill.Name)
			return true

		case "keyword":
			log.Printf("[Skills] - Checking keywords: %v", trigger.Keywords)
			// 检查关键词
			for _, keyword := range trigger.Keywords {
				lowerKeyword := strings.ToLower(keyword)
				lowerMessage := strings.ToLower(context.UserMessage)
				matched := strings.Contains(lowerMessage, lowerKeyword)
				log.Printf("[Skills] - Keyword '%s' matched: %v", keyword, matched)
				if matched {
					log.Printf("[Skills] - Keyword trigger matched for %s", skill.Name)
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

		case "file_pattern":
			if trigger.Pattern != "" && len(context.Files) > 0 {
				pat := strings.TrimSpace(trigger.Pattern)
				// 简化实现：如果模式中包含 "*"，去掉 "*" 后按子串匹配；
				// 否则直接按子串匹配。
				raw := strings.ReplaceAll(pat, "*", "")
				if raw == "" {
					continue
				}
				for _, f := range context.Files {
					if strings.Contains(f, raw) {
						return true
					}
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
