package memory

import (
	"context"
	"fmt"

	"github.com/wordflowlab/agentsdk/pkg/vector"
)

// SemanticMemoryConfig 语义记忆配置。
// 核心运行时仅依赖接口, 不关心具体 VectorStore/Embedder 实现。
type SemanticMemoryConfig struct {
	Store          vector.VectorStore
	Embedder       vector.Embedder
	NamespaceScope string // "user" | "project" | "resource" | "global"
	TopK           int
}

// SemanticMemory 语义记忆组件, 用于对文本进行向量索引和检索。
// 如果 Store 或 Embedder 为空, 所有方法将成为 no-op。
type SemanticMemory struct {
	cfg SemanticMemoryConfig
}

// NewSemanticMemory 创建语义记忆组件。
func NewSemanticMemory(cfg SemanticMemoryConfig) *SemanticMemory {
	if cfg.TopK <= 0 {
		cfg.TopK = 5
	}
	return &SemanticMemory{cfg: cfg}
}

// Enabled 返回语义记忆是否启用。
func (sm *SemanticMemory) Enabled() bool {
	return sm != nil && sm.cfg.Store != nil && sm.cfg.Embedder != nil
}

// Close 关闭底层 VectorStore。
func (sm *SemanticMemory) Close() error {
	if sm == nil || sm.cfg.Store == nil {
		return nil
	}
	return sm.cfg.Store.Close()
}

// Index 将一段文本写入向量索引。
// docID 应全局唯一, meta 中可包含 user_id/project_id/resource_id 等信息。
func (sm *SemanticMemory) Index(ctx context.Context, docID string, text string, meta map[string]interface{}) error {
	if sm == nil || sm.cfg.Store == nil || sm.cfg.Embedder == nil {
		return nil
	}
	if docID == "" || text == "" {
		return fmt.Errorf("docID and text are required")
	}

	vecs, err := sm.cfg.Embedder.EmbedText(ctx, []string{text})
	if err != nil {
		return fmt.Errorf("embed text: %w", err)
	}
	if len(vecs) == 0 {
		return fmt.Errorf("embedder returned empty vectors")
	}

	// 将文本复制到 metadata 中, 方便检索结果直接携带原文片段。
	metaCopy := make(map[string]interface{}, len(meta)+1)
	for k, v := range meta {
		metaCopy[k] = v
	}
	metaCopy["text"] = text

	return sm.cfg.Store.Upsert(ctx, []vector.Document{
		{
			ID:        docID,
			Text:      text,
			Embedding: vecs[0],
			Metadata:  metaCopy,
			Namespace: sm.namespaceFromMeta(meta),
		},
	})
}

// Search 在指定命名空间内执行向量检索。
// query 为用户自然语言查询, meta 用于构造 namespace/过滤。
func (sm *SemanticMemory) Search(ctx context.Context, query string, meta map[string]interface{}, topK int) ([]vector.Hit, error) {
	if sm == nil || sm.cfg.Store == nil || sm.cfg.Embedder == nil {
		return nil, nil
	}
	if query == "" {
		return nil, nil
	}
	if topK <= 0 {
		topK = sm.cfg.TopK
	}

	vecs, err := sm.cfg.Embedder.EmbedText(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) == 0 {
		return nil, fmt.Errorf("embedder returned empty vectors")
	}

	return sm.cfg.Store.Query(ctx, vector.Query{
		Vector:    vecs[0],
		TopK:      topK,
		Namespace: sm.namespaceFromMeta(meta),
		Filter:    meta,
	})
}

func (sm *SemanticMemory) namespaceFromMeta(meta map[string]interface{}) string {
	if meta == nil {
		return ""
	}
	scope := sm.cfg.NamespaceScope
	switch scope {
	case "user":
		if v, ok := meta["user_id"].(string); ok && v != "" {
			return "users/" + v
		}
	case "project":
		if v, ok := meta["project_id"].(string); ok && v != "" {
			return "projects/" + v
		}
	case "resource":
		user := ""
		if u, ok := meta["user_id"].(string); ok && u != "" {
			user = "users/" + u + "/"
		}
		if pid, ok := meta["project_id"].(string); ok && pid != "" {
			return user + "projects/" + pid
		}
		if rid, ok := meta["resource_id"].(string); ok && rid != "" {
			return user + "resources/" + rid
		}
	}
	// 默认命名空间
	return ""
}
