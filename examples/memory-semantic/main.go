package main

import (
	"context"
	"fmt"

	"github.com/wordflowlab/agentsdk/pkg/memory"
	"github.com/wordflowlab/agentsdk/pkg/vector"
)

// 本示例演示如何使用 SemanticMemory + 内存 VectorStore + MockEmbedder
// 实现一个最简版本的语义检索能力。
//
// 注意: 该示例仅用于演示接口用法, 不适合作为生产环境的 RAG 实现。
func main() {
	ctx := context.Background()

	// 1. 创建向量存储和 embedder
	store := vector.NewMemoryStore()
	embedder := vector.NewMockEmbedder(16)

	// 2. 创建语义记忆组件
	semMem := memory.NewSemanticMemory(memory.SemanticMemoryConfig{
		Store:          store,
		Embedder:       embedder,
		NamespaceScope: "resource",
		TopK:           3,
	})

	// 3. 索引几段示例文本
	docs := []struct {
		id   string
		text string
		meta map[string]interface{}
	}{
		{
			id:   "doc-1",
			text: "Paris is the capital of France.",
			meta: map[string]interface{}{"user_id": "alice", "resource_id": "europe-notes"},
		},
		{
			id:   "doc-2",
			text: "Berlin is the capital of Germany.",
			meta: map[string]interface{}{"user_id": "alice", "resource_id": "europe-notes"},
		},
		{
			id:   "doc-3",
			text: "Tokyo is the capital of Japan.",
			meta: map[string]interface{}{"user_id": "bob", "resource_id": "asia-notes"},
		},
	}

	for _, d := range docs {
		if err := semMem.Index(ctx, d.id, d.text, d.meta); err != nil {
			panic(fmt.Sprintf("index %s: %v", d.id, err))
		}
	}

	// 4. 在 Alice 的 europe-notes 命名空间内进行语义检索
	query := "What is the capital of France?"
	meta := map[string]interface{}{"user_id": "alice", "resource_id": "europe-notes"}

	hits, err := semMem.Search(ctx, query, meta, 3)
	if err != nil {
		panic(fmt.Sprintf("semantic search failed: %v", err))
	}

	fmt.Printf("Query: %q\n", query)
	fmt.Println("Semantic search hits:")
	for _, h := range hits {
		fmt.Printf("  ID=%s, score=%.4f, metadata=%v\n", h.ID, h.Score, h.Metadata)
	}
}

