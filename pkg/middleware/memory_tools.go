package middleware

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wordflowlab/agentsdk/pkg/memory"
	"github.com/wordflowlab/agentsdk/pkg/tools"
)

// createMemoryTools 基于 AgentMemoryMiddleware 的 backend/memoryPath 创建记忆相关工具
func (m *AgentMemoryMiddleware) createMemoryTools() []tools.Tool {
	manager, err := memory.NewManager(&memory.ManagerConfig{
		Backend:    m.backend,
		MemoryPath: m.memoryPath,
	})
	if err != nil {
		// 如果初始化失败,记录日志,但不中断整体中间件创建
		// 工具列表留空即可
		return nil
	}

	return []tools.Tool{
		NewMemorySearchTool(manager, m.baseNamespace),
		NewMemoryWriteTool(manager, m.baseNamespace),
	}
}

// MemorySearchTool 基于 Manager 的记忆搜索工具
// 特点:
// - 仅在 memoryPath 下搜索,避免扫描整个文件系统
// - 默认使用大小写不敏感的字面量匹配
type MemorySearchTool struct {
	manager       *memory.Manager
	baseNamespace string
}

// NewMemorySearchTool 创建搜索工具
func NewMemorySearchTool(manager *memory.Manager, baseNamespace string) tools.Tool {
	return &MemorySearchTool{
		manager:       manager,
		baseNamespace: strings.TrimSpace(baseNamespace),
	}
}

func (t *MemorySearchTool) Name() string {
	return "memory_search"
}

func (t *MemorySearchTool) Description() string {
	return "Search long-term memory files using grep-style matching within the configured memory path."
}

func (t *MemorySearchTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Text to search for (case-insensitive by default).",
			},
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "Optional logical namespace, e.g. \"users/alice\" or \"projects/demo\". Limits search to that subtree.",
			},
			"regex": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, treat query as a raw regular expression.",
			},
			"glob": map[string]interface{}{
				"type":        "string",
				"description": "Optional glob filter for files, e.g. \"*.md\".",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of matches to return (default 20).",
			},
		},
		"required": []string{"query"},
	}
}

func (t *MemorySearchTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	query, _ := input["query"].(string)

	rawNamespace, _ := input["namespace"].(string)

	regex, _ := input["regex"].(bool)

	glob, _ := input["glob"].(string)

	maxResults := 20
	if v, ok := input["max_results"].(float64); ok && int(v) > 0 {
		maxResults = int(v)
	}

	// 组合基础命名空间 + 调用命名空间
	// 规则:
	// - 如果 namespace 以 "/" 开头, 视为全局命名空间, 不叠加 baseNamespace
	// - 否则在 baseNamespace 下叠加
	ns := strings.TrimSpace(rawNamespace)
	if strings.HasPrefix(ns, "/") {
		ns = strings.TrimPrefix(ns, "/")
	} else if t.baseNamespace != "" {
		if ns != "" {
			ns = filepath.ToSlash(filepath.Join(t.baseNamespace, ns))
		} else {
			ns = t.baseNamespace
		}
	}

	matches, err := t.manager.Search(ctx, &memory.SearchOptions{
		Query:      query,
		Regex:      regex,
		Namespace:  ns,
		Glob:       glob,
		MaxResults: maxResults,
	})
	if err != nil {
		return map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("memory search failed: %v", err),
		}, nil
	}

	items := make([]map[string]interface{}, 0, len(matches))
	for _, m := range matches {
		items = append(items, map[string]interface{}{
			"path":        m.Path,
			"line_number": m.LineNumber,
			"line":        m.Line,
			"match":       m.Match,
		})
	}

	return map[string]interface{}{
		"ok":          true,
		"query":       query,
		"namespace":   ns,
		"regex":       regex,
		"glob":        glob,
		"count":       len(items),
		"memory_path": t.manager.MemoryPath(),
		"matches":     items,
	}, nil
}

func (t *MemorySearchTool) Prompt() string {
	return `Search your long-term memory files without using vector databases.

Guidelines:
- This tool searches only within the configured memory directory (e.g., /memories/).
- By default, the search is case-insensitive and treats the query as plain text.
- Set "regex": true to use full regular expressions when needed.
- Use "glob" to narrow down files, e.g. {"query": "deploy", "glob": "*.md"}.

Typical uses:
- Find previously stored user preferences or decisions.
- Locate project-specific documentation and notes.
- Retrieve past summaries or checklists you wrote earlier.`
}

// MemoryWriteTool 记忆写入/追加工具
// 通过普通文本写入而不是向量数据库,保持可读可编辑
type MemoryWriteTool struct {
	manager       *memory.Manager
	baseNamespace string
}

// NewMemoryWriteTool 创建写入工具
func NewMemoryWriteTool(manager *memory.Manager, baseNamespace string) tools.Tool {
	return &MemoryWriteTool{
		manager:       manager,
		baseNamespace: strings.TrimSpace(baseNamespace),
	}
}

func (t *MemoryWriteTool) Name() string {
	return "memory_write"
}

func (t *MemoryWriteTool) Description() string {
	return "Append or overwrite long-term memory notes stored as plaintext/Markdown files."
}

func (t *MemoryWriteTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file": map[string]interface{}{
				"type":        "string",
				"description": "Memory file name relative to the memory root, e.g. \"project_notes.md\".",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Note content to write.",
			},
			"namespace": map[string]interface{}{
				"type":        "string",
				"description": "Optional logical namespace, e.g. \"users/alice\" or \"projects/demo\". The file will be created under this subtree.",
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "Write mode: \"append\" (default) or \"overwrite\".",
				"enum":        []interface{}{"append", "overwrite"},
			},
			"title": map[string]interface{}{
				"type":        "string",
				"description": "Optional title for the note section (used in append mode).",
			},
		},
		"required": []string{"file", "content"},
	}
}

func (t *MemoryWriteTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	file, _ := input["file"].(string)
	content, _ := input["content"].(string)
	mode, _ := input["mode"].(string)
	title, _ := input["title"].(string)
	rawNamespace, _ := input["namespace"].(string)

	if mode == "" {
		mode = "append"
	}

	// 组合基础命名空间 + 调用命名空间
	// 规则:
	// - 如果 namespace 以 "/" 开头, 视为全局命名空间, 不叠加 baseNamespace
	// - 否则在 baseNamespace 下叠加
	ns := strings.TrimSpace(rawNamespace)
	if strings.HasPrefix(ns, "/") {
		ns = strings.TrimPrefix(ns, "/")
	} else if t.baseNamespace != "" {
		if ns != "" {
			ns = filepath.ToSlash(filepath.Join(t.baseNamespace, ns))
		} else {
			ns = t.baseNamespace
		}
	}

	// 将命名空间与 file 组合成相对 MemoryPath 的完整路径
	combinedFile := strings.TrimSpace(file)
	if ns != "" {
		combinedFile = filepath.ToSlash(filepath.Join(ns, combinedFile))
	}

	switch mode {
	case "append":
		path, err := t.manager.AppendNote(ctx, combinedFile, title, content)
		if err != nil {
			return map[string]interface{}{
				"ok":    false,
				"error": fmt.Sprintf("append note failed: %v", err),
			}, nil
		}

		return map[string]interface{}{
			"ok":          true,
			"mode":        "append",
			"path":        path,
			"namespace":   ns,
			"memory_path": t.manager.MemoryPath(),
			"message":     "Note appended to memory file successfully.",
		}, nil

	case "overwrite":
		path, err := t.manager.OverwriteWithNote(ctx, combinedFile, title, content)
		if err != nil {
			return map[string]interface{}{
				"ok":    false,
				"error": fmt.Sprintf("overwrite note failed: %v", err),
			}, nil
		}
		return map[string]interface{}{
			"ok":          true,
			"mode":        "overwrite",
			"path":        path,
			"namespace":   ns,
			"memory_path": t.manager.MemoryPath(),
			"message":     "Memory file overwritten with new note section.",
		}, nil

	default:
		return map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("unsupported mode: %s (expected \"append\" or \"overwrite\")", mode),
		}, nil
	}
}

func (t *MemoryWriteTool) Prompt() string {
	return `Write stable, human-readable long-term memory without using any vector database.

Guidelines:
- Use this tool to store durable preferences, decisions, and summaries.
- Always choose a descriptive file name, e.g. "user_profile.md" or "project_X_notes.md".
- Prefer APPEND mode to preserve history; use OVERWRITE only for small canonical facts.
- In append mode, set "title" to describe the note (e.g., "2025-01-10: Deployment checklist").

Examples:
- Append a new learning: {"file": "project_X_notes.md", "mode": "append", "title": "Postmortem", "content": "..."}
- Store user preferences: {"file": "user/alice.md", "content": "Alice prefers concise code reviews."}`
}
