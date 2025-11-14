package vector

import "context"

// Embedder 为文本生成向量的抽象接口。
// 具体实现可以基于 OpenAI、Anthropic、本地模型或外部 HTTP 服务。
type Embedder interface {
	EmbedText(ctx context.Context, texts []string) ([][]float32, error)
}

