package skills

// SkillDefinition Skill 定义
type SkillDefinition struct {
	// 基础信息
	Name         string   // 技能名
	Description  string   // 描述
	AllowedTools []string // 允许使用的工具

	// 知识库内容
	KnowledgeBase string // 注入到 SystemPrompt 的内容

	// 触发条件
	Triggers []TriggerConfig
}

// TriggerConfig 触发配置
type TriggerConfig struct {
	Type      string   // "keyword" | "context" | "always"
	Keywords  []string // 关键词列表
	Condition string   // 条件描述
}

// SkillContext 技能上下文
type SkillContext struct {
	UserMessage string                 // 用户输入
	Command     string                 // 当前命令（如 "/write"）
	Files       []string               // 涉及的文件
	Metadata    map[string]interface{} // 额外元数据
}
