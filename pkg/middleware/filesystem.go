package middleware

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/wordflowlab/agentsdk/pkg/backends"
	"github.com/wordflowlab/agentsdk/pkg/tools"
	"github.com/wordflowlab/agentsdk/pkg/tools/builtin"
)

// FilesystemMiddlewareConfig 文件系统中间件配置
type FilesystemMiddlewareConfig struct {
	Backend            backends.BackendProtocol // 后端存储
	TokenLimit         int                      // 大结果驱逐阈值(tokens)
	EnableEviction     bool                     // 是否启用自动驱逐
	AllowedPathPrefixes []string                 // 允许的路径前缀列表(用于路径安全验证)
	EnablePathValidation bool                    // 是否启用路径验证(默认: true)
	CustomToolDescriptions map[string]string     // 自定义工具描述
	SystemPromptOverride string                  // 覆盖默认系统提示词
}

// FilesystemMiddleware 文件系统中间件
// 功能:
// 1. 注入文件系统工具 (ls, read, write, edit, glob, grep)
// 2. 自动驱逐大结果到文件
// 3. 增强系统提示词
// 4. 路径安全验证
type FilesystemMiddleware struct {
	*BaseMiddleware
	backend              backends.BackendProtocol
	tokenLimit           int
	enableEviction       bool
	allowedPathPrefixes  []string
	enablePathValidation bool
	customToolDescriptions map[string]string
	systemPromptOverride string
	fsTools              []tools.Tool
}

// NewFilesystemMiddleware 创建文件系统中间件
func NewFilesystemMiddleware(config *FilesystemMiddlewareConfig) *FilesystemMiddleware {
	if config.TokenLimit == 0 {
		config.TokenLimit = 20000 // 默认 20k tokens
	}

	// 默认启用路径验证
	enablePathValidation := true
	if config.EnablePathValidation == false && len(config.AllowedPathPrefixes) == 0 {
		// 只有在明确禁用且没有指定前缀时才不验证
		enablePathValidation = false
	}

	m := &FilesystemMiddleware{
		BaseMiddleware:         NewBaseMiddleware("filesystem", 100),
		backend:                config.Backend,
		tokenLimit:             config.TokenLimit,
		enableEviction:         config.EnableEviction,
		allowedPathPrefixes:    config.AllowedPathPrefixes,
		enablePathValidation:   enablePathValidation,
		customToolDescriptions: config.CustomToolDescriptions,
		systemPromptOverride:   config.SystemPromptOverride,
	}

	// 创建文件系统工具
	m.fsTools = m.createFilesystemTools()

	log.Printf("[FilesystemMiddleware] Path validation: %v, Allowed prefixes: %v",
		m.enablePathValidation, m.allowedPathPrefixes)
	return m
}

// createFilesystemTools 创建文件系统工具
func (m *FilesystemMiddleware) createFilesystemTools() []tools.Tool {
	var fsTools []tools.Tool

	// 基础工具: fs_read, fs_write
	if tool, err := builtin.NewFsReadTool(nil); err == nil {
		fsTools = append(fsTools, tool)
	}
	if tool, err := builtin.NewFsWriteTool(nil); err == nil {
		fsTools = append(fsTools, tool)
	}

	// 如果有 backend,添加增强工具
	if m.backend != nil {
		// fs_ls
		fsTools = append(fsTools, &FsLsTool{backend: m.backend, middleware: m})
		// fs_edit
		fsTools = append(fsTools, &FsEditTool{backend: m.backend, middleware: m})
		// fs_glob
		fsTools = append(fsTools, &FsGlobTool{backend: m.backend, middleware: m})
		// fs_grep
		fsTools = append(fsTools, &FsGrepTool{backend: m.backend, middleware: m})
	}

	log.Printf("[FilesystemMiddleware] Created %d filesystem tools", len(fsTools))
	return fsTools
}

// Tools 返回文件系统工具
func (m *FilesystemMiddleware) Tools() []tools.Tool {
	return m.fsTools
}

// WrapModelCall 包装模型调用
func (m *FilesystemMiddleware) WrapModelCall(ctx context.Context, req *ModelRequest, handler ModelCallHandler) (*ModelResponse, error) {
	// 使用自定义系统提示词或默认提示词
	prompt := FILESYSTEM_SYSTEM_PROMPT
	if m.systemPromptOverride != "" {
		prompt = m.systemPromptOverride
	}

	// 增强系统提示词
	if req.SystemPrompt != "" {
		req.SystemPrompt += "\n\n" + prompt
	} else {
		req.SystemPrompt = prompt
	}

	// 调用下一层
	return handler(ctx, req)
}

// WrapToolCall 包装工具调用
func (m *FilesystemMiddleware) WrapToolCall(ctx context.Context, req *ToolCallRequest, handler ToolCallHandler) (*ToolCallResponse, error) {
	// 执行工具调用
	resp, err := handler(ctx, req)
	if err != nil {
		return resp, err
	}

	// 检查是否需要驱逐大结果
	if m.enableEviction && m.backend != nil {
		resp = m.evictLargeResults(ctx, req, resp)
	}

	return resp, nil
}

// evictLargeResults 驱逐大结果到文件
func (m *FilesystemMiddleware) evictLargeResults(ctx context.Context, req *ToolCallRequest, resp *ToolCallResponse) *ToolCallResponse {
	// 简单估算: 1 token ≈ 4 chars
	resultStr := fmt.Sprintf("%v", resp.Result)
	estimatedTokens := len(resultStr) / 4

	if estimatedTokens > m.tokenLimit {
		// 生成文件路径
		path := fmt.Sprintf("/large_tool_results/%s.txt", req.ToolCallID)

		// 写入文件
		if _, err := m.backend.Write(ctx, path, resultStr); err == nil {
			// 返回简化结果
			lines := splitLines(resultStr, 10)
			preview := ""
			if len(lines) > 10 {
				preview = joinLines(lines[:10])
			} else {
				preview = resultStr
			}

			resp.Result = map[string]interface{}{
				"ok":      true,
				"evicted": true,
				"path":    path,
				"message": fmt.Sprintf("Result too large (%d tokens), saved to %s", estimatedTokens, path),
				"preview": preview,
			}

			log.Printf("[FilesystemMiddleware] Evicted large result (%d tokens) to %s", estimatedTokens, path)
		}
	}

	return resp
}

// FILESYSTEM_SYSTEM_PROMPT 文件系统提示词
const FILESYSTEM_SYSTEM_PROMPT = `### Filesystem Tools

You have access to the following filesystem tools:

- **fs_read**: Read file contents with optional offset/limit
- **fs_write**: Write content to a file
- **fs_ls**: List directory contents
- **fs_edit**: Edit files using string replacement
- **fs_glob**: Find files matching glob patterns
- **fs_grep**: Search for patterns in files

Guidelines:
- Always use relative paths from the sandbox root
- Large results will be automatically saved to files
- Use fs_edit for precise modifications
- Use fs_glob and fs_grep for code exploration`

// 辅助函数
func splitLines(s string, limit int) []string {
	lines := []string{}
	current := ""
	for _, r := range s {
		current += string(r)
		if r == '\n' {
			lines = append(lines, current)
			current = ""
			if len(lines) >= limit {
				break
			}
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for _, line := range lines {
		result += line
	}
	return result
}

// validatePath 验证路径安全性
// 参考: deepagents/filesystem.py:87-129
func (m *FilesystemMiddleware) validatePath(path string) (string, error) {
	if !m.enablePathValidation {
		return path, nil
	}

	// 1. 检查路径遍历攻击
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("路径遍历不允许(包含 '..'): %s", path)
	}

	if strings.HasPrefix(path, "~") {
		return "", fmt.Errorf("路径遍历不允许(以 '~' 开头): %s", path)
	}

	// 2. 规范化路径
	normalized := filepath.Clean(path)
	// 转换为 Unix 风格路径(统一使用 /)
	normalized = filepath.ToSlash(normalized)

	// 3. 确保路径以 / 开头
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}

	// 4. 检查允许的前缀
	if len(m.allowedPathPrefixes) > 0 {
		allowed := false
		for _, prefix := range m.allowedPathPrefixes {
			// 规范化前缀(去掉尾部斜杠)
			normalizedPrefix := filepath.Clean(prefix)
			normalizedPrefix = filepath.ToSlash(normalizedPrefix)
			if !strings.HasPrefix(normalizedPrefix, "/") {
				normalizedPrefix = "/" + normalizedPrefix
			}

			// 检查前缀匹配(normalized == prefix 或 normalized 在 prefix 下)
			if normalized == normalizedPrefix || strings.HasPrefix(normalized, normalizedPrefix+"/") {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("路径必须以以下前缀之一开头 %v: %s", m.allowedPathPrefixes, normalized)
		}
	}

	return normalized, nil
}
