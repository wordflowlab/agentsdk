package commands

// CommandDefinition Slash Command 定义
type CommandDefinition struct {
	// 基础信息
	Name         string   // 命令名（如 "write"）
	Description  string   // 描述
	ArgumentHint string   // 参数提示
	AllowedTools []string // 允许使用的工具

	// 模型要求
	Models struct {
		Preferred           []string // 推荐的模型
		MinimumCapabilities []string // 最小能力要求
	}

	// 脚本路径
	Scripts struct {
		Sh string // bash 脚本路径
		Ps string // powershell 脚本路径
	}

	// 提示词模板
	PromptTemplate string // Markdown 内容
}
