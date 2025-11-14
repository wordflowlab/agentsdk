package middleware

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wordflowlab/agentsdk/pkg/memory"
	"github.com/wordflowlab/agentsdk/pkg/tools"
)

// UpdateWorkingMemoryTool Working Memory 更新工具
// 特点：
// - 自动从 ToolContext 获取 threadID 和 resourceID
// - 支持 JSON Schema 验证（如果配置）
// - 支持 Markdown 或 JSON 格式的内容
type UpdateWorkingMemoryTool struct {
	manager *memory.WorkingMemoryManager
	schema  *memory.JSONSchema
}

// NewUpdateWorkingMemoryTool 创建 Working Memory 更新工具
func NewUpdateWorkingMemoryTool(manager *memory.WorkingMemoryManager) tools.Tool {
	return &UpdateWorkingMemoryTool{
		manager: manager,
		schema:  manager.GetSchema(),
	}
}

func (t *UpdateWorkingMemoryTool) Name() string {
	return "update_working_memory"
}

func (t *UpdateWorkingMemoryTool) Description() string {
	desc := "Update the working memory with new information. Any data not included will be overwritten."
	if t.schema != nil {
		desc += " The memory content must match the configured schema."
	}
	return desc
}

func (t *UpdateWorkingMemoryTool) InputSchema() map[string]interface{} {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"memory": map[string]interface{}{
				"type": "string",
			},
		},
		"required": []string{"memory"},
	}

	// 根据配置的 Schema 调整描述
	if t.schema != nil && t.schema.Type == "object" {
		schema["properties"].(map[string]interface{})["memory"].(map[string]interface{})["description"] =
			"The JSON formatted working memory content to store. This MUST be a valid JSON string."
	} else {
		schema["properties"].(map[string]interface{})["memory"].(map[string]interface{})["description"] =
			"The Markdown formatted working memory content to store. This MUST be a string."
	}

	return schema
}

func (t *UpdateWorkingMemoryTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	// 从 input 中获取 memory 内容
	memoryContent, ok := input["memory"].(string)
	if !ok {
		return map[string]interface{}{
			"success": false,
			"error":   "memory field is required and must be a string",
		}, nil
	}

	// 从 ToolContext 获取 threadID 和 resourceID
	threadID := tc.ThreadID
	resourceID := tc.ResourceID

	if threadID == "" && resourceID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "threadID and resourceID cannot both be empty. Please ensure they are set in the context.",
		}, nil
	}

	// 如果配置了 Schema，尝试美化 JSON 格式
	if t.schema != nil && t.schema.Type == "object" {
		// 尝试解析并重新格式化
		var data interface{}
		if err := json.Unmarshal([]byte(memoryContent), &data); err == nil {
			formatted, err := json.MarshalIndent(data, "", "  ")
			if err == nil {
				memoryContent = string(formatted)
			}
		}
	}

	// 更新 Working Memory
	if err := t.manager.Update(ctx, threadID, resourceID, memoryContent); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to update working memory: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success":    true,
		"thread_id":  threadID,
		"resource_id": resourceID,
		"scope":      string(t.manager.GetScope()),
	}, nil
}

func (t *UpdateWorkingMemoryTool) Prompt() string {
	prompt := `Update your working memory to track important information across the conversation.

Working Memory allows you to maintain a persistent state that survives across multiple interactions.
Use this tool to:
- Store user preferences and context
- Track the current state of a task or project
- Remember important decisions and their rationale
- Maintain a structured knowledge base

Guidelines:
- Update working memory whenever you learn something important about the user or task
- Include only relevant information - this will be loaded on every request
- Keep the content concise and well-structured`

	if t.schema != nil {
		prompt += "\n- The content must match the configured JSON schema"
	} else {
		prompt += "\n- Use clear Markdown formatting for better readability"
	}

	return prompt
}

// ExperimentalUpdateWorkingMemoryTool Working Memory 更新工具（实验性，支持 find/replace）
// 对标 Mastra 的 __experimental_updateWorkingMemoryToolVNext
type ExperimentalUpdateWorkingMemoryTool struct {
	manager *memory.WorkingMemoryManager
	schema  *memory.JSONSchema
}

// NewExperimentalUpdateWorkingMemoryTool 创建实验性 Working Memory 更新工具
func NewExperimentalUpdateWorkingMemoryTool(manager *memory.WorkingMemoryManager) tools.Tool {
	return &ExperimentalUpdateWorkingMemoryTool{
		manager: manager,
		schema:  manager.GetSchema(),
	}
}

func (t *ExperimentalUpdateWorkingMemoryTool) Name() string {
	return "update_working_memory_experimental"
}

func (t *ExperimentalUpdateWorkingMemoryTool) Description() string {
	return "Update working memory with support for find/replace operations (experimental feature)."
}

func (t *ExperimentalUpdateWorkingMemoryTool) InputSchema() map[string]interface{} {
	contentDesc := "The Markdown formatted working memory content to store"
	if t.schema != nil && t.schema.Type == "object" {
		contentDesc = "The JSON formatted working memory content to store"
	}

	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"new_memory": map[string]interface{}{
				"type":        "string",
				"description": contentDesc,
			},
			"search_string": map[string]interface{}{
				"type": "string",
				"description": "The working memory string to find and replace. If omitted or doesn't exist, new_memory will be appended. " +
					"Replacing single lines at a time is encouraged for greater accuracy. " +
					"If update_reason is not 'append-new-memory', this field must be provided.",
			},
			"update_reason": map[string]interface{}{
				"type": "string",
				"enum": []string{"append-new-memory", "clarify-existing-memory", "replace-irrelevant-memory"},
				"description": "The reason for updating working memory. Passing any value other than 'append-new-memory' requires a search_string. " +
					"Defaults to 'append-new-memory'.",
			},
		},
		"required": []string{"new_memory"},
	}
}

func (t *ExperimentalUpdateWorkingMemoryTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	newMemory, _ := input["new_memory"].(string)
	searchString, _ := input["search_string"].(string)
	updateReason, _ := input["update_reason"].(string)

	if updateReason == "" {
		updateReason = "append-new-memory"
	}

	threadID := tc.ThreadID
	resourceID := tc.ResourceID

	if threadID == "" && resourceID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "threadID and resourceID cannot both be empty",
		}, nil
	}

	// 规则：resource scope 不允许因"不相关"而替换内容
	if searchString != "" &&
		t.manager.GetScope() == memory.ScopeResource &&
		updateReason == "replace-irrelevant-memory" {
		searchString = "" // 强制改为追加模式
	}

	// append-new-memory 时忽略 search_string
	if updateReason == "append-new-memory" && searchString != "" {
		searchString = ""
	}

	// 非追加模式必须提供 search_string
	if updateReason != "append-new-memory" && searchString == "" {
		return map[string]interface{}{
			"success": false,
			"reason": fmt.Sprintf("update_reason was %s but no search_string was provided. "+
				"Unable to replace undefined with \"%s\"", updateReason, newMemory),
		}, nil
	}

	// 执行更新
	var err error
	if searchString == "" {
		// 追加或完全覆盖
		err = t.manager.Update(ctx, threadID, resourceID, newMemory)
	} else {
		// 查找替换
		err = t.manager.FindAndReplace(ctx, threadID, resourceID, searchString, newMemory)
	}

	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to update working memory: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success":       true,
		"thread_id":     threadID,
		"resource_id":   resourceID,
		"scope":         string(t.manager.GetScope()),
		"update_reason": updateReason,
	}, nil
}

func (t *ExperimentalUpdateWorkingMemoryTool) Prompt() string {
	return `Update working memory with advanced find/replace capabilities (experimental).

This tool supports three update modes:
1. append-new-memory: Add new information to the end (default)
2. clarify-existing-memory: Find and replace existing content to clarify or correct it
3. replace-irrelevant-memory: Replace content that's no longer relevant (thread scope only)

Usage:
- For simple additions, use {"new_memory": "...", "update_reason": "append-new-memory"}
- For updates, provide both new_memory and search_string
- Replace single lines for better accuracy`
}
