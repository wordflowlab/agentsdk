package middleware

import (
	"context"
	"fmt"

	"github.com/wordflowlab/agentsdk/pkg/backends"
	"github.com/wordflowlab/agentsdk/pkg/tools"
)

// FsLsTool 目录列表工具
type FsLsTool struct {
	backend    backends.BackendProtocol
	middleware *FilesystemMiddleware
}

func (t *FsLsTool) Name() string {
	return "fs_ls"
}

func (t *FsLsTool) Description() string {
	if t.middleware != nil && t.middleware.customToolDescriptions != nil {
		if customDesc, ok := t.middleware.customToolDescriptions["fs_ls"]; ok {
			return customDesc
		}
	}
	return "List directory contents with detailed file information"
}

func (t *FsLsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to list (default: current directory)",
			},
		},
	}
}

func (t *FsLsTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	path := "."
	if p, ok := input["path"].(string); ok {
		path = p
	}

	// 路径安全验证
	if t.middleware != nil {
		validatedPath, err := t.middleware.validatePath(path)
		if err != nil {
			return map[string]interface{}{
				"ok":    false,
				"error": fmt.Sprintf("path validation failed: %v", err),
			}, nil
		}
		path = validatedPath
	}

	results, err := t.backend.ListInfo(ctx, path)
	if err != nil {
		return map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("failed to list directory: %v", err),
		}, nil
	}

	items := make([]map[string]interface{}, 0, len(results))
	for _, info := range results {
		items = append(items, map[string]interface{}{
			"path":      info.Path,
			"is_dir":    info.IsDirectory,
			"size":      info.Size,
			"modified":  info.ModifiedTime.Format("2006-01-02 15:04:05"),
		})
	}

	return map[string]interface{}{
		"ok":    true,
		"path":  path,
		"items": items,
		"count": len(items),
	}, nil
}

func (t *FsLsTool) Prompt() string {
	return `Use this tool to list directory contents with detailed file information.

Features:
- Shows file size, modification time, and type (file/directory)
- Supports both relative and absolute paths
- Returns structured data for easy parsing

Example usage:
- List current directory: {"path": "."}
- List specific directory: {"path": "src/components"}`
}

// FsEditTool 文件编辑工具
type FsEditTool struct {
	backend    backends.BackendProtocol
	middleware *FilesystemMiddleware
}

func (t *FsEditTool) Name() string {
	return "fs_edit"
}

func (t *FsEditTool) Description() string {
	if t.middleware != nil && t.middleware.customToolDescriptions != nil {
		if customDesc, ok := t.middleware.customToolDescriptions["fs_edit"]; ok {
			return customDesc
		}
	}
	return "Edit files using precise string replacement"
}

func (t *FsEditTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to edit",
			},
			"old_string": map[string]interface{}{
				"type":        "string",
				"description": "String to replace",
			},
			"new_string": map[string]interface{}{
				"type":        "string",
				"description": "Replacement string",
			},
			"replace_all": map[string]interface{}{
				"type":        "boolean",
				"description": "Replace all occurrences (default: false, replace only first)",
			},
		},
		"required": []string{"path", "old_string", "new_string"},
	}
}

func (t *FsEditTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	path, _ := input["path"].(string)
	oldStr, _ := input["old_string"].(string)
	newStr, _ := input["new_string"].(string)
	replaceAll := false
	if ra, ok := input["replace_all"].(bool); ok {
		replaceAll = ra
	}

	// 路径安全验证
	if t.middleware != nil {
		validatedPath, err := t.middleware.validatePath(path)
		if err != nil {
			return map[string]interface{}{
				"ok":    false,
				"error": fmt.Sprintf("path validation failed: %v", err),
			}, nil
		}
		path = validatedPath
	}

	result, err := t.backend.Edit(ctx, path, oldStr, newStr, replaceAll)
	if err != nil {
		return map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("failed to edit file: %v", err),
		}, nil
	}

	// 检查 Error 字段判断是否成功
	if result.Error != "" {
		return map[string]interface{}{
			"ok":    false,
			"error": result.Error,
		}, nil
	}

	return map[string]interface{}{
		"ok":           true,
		"path":         path,
		"replacements": result.ReplacementsMade,
		"replace_all":  replaceAll,
	}, nil
}

func (t *FsEditTool) Prompt() string {
	return `Use this tool for precise file editing via string replacement.

Guidelines:
- ALWAYS read the file first with fs_read to ensure you have the exact string to replace
- Use replace_all=true to replace all occurrences, false for just the first
- The old_string must match exactly (including whitespace)
- Prefer this over fs_write when making small, targeted changes

Safety:
- Validates that old_string exists before replacing
- Returns the number of replacements made
- Atomic operation (all-or-nothing)`
}

// FsGlobTool Glob 模式匹配工具
type FsGlobTool struct {
	backend    backends.BackendProtocol
	middleware *FilesystemMiddleware
}

func (t *FsGlobTool) Name() string {
	return "fs_glob"
}

func (t *FsGlobTool) Description() string {
	if t.middleware != nil && t.middleware.customToolDescriptions != nil {
		if customDesc, ok := t.middleware.customToolDescriptions["fs_glob"]; ok {
			return customDesc
		}
	}
	return "Find files matching glob patterns (e.g., **/*.go, src/**/*.ts)"
}

func (t *FsGlobTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Glob pattern (supports *, **, ?)",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Base path to search from (default: .)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *FsGlobTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	pattern, _ := input["pattern"].(string)
	path := "."
	if p, ok := input["path"].(string); ok {
		path = p
	}

	// 路径安全验证
	if t.middleware != nil {
		validatedPath, err := t.middleware.validatePath(path)
		if err != nil {
			return map[string]interface{}{
				"ok":    false,
				"error": fmt.Sprintf("path validation failed: %v", err),
			}, nil
		}
		path = validatedPath
	}

	results, err := t.backend.GlobInfo(ctx, pattern, path)
	if err != nil {
		return map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("failed to glob: %v", err),
		}, nil
	}

	files := make([]string, 0, len(results))
	for _, info := range results {
		files = append(files, info.Path)
	}

	return map[string]interface{}{
		"ok":      true,
		"pattern": pattern,
		"files":   files,
		"count":   len(files),
	}, nil
}

func (t *FsGlobTool) Prompt() string {
	return `Use this tool to find files matching glob patterns.

Pattern syntax:
- * matches any characters except /
- ** matches any characters including /
- ? matches a single character

Examples:
- Find all Go files: {"pattern": "**/*.go"}
- Find TypeScript in src: {"pattern": "*.ts", "path": "src"}
- Find test files: {"pattern": "**/*_test.go"}

Use cases:
- Discovering project structure
- Finding files to edit
- Locating configuration files`
}

// FsGrepTool 正则搜索工具
type FsGrepTool struct {
	backend    backends.BackendProtocol
	middleware *FilesystemMiddleware
}

func (t *FsGrepTool) Name() string {
	return "fs_grep"
}

func (t *FsGrepTool) Description() string {
	if t.middleware != nil && t.middleware.customToolDescriptions != nil {
		if customDesc, ok := t.middleware.customToolDescriptions["fs_grep"]; ok {
			return customDesc
		}
	}
	return "Search for regex patterns in files, similar to grep"
}

func (t *FsGrepTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Regular expression pattern to search for",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to search in (default: .)",
			},
			"glob": map[string]interface{}{
				"type":        "string",
				"description": "File filter pattern (e.g., *.go)",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *FsGrepTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	pattern, _ := input["pattern"].(string)
	path := "."
	if p, ok := input["path"].(string); ok {
		path = p
	}
	glob := ""
	if g, ok := input["glob"].(string); ok {
		glob = g
	}

	// 路径安全验证
	if t.middleware != nil {
		validatedPath, err := t.middleware.validatePath(path)
		if err != nil {
			return map[string]interface{}{
				"ok":    false,
				"error": fmt.Sprintf("path validation failed: %v", err),
			}, nil
		}
		path = validatedPath
	}

	matches, err := t.backend.GrepRaw(ctx, pattern, path, glob)
	if err != nil {
		return map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("failed to grep: %v", err),
		}, nil
	}

	results := make([]map[string]interface{}, 0, len(matches))
	for _, match := range matches {
		results = append(results, map[string]interface{}{
			"path":        match.Path,
			"line_number": match.LineNumber,
			"line":        match.Line,
			"match":       match.Match,
		})
	}

	return map[string]interface{}{
		"ok":      true,
		"pattern": pattern,
		"matches": results,
		"count":   len(results),
	}, nil
}

func (t *FsGrepTool) Prompt() string {
	return `Use this tool to search for patterns across files.

Features:
- Full regex support
- Shows line numbers and context
- Filter by file type using glob
- Fast search across large codebases

Examples:
- Find function definitions: {"pattern": "func \\w+\\("}
- Find TODOs: {"pattern": "TODO:"}
- Find in Go files only: {"pattern": "error", "glob": "*.go"}

Use cases:
- Finding where code is used
- Locating error messages
- Discovering API endpoints
- Understanding code structure`
}
