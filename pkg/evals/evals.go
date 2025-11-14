package evals

import (
	"context"
	"strings"
	"unicode"
)

// ScoreResult 评估结果
type ScoreResult struct {
	// Name 评估名称,如 "keyword_coverage", "lexical_similarity"
	Name string `json:"name"`
	// Value 得分,范围通常在 [0,1]
	Value float64 `json:"value"`
	// Details 额外信息,如匹配到的关键词列表
	Details map[string]interface{} `json:"details,omitempty"`
}

// TextEvalInput 文本评估输入
type TextEvalInput struct {
	// Answer 待评估的模型输出
	Answer string `json:"answer"`
	// Context 可选的上下文(如参考资料、提示信息等)
	Context []string `json:"context,omitempty"`
	// Reference 可选参考答案/期望输出,用于相似度比较
	Reference string `json:"reference,omitempty"`
}

// Scorer 文本评估器接口。
// 设计参考: @mastra/evals, 但本实现仅提供本地启发式 scorer,不依赖 LLM。
type Scorer interface {
	Score(ctx context.Context, input *TextEvalInput) (*ScoreResult, error)
}

// =========================
// 1. 关键词覆盖率评估器
// =========================

// KeywordCoverageConfig 关键词覆盖率配置
type KeywordCoverageConfig struct {
	// Keywords 需要在答案中出现的关键短语
	Keywords []string
	// CaseInsensitive 是否大小写不敏感(默认: true)
	CaseInsensitive bool
}

// KeywordCoverageScorer 根据关键词覆盖率对答案打分。
// 得分 = 覆盖到的关键词数量 / 总关键词数量, 范围 [0,1]。
type KeywordCoverageScorer struct {
	cfg KeywordCoverageConfig
}

// NewKeywordCoverageScorer 创建关键词覆盖率评估器
func NewKeywordCoverageScorer(cfg KeywordCoverageConfig) *KeywordCoverageScorer {
	if cfg.CaseInsensitive {
		normalized := make([]string, 0, len(cfg.Keywords))
		for _, k := range cfg.Keywords {
			normalized = append(normalized, strings.ToLower(strings.TrimSpace(k)))
		}
		cfg.Keywords = normalized
	}
	return &KeywordCoverageScorer{cfg: cfg}
}

// Score 实现 Scorer 接口
func (s *KeywordCoverageScorer) Score(ctx context.Context, input *TextEvalInput) (*ScoreResult, error) {
	if input == nil {
		return &ScoreResult{
			Name:  "keyword_coverage",
			Value: 0,
			Details: map[string]interface{}{
				"matched":   []string{},
				"unmatched": s.cfg.Keywords,
			},
		}, nil
	}

	answer := strings.TrimSpace(input.Answer)
	if s.cfg.CaseInsensitive {
		answer = strings.ToLower(answer)
	}

	matched := make([]string, 0, len(s.cfg.Keywords))
	unmatched := make([]string, 0, len(s.cfg.Keywords))

	for _, kw := range s.cfg.Keywords {
		if kw == "" {
			continue
		}
		if strings.Contains(answer, kw) {
			matched = append(matched, kw)
		} else {
			unmatched = append(unmatched, kw)
		}
	}

	total := len(matched) + len(unmatched)
	score := 0.0
	if total > 0 {
		score = float64(len(matched)) / float64(total)
	}

	return &ScoreResult{
		Name:  "keyword_coverage",
		Value: score,
		Details: map[string]interface{}{
			"matched":   matched,
			"unmatched": unmatched,
			"total":     total,
		},
	}, nil
}

// =========================
// 2. 词汇相似度评估器(简单 Jaccard)
// =========================

// LexicalSimilarityConfig 词汇相似度配置
type LexicalSimilarityConfig struct {
	// MinTokenLength 参与比较的最小 token 长度(过滤掉太短的词,默认: 2)
	MinTokenLength int
}

// LexicalSimilarityScorer 基于词汇集合的简单 Jaccard 相似度评估器。
// score = |A ∩ B| / |A ∪ B|, 范围 [0,1]。
type LexicalSimilarityScorer struct {
	cfg LexicalSimilarityConfig
}

// NewLexicalSimilarityScorer 创建词汇相似度评估器
func NewLexicalSimilarityScorer(cfg LexicalSimilarityConfig) *LexicalSimilarityScorer {
	if cfg.MinTokenLength <= 0 {
		cfg.MinTokenLength = 2
	}
	return &LexicalSimilarityScorer{cfg: cfg}
}

// Score 实现 Scorer 接口
func (s *LexicalSimilarityScorer) Score(ctx context.Context, input *TextEvalInput) (*ScoreResult, error) {
	if input == nil {
		return &ScoreResult{Name: "lexical_similarity", Value: 0}, nil
	}

	aTokens := tokenize(input.Answer, s.cfg.MinTokenLength)
	bTokens := tokenize(input.Reference, s.cfg.MinTokenLength)

	if len(aTokens) == 0 && len(bTokens) == 0 {
		return &ScoreResult{Name: "lexical_similarity", Value: 1}, nil
	}

	intersection := 0
	union := make(map[string]bool)

	for token := range aTokens {
		union[token] = true
	}
	for token := range bTokens {
		if aTokens[token] {
			intersection++
		}
		union[token] = true
	}

	score := 0.0
	if len(union) > 0 {
		score = float64(intersection) / float64(len(union))
	}

	return &ScoreResult{
		Name:  "lexical_similarity",
		Value: score,
		Details: map[string]interface{}{
			"intersection": intersection,
			"union_size":   len(union),
		},
	}, nil
}

// tokenize 将文本拆分为简单的词汇集合,用于词汇相似度计算。
func tokenize(text string, minLen int) map[string]bool {
	text = strings.ToLower(text)
	var tokens []string
	var cur strings.Builder

	flush := func() {
		if cur.Len() >= minLen {
			tokens = append(tokens, cur.String())
		}
		cur.Reset()
	}

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()

	result := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		result[t] = true
	}
	return result
}

