package main

import (
	"context"
	"fmt"

	"github.com/wordflowlab/agentsdk/pkg/evals"
)

// 本示例演示如何在不依赖任何外部服务的情况下,使用 pkg/evals 对模型输出做简单评估。
//
// 包含两个评估维度:
// - KeywordCoverageScorer: 关键词覆盖率
// - LexicalSimilarityScorer: 词汇级相似度(Jaccard)
func main() {
	ctx := context.Background()

	answer := "Paris is the capital of France. It is located in Europe."
	reference := "Paris is the capital city of France, a country in Europe."

	input := &evals.TextEvalInput{
		Answer:    answer,
		Reference: reference,
	}

	// 1. 关键词覆盖率: 检查答案是否提到了关键事实
	kwScorer := evals.NewKeywordCoverageScorer(evals.KeywordCoverageConfig{
		Keywords:        []string{"paris", "capital", "france", "europe"},
		CaseInsensitive: true,
	})

	kwScore, _ := kwScorer.Score(ctx, input)

	fmt.Println("=== Keyword Coverage ===")
	fmt.Printf("Score: %.2f\n", kwScore.Value)
	if matched, ok := kwScore.Details["matched"]; ok {
		fmt.Printf("Matched: %v\n", matched)
	}
	if unmatched, ok := kwScore.Details["unmatched"]; ok {
		fmt.Printf("Unmatched: %v\n", unmatched)
	}
	fmt.Println()

	// 2. 词汇相似度: 粗略衡量答案与参考输出的相似程度
	simScorer := evals.NewLexicalSimilarityScorer(evals.LexicalSimilarityConfig{
		MinTokenLength: 2,
	})

	simScore, _ := simScorer.Score(ctx, input)

	fmt.Println("=== Lexical Similarity ===")
	fmt.Printf("Score: %.2f\n", simScore.Value)
	fmt.Printf("Details: %+v\n", simScore.Details)
}

