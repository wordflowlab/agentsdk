package builtin

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/wordflowlab/agentsdk/pkg/tools"
)

// ReadTool 增强的文件读取工具
// 兼容标准Read工具功能
type ReadTool struct{}

// NewReadTool 创建文件读取工具
func NewReadTool(config map[string]interface{}) (tools.Tool, error) {
	return &ReadTool{}, nil
}

func (t *ReadTool) Name() string {
	return "Read" // 使用标准的工具名称
}

func (t *ReadTool) Description() string {
	return "读取本地文件系统中的文件内容"
}

func (t *ReadTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "要读取的文件路径，必须是绝对路径",
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "读取的起始行号（从1开始计数），默认为1",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "读取的最大行数，默认读取整个文件",
			},
		},
		"required": []string{"file_path"},
	}
}

func (t *ReadTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	// 获取参数
	filePath, ok := input["file_path"].(string)
	if !ok {
		return map[string]interface{}{
			"ok": false,
			"error": "file_path must be a string",
			"recommendations": []string{
				"确保file_path参数为字符串类型",
				"检查参数传递是否正确",
			},
		}, nil
	}

	if filePath == "" {
		return map[string]interface{}{
			"ok": false,
			"error": "file_path cannot be empty",
			"recommendations": []string{
				"提供有效的文件路径",
				"检查文件是否存在",
			},
		}, nil
	}

	// 验证文件路径安全性
	if err := t.validatePath(filePath); err != nil {
		return map[string]interface{}{
			"ok": false,
			"error": fmt.Sprintf("invalid file path: %v", err),
			"recommendations": []string{
				"使用相对路径或允许的绝对路径",
				"确保路径不包含 '..' 避免路径遍历攻击",
			},
		}, nil
	}

	// 获取可选参数
	offset := 0
	if o, ok := input["offset"].(float64); ok {
		offset = int(o)
	}
	if offset < 1 {
		offset = 1
	}

	limit := 0
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
	}

	start := time.Now()

	// 读取文件内容
	content, err := tc.Sandbox.FS().Read(ctx, filePath)
	if err != nil {
		return map[string]interface{}{
			"ok": false,
			"error": fmt.Sprintf("failed to read file: %v", err),
			"recommendations": []string{
				"检查文件路径是否正确",
				"确认文件是否存在",
				"验证是否有读取权限",
				"检查文件是否被其他进程占用",
			},
			"file_path": filePath,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	// 如果文件为空
	if content == "" {
		return map[string]interface{}{
			"ok": true,
			"file_path": filePath,
			"content": "",
			"lines": 0,
			"offset": offset,
			"limit": limit,
			"truncated": false,
			"file_size": 0,
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	// 分割成行
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// 处理offset (转换为0基索引)
	startLine := offset - 1
	if startLine >= totalLines {
		return map[string]interface{}{
			"ok": true,
			"file_path": filePath,
			"content": "",
			"lines": 0,
			"offset": offset,
			"limit": limit,
			"truncated": false,
			"total_lines": totalLines,
			"file_size": len(content),
			"duration_ms": time.Since(start).Milliseconds(),
		}, nil
	}

	if startLine < 0 {
		startLine = 0
	}

	// 应用limit限制
	endLine := totalLines
	truncated := false
	readLines := totalLines - startLine

	if limit > 0 {
		endLine = startLine + limit
		if endLine > totalLines {
			endLine = totalLines
		} else {
			truncated = true
		}
		readLines = limit
	}

	// 提取指定行范围
	selectedLines := lines[startLine:endLine]
	resultContent := strings.Join(selectedLines, "\n")

	// 获取文件信息
	fileSize := len(content)
	fileExt := strings.ToLower(filepath.Ext(filePath))
	fileType := "unknown"

	switch fileExt {
	case ".go":
		fileType = "go"
	case ".js", ".jsx":
		fileType = "javascript"
	case ".ts", ".tsx":
		fileType = "typescript"
	case ".py":
		fileType = "python"
	case ".java":
		fileType = "java"
	case ".cpp", ".cc", ".cxx":
		fileType = "cpp"
	case ".c":
		fileType = "c"
	case ".h", ".hpp":
		fileType = "header"
	case ".md", ".markdown":
		fileType = "markdown"
	case ".json":
		fileType = "json"
	case ".yaml", ".yml":
		fileType = "yaml"
	case ".xml":
		fileType = "xml"
	case ".html", ".htm":
		fileType = "html"
	case ".css":
		fileType = "css"
	case ".sh", ".bash":
		fileType = "shell"
	case ".sql":
		fileType = "sql"
	case ".txt":
		fileType = "text"
	case ".log":
		fileType = "log"
	}

	return map[string]interface{}{
		"ok": true,
		"file_path": filePath,
		"content": resultContent,
		"lines": readLines,
		"offset": offset,
		"limit": limit,
		"truncated": truncated,
		"total_lines": totalLines,
		"file_size": fileSize,
		"file_type": fileType,
		"duration_ms": time.Since(start).Milliseconds(),
	}, nil
}

// validatePath 验证文件路径安全性
func (t *ReadTool) validatePath(path string) error {
	// 清理路径
	cleanPath := filepath.Clean(path)

	// 检查路径遍历攻击
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	return nil
}

func (t *ReadTool) Prompt() string {
	return `读取本地文件系统中的文件内容。

功能特性：
- 支持偏移量和行数限制的分页读取
- 自动文件类型识别
- 安全的路径验证
- 详细的执行时间统计

使用指南：
- file_path: 必需参数，要读取的文件路径
- offset: 可选参数，起始行号（从1开始）
- limit: 可选参数，最大读取行数

安全性：
- 路径遍历攻击防护
- 沙箱环境隔离
- 权限检查集成`
}
