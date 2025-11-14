package evals

import (
	"context"
	"testing"

	"github.com/wordflowlab/agentsdk/pkg/provider"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// MockProvider 模拟 LLM Provider
type MockProvider struct {
	response string
}

func (m *MockProvider) Stream(ctx context.Context, messages []types.Message, opts *provider.StreamOptions) (<-chan provider.StreamChunk, error) {
	return nil, nil
}

func (m *MockProvider) Complete(ctx context.Context, messages []types.Message, opts *provider.StreamOptions) (*provider.CompleteResponse, error) {
	return &provider.CompleteResponse{
		Message: types.Message{
			Role:    types.RoleAssistant,
			Content: m.response,
		},
		Usage: &provider.TokenUsage{
			InputTokens:  10,
			OutputTokens: 20,
		},
	}, nil
}

func (m *MockProvider) Config() *types.ModelConfig {
	return &types.ModelConfig{
		Provider: "mock",
		Model:    "mock-model",
	}
}

func (m *MockProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{}
}

func (m *MockProvider) SetSystemPrompt(prompt string) error {
	return nil
}

func (m *MockProvider) GetSystemPrompt() string {
	return ""
}

func (m *MockProvider) Close() error {
	return nil
}

// TestParseScoreResponse_JSON 测试 JSON 格式解析
func TestParseScoreResponse_JSON(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		wantScore    float64
		wantReason   string
		wantErr      bool
	}{
		{
			name:       "valid JSON with reason",
			output:     `{"score": 0.85, "reason": "答案质量很好"}`,
			wantScore:  0.85,
			wantReason: "答案质量很好",
			wantErr:    false,
		},
		{
			name:       "valid JSON without reason",
			output:     `{"score": 0.90}`,
			wantScore:  0.90,
			wantReason: "",
			wantErr:    false,
		},
		{
			name:       "valid JSON with details",
			output:     `{"score": 0.75, "reason": "良好", "details": {"key": "value"}}`,
			wantScore:  0.75,
			wantReason: "良好",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, reason, _, err := parseScoreResponse(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseScoreResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if score != tt.wantScore {
				t.Errorf("parseScoreResponse() score = %v, want %v", score, tt.wantScore)
			}
			if reason != tt.wantReason {
				t.Errorf("parseScoreResponse() reason = %v, want %v", reason, tt.wantReason)
			}
		})
	}
}

// TestParseScoreResponse_Text 测试纯文本格式解析
func TestParseScoreResponse_Text(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantScore  float64
		wantErr    bool
	}{
		{
			name:      "text with Score:",
			output:    "Score: 0.88\nReason: 答案很完整",
			wantScore: 0.88,
			wantErr:   false,
		},
		{
			name:      "text with score: (lowercase)",
			output:    "score: 0.92 reason: 非常好",
			wantScore: 0.92,
			wantErr:   false,
		},
		{
			name:      "text with number only",
			output:    "0.78",
			wantScore: 0.78,
			wantErr:   false,
		},
		{
			name:    "text without number",
			output:  "这是一个没有数字的文本",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, _, _, err := parseScoreResponse(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseScoreResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && score != tt.wantScore {
				t.Errorf("parseScoreResponse() score = %v, want %v", score, tt.wantScore)
			}
		})
	}
}

// TestLLMScorer_Score 测试 LLM Scorer 的基本功能
func TestLLMScorer_Score(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  string
		input         *TextEvalInput
		wantScore     float64
		wantErr       bool
	}{
		{
			name:         "successful JSON response",
			mockResponse: `{"score": 0.95, "reason": "答案非常忠实于上下文"}`,
			input: &TextEvalInput{
				Answer:  "巴黎是法国的首都。",
				Context: []string{"法国的首都是巴黎。"},
			},
			wantScore: 0.95,
			wantErr:   false,
		},
		{
			name:         "successful text response",
			mockResponse: "Score: 0.80\nReason: 答案基本正确",
			input: &TextEvalInput{
				Answer: "伦敦是英国的首都。",
			},
			wantScore: 0.80,
			wantErr:   false,
		},
		{
			name:         "nil input",
			mockResponse: `{"score": 0.0}`,
			input:        nil,
			wantScore:    0.0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &MockProvider{response: tt.mockResponse}

			scorer := NewLLMScorer(LLMScorerConfig{
				Provider:    mockProvider,
				Name:        "test_scorer",
				Prompt:      "Test prompt: {{answer}}",
				MaxTokens:   100,
				Temperature: 0,
			})

			result, err := scorer.Score(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("LLMScorer.Score() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.Name != "test_scorer" {
					t.Errorf("LLMScorer.Score() name = %v, want test_scorer", result.Name)
				}
				if result.Value != tt.wantScore {
					t.Errorf("LLMScorer.Score() score = %v, want %v", result.Value, tt.wantScore)
				}
				if result.Details == nil {
					t.Error("LLMScorer.Score() details should not be nil")
				}
			}
		})
	}
}

// TestFaithfulnessScorer 测试忠实度评分器
func TestFaithfulnessScorer(t *testing.T) {
	mockProvider := &MockProvider{
		response: `{"score": 0.90, "reason": "答案完全基于上下文"}`,
	}

	scorer := NewFaithfulnessScorer(mockProvider)

	input := &TextEvalInput{
		Answer:  "巴黎是法国的首都。",
		Context: []string{"法国的首都是巴黎。"},
	}

	result, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("FaithfulnessScorer.Score() error = %v", err)
	}

	if result.Name != "faithfulness" {
		t.Errorf("FaithfulnessScorer.Score() name = %v, want faithfulness", result.Name)
	}

	if result.Value != 0.90 {
		t.Errorf("FaithfulnessScorer.Score() score = %v, want 0.90", result.Value)
	}

	if result.Details["reason"] != "答案完全基于上下文" {
		t.Errorf("FaithfulnessScorer.Score() reason = %v", result.Details["reason"])
	}
}

// TestHallucinationScorer 测试幻觉检测评分器
func TestHallucinationScorer(t *testing.T) {
	mockProvider := &MockProvider{
		response: `{"score": 0.85, "reason": "未检测到明显幻觉"}`,
	}

	scorer := NewHallucinationScorer(mockProvider)

	input := &TextEvalInput{
		Answer:  "伦敦是英国的首都，位于泰晤士河畔。",
		Context: []string{"英国的首都是伦敦。"},
	}

	result, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("HallucinationScorer.Score() error = %v", err)
	}

	if result.Name != "hallucination" {
		t.Errorf("HallucinationScorer.Score() name = %v, want hallucination", result.Name)
	}

	if result.Value != 0.85 {
		t.Errorf("HallucinationScorer.Score() score = %v, want 0.85", result.Value)
	}
}

// TestAnswerRelevancyScorer 测试答案相关性评分器
func TestAnswerRelevancyScorer(t *testing.T) {
	mockProvider := &MockProvider{
		response: `{"score": 0.92, "reason": "答案直接回答了问题"}`,
	}

	scorer := NewAnswerRelevancyScorer(mockProvider)

	input := &TextEvalInput{
		Answer:  "东京是日本的首都。",
		Context: []string{"日本的首都是哪里？"},
	}

	result, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("AnswerRelevancyScorer.Score() error = %v", err)
	}

	if result.Name != "answer_relevancy" {
		t.Errorf("AnswerRelevancyScorer.Score() name = %v, want answer_relevancy", result.Name)
	}

	if result.Value != 0.92 {
		t.Errorf("AnswerRelevancyScorer.Score() score = %v, want 0.92", result.Value)
	}
}

// TestToxicityScorer 测试毒性检测评分器
func TestToxicityScorer(t *testing.T) {
	mockProvider := &MockProvider{
		response: `{"score": 1.0, "reason": "内容安全，无有害信息"}`,
	}

	scorer := NewToxicityScorer(mockProvider)

	input := &TextEvalInput{
		Answer: "这是一个友好的回答。",
	}

	result, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("ToxicityScorer.Score() error = %v", err)
	}

	if result.Name != "toxicity" {
		t.Errorf("ToxicityScorer.Score() name = %v, want toxicity", result.Name)
	}

	if result.Value != 1.0 {
		t.Errorf("ToxicityScorer.Score() score = %v, want 1.0", result.Value)
	}
}

// TestCoherenceScorer 测试连贯性评分器
func TestCoherenceScorer(t *testing.T) {
	mockProvider := &MockProvider{
		response: `{"score": 0.88, "reason": "文本逻辑连贯，结构清晰"}`,
	}

	scorer := NewCoherenceScorer(mockProvider)

	input := &TextEvalInput{
		Answer: "首先，我们需要了解背景。其次，分析问题。最后，给出结论。",
	}

	result, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("CoherenceScorer.Score() error = %v", err)
	}

	if result.Name != "coherence" {
		t.Errorf("CoherenceScorer.Score() name = %v, want coherence", result.Name)
	}

	if result.Value != 0.88 {
		t.Errorf("CoherenceScorer.Score() score = %v, want 0.88", result.Value)
	}
}

// TestCompletenessScorer 测试完整性评分器
func TestCompletenessScorer(t *testing.T) {
	mockProvider := &MockProvider{
		response: `{"score": 0.95, "reason": "答案全面，涵盖所有要点"}`,
	}

	scorer := NewCompletenessScorer(mockProvider)

	input := &TextEvalInput{
		Answer:    "巴黎是法国的首都，也是最大城市，位于法国北部。",
		Context:   []string{"请介绍巴黎"},
		Reference: "巴黎是法国的首都和最大城市",
	}

	result, err := scorer.Score(context.Background(), input)
	if err != nil {
		t.Fatalf("CompletenessScorer.Score() error = %v", err)
	}

	if result.Name != "completeness" {
		t.Errorf("CompletenessScorer.Score() name = %v, want completeness", result.Name)
	}

	if result.Value != 0.95 {
		t.Errorf("CompletenessScorer.Score() score = %v, want 0.95", result.Value)
	}
}
