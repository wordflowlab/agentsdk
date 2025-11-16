package builtin

import "github.com/wordflowlab/agentsdk/pkg/tools"

// RegisterAll 注册所有内置工具
func RegisterAll(registry *tools.Registry) {
	// 文件系统工具
	registry.Register("fs_read", NewFsReadTool)
	registry.Register("fs_write", NewFsWriteTool)

	// Bash工具
	registry.Register("bash_run", NewBashRunTool)

	// 网络工具 (Phase 6B-1)
	registry.Register("http_request", NewHttpRequestTool)
	registry.Register("web_search", NewWebSearchTool)

	// Skills 工具 (需要显式在模板中启用)
	registry.Register("skill_call", NewSkillCallTool)
}

// FileSystemTools 返回文件系统工具列表
func FileSystemTools() []string {
	return []string{"fs_read", "fs_write"}
}

// BashTools 返回Bash工具列表
func BashTools() []string {
	return []string{"bash_run"}
}

// NetworkTools 返回网络工具列表
func NetworkTools() []string {
	return []string{"http_request", "web_search"}
}

// AllTools 返回所有内置工具列表
func AllTools() []string {
	tools := append(FileSystemTools(), BashTools()...)
	tools = append(tools, NetworkTools()...)
	tools = append(tools, "skill_call")
	return tools
}
