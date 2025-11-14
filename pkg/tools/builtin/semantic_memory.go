package builtin

import (
	"context"
	"fmt"

	"github.com/wordflowlab/agentsdk/pkg/memory"
	"github.com/wordflowlab/agentsdk/pkg/tools"
)

// SemanticSearchTool 基于 SemanticMemory 的语义检索工具。
// 输入:
//   {
//     "query": string,
//     "top_k": number (可选, 默认使用 SemanticMemoryConfig.TopK),
//     "metadata": object (可选, 如 {"user_id":"alice","project_id":"demo"})
//   }
// 输出:
//   [
//     {"id": "...", "score": 0.87, "metadata": {...}},
//     ...
//   ]
type SemanticSearchTool struct {
	sm *memory.SemanticMemory
}

func NewSemanticSearchTool(sm *memory.SemanticMemory) tools.Tool {
	return &SemanticSearchTool{sm: sm}
}

func (t *SemanticSearchTool) Name() string {
	return "semantic_search"
}

func (t *SemanticSearchTool) Description() string {
	return "Perform semantic search over indexed texts using a configured vector store and embedder."
}

func (t *SemanticSearchTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Natural language query text.",
			},
			"top_k": map[string]interface{}{
				"type":        "integer",
				"description": "Optional number of results to return.",
			},
			"metadata": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": true,
				"description":          "Optional metadata map (e.g. user_id, project_id) to scope search.",
			},
		},
		"required": []string{"query"},
	}
}

func (t *SemanticSearchTool) Execute(ctx context.Context, input map[string]interface{}, tc *tools.ToolContext) (interface{}, error) {
	if t.sm == nil || !t.sm.Enabled() {
		return nil, fmt.Errorf("semantic memory not configured")
	}

	rawQuery, _ := input["query"].(string)
	if rawQuery == "" {
		return nil, fmt.Errorf("query is required")
	}

	// 可选 top_k
	topK := 0
	if v, ok := input["top_k"].(float64); ok {
		topK = int(v)
	}

	// 可选 metadata
	meta := map[string]interface{}{}
	if m, ok := input["metadata"].(map[string]interface{}); ok && m != nil {
		meta = m
	}

	hits, err := t.sm.Search(ctx, rawQuery, meta, topK)
	if err != nil {
		return nil, err
	}

	// 简单序列化 hits 为 JSON 友好的结构
	out := make([]map[string]interface{}, 0, len(hits))
	for _, h := range hits {
		out = append(out, map[string]interface{}{
			"id":       h.ID,
			"score":    h.Score,
			"metadata": h.Metadata,
		})
	}
	return out, nil
}

func (t *SemanticSearchTool) Prompt() string {
	return "Use this tool to perform semantic search over previously indexed texts when keyword search is insufficient. " +
		"Provide a clear natural language query and optional metadata (user_id, project_id, resource_id) to narrow the search scope."
}

