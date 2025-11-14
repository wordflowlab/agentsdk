package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/wordflowlab/agentsdk/pkg/agent"
	"github.com/wordflowlab/agentsdk/pkg/evals"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/provider"
	"github.com/wordflowlab/agentsdk/pkg/session"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// Server 提供基于 HTTP 的 AgentSDK 接入层。
// 设计目标:
// - 封装 Agent 创建和一次性 Chat 调用
// - 提供简单、可扩展的 REST 接口,便于前端/第三方集成
// - 后续可以在此基础上扩展 streaming、会话管理等能力
type Server struct {
	deps         *agent.Dependencies
	workflowRuns *workflowRunStore
}

// New 创建一个 Server 实例。
func New(deps *agent.Dependencies) *Server {
	return &Server{
		deps:         deps,
		workflowRuns: newWorkflowRunStore(),
	}
}

// ======================
// 1. Chat 接口
// ======================

// ChatRequest 表示 HTTP Chat 请求体。
// 这是一个最小可用结构,后续可按需扩展。
type ChatRequest struct {
	// TemplateID 指定要使用的 Agent 模板 ID。
	TemplateID string `json:"template_id"`
	// Input 用户输入的自然语言文本。
	Input string `json:"input"`

	// RoutingProfile 可选路由偏好,与 types.AgentConfig.RoutingProfile 对应。
	// 例如: "cost"、"quality"、"latency"。
	RoutingProfile string `json:"routing_profile,omitempty"`

	// ModelConfig 可选模型配置,未提供时使用模板默认配置。
	ModelConfig *types.ModelConfig `json:"model_config,omitempty"`
	// Sandbox 可选沙箱配置,未提供时使用默认本地沙箱。
	Sandbox *types.SandboxConfig `json:"sandbox,omitempty"`
	// Middlewares 可选中间件列表。
	Middlewares []string `json:"middlewares,omitempty"`
	// Metadata 可选元数据,会传递给 AgentConfig.Metadata。
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ChatResponse 表示 Chat 请求的同步响应。
type ChatResponse struct {
	AgentID string `json:"agent_id"`
	Text    string `json:"text"`
	Status  string `json:"status"`

	// ErrorMessage 非空表示请求失败时的错误信息。
	ErrorMessage string `json:"error_message,omitempty"`
}

// ChatHandler 返回一个 HTTP handler,用于处理同步 Chat 请求。
//
// 路径示例:
//   POST /v1/agents/chat
//
// 请求体:
//   {
//     "template_id": "assistant",
//     "input": "你好,帮我总结一下 README",
//     "metadata": {"user_id": "alice"},
//     "middlewares": ["filesystem", "agent_memory"]
//   }
//
// 响应体:
//   {
//     "agent_id": "agt:...",
//     "text": "...",
//     "status": "ok"
//   }
func (s *Server) ChatHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if req.TemplateID == "" {
			http.Error(w, "template_id is required", http.StatusBadRequest)
			return
		}
		if req.Input == "" {
			http.Error(w, "input is required", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		// 为单次请求设置一个合理的超时,避免挂死。
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		agentConfig := &types.AgentConfig{
			TemplateID:     req.TemplateID,
			ModelConfig:    req.ModelConfig,
			Sandbox:        req.Sandbox,
			Middlewares:    req.Middlewares,
			Metadata:       req.Metadata,
			RoutingProfile: req.RoutingProfile,
		}

		ag, err := agent.Create(ctx, agentConfig, s.deps)
		if err != nil {
			logging.Error(ctx, "http.chat.error", map[string]interface{}{
				"template_id":     req.TemplateID,
				"routing_profile": req.RoutingProfile,
				"stage":           "create_agent",
				"error":           err.Error(),
				"latency_ms":      time.Since(start).Milliseconds(),
			})

			writeJSON(w, http.StatusInternalServerError, &ChatResponse{
				Status:       "error",
				ErrorMessage: err.Error(),
			})
			return
		}
		defer ag.Close()

		result, err := ag.Chat(ctx, req.Input)
		latencyMs := time.Since(start).Milliseconds()
		if err != nil {
			logging.Error(ctx, "http.chat.error", map[string]interface{}{
				"template_id":     req.TemplateID,
				"routing_profile": req.RoutingProfile,
				"stage":           "chat",
				"agent_id":        ag.ID(),
				"error":           err.Error(),
				"latency_ms":      latencyMs,
			})

			writeJSON(w, http.StatusInternalServerError, &ChatResponse{
				AgentID:      ag.ID(),
				Status:       "error",
				ErrorMessage: err.Error(),
			})
			return
		}

		logging.Info(ctx, "http.chat.completed", map[string]interface{}{
			"template_id":     req.TemplateID,
			"routing_profile": req.RoutingProfile,
			"agent_id":        ag.ID(),
			"text_len":        len(result.Text),
			"latency_ms":      latencyMs,
		})

		writeJSON(w, http.StatusOK, &ChatResponse{
			AgentID: ag.ID(),
			Text:    result.Text,
			Status:  "ok",
		})
	})
}

// writeJSON 帮助函数: 设置正确的响应头并编码 JSON。
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ChatStreamHandler 提供基于 Server-Sent Events (SSE) 的流式 Chat 接口。
//
// 路径示例:
//   POST /v1/agents/chat/stream
//
// 请求体与 ChatHandler 相同,但响应为 text/event-stream。
// 每个事件均为一行 JSON:
//   data: {"cursor":1,"bookmark":{...},"event":{...}}\n\n
//
// 前端可以根据 event 字段中的具体类型(decoding types.*Event)来渲染流式思考/文本/工具调用等。
func (s *Server) ChatStreamHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.TemplateID == "" || req.Input == "" {
			http.Error(w, "template_id and input are required", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		agentConfig := &types.AgentConfig{
			TemplateID:     req.TemplateID,
			ModelConfig:    req.ModelConfig,
			Sandbox:        req.Sandbox,
			Middlewares:    req.Middlewares,
			Metadata:       req.Metadata,
			RoutingProfile: req.RoutingProfile,
		}

		ag, err := agent.Create(ctx, agentConfig, s.deps)
		if err != nil {
			logging.Error(ctx, "http.chat_stream.error", map[string]interface{}{
				"template_id":     req.TemplateID,
				"routing_profile": req.RoutingProfile,
				"stage":           "create_agent",
				"error":           err.Error(),
				"latency_ms":      time.Since(start).Milliseconds(),
			})

			http.Error(w, "create agent failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer ag.Close()

		// 设置 SSE 响应头
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// 订阅 Progress + Monitor 事件
		eventCh := ag.Subscribe(
			[]types.AgentChannel{types.ChannelProgress, types.ChannelMonitor},
			nil,
		)
		defer ag.Unsubscribe(eventCh)

		// 启动 Chat 调用
		go func() {
			_, _ = ag.Chat(ctx, req.Input)
		}()

		enc := json.NewEncoder(w)

		// 流式发送事件
		for {
			select {
			case <-ctx.Done():
				return
			case env, ok := <-eventCh:
				if !ok {
					return
				}

				// 将 envelope 包装为一条 SSE 消息
				w.Write([]byte("data: "))
				if err := enc.Encode(env); err != nil {
					return
				}
				w.Write([]byte("\n"))
				flusher.Flush()

				// 如果是 ProgressDoneEvent, 可以结束流
				if evt, ok := env.Event.(types.EventType); ok {
					if evt.EventType() == "done" {
						return
					}
				}

				// 在 done 时记录一次完成日志
				if evt, ok := env.Event.(types.EventType); ok {
					if evt.EventType() == "done" {
						logging.Info(ctx, "http.chat_stream.completed", map[string]interface{}{
							"template_id":     req.TemplateID,
							"routing_profile": req.RoutingProfile,
							"agent_id":        ag.ID(),
							"latency_ms":      time.Since(start).Milliseconds(),
						})
						return
					}
				}
			}
		}
	})
}

// ======================
// 2. 文本评估(Evals) 接口
// ======================

// TextEvalRequest 表示一次文本评估请求。
// 该接口仅使用本地启发式 scorer, 不依赖外部 LLM。
type TextEvalRequest struct {
	// Answer 待评估的模型输出。
	Answer string `json:"answer"`
	// Context 可选的上下文信息, 当前实现不会直接使用, 预留扩展。
	Context []string `json:"context,omitempty"`
	// Reference 可选参考答案, 用于词汇相似度评估。
	Reference string `json:"reference,omitempty"`
	// Keywords 关键词列表, 用于关键词覆盖率评估。
	Keywords []string `json:"keywords,omitempty"`
	// Scorers 指定要启用的 scorer 名称, 为空时默认启用:
	// ["keyword_coverage", "lexical_similarity"]。
	Scorers []string `json:"scorers,omitempty"`
}

// TextEvalResponse 表示文本评估的响应。
type TextEvalResponse struct {
	Scores       []evals.ScoreResult `json:"scores"`
	ErrorMessage string              `json:"error_message,omitempty"`
}

// SessionEvalRequest 表示基于 Session 事件的评估请求。
type SessionEvalRequest struct {
	// Events 会话事件列表, 使用 pkg/session.Event 结构。
	Events []session.Event `json:"events"`
	// Reference 可选参考答案, 用于词汇相似度评估。
	Reference string `json:"reference,omitempty"`
	// Keywords 关键词列表, 用于关键词覆盖率评估。
	Keywords []string `json:"keywords,omitempty"`
	// Scorers 指定要启用的 scorer 名称, 为空时默认启用:
	// ["keyword_coverage", "lexical_similarity"]。
	Scorers []string `json:"scorers,omitempty"`
}

// TextEvalHandler 返回一个 HTTP handler, 用于处理文本评估请求。
//
// 路径示例:
//   POST /v1/evals/text
//
// 请求体:
//   {
//     "answer": "Paris is the capital of France.",
//     "reference": "Paris is the capital city of France, a country in Europe.",
//     "keywords": ["paris", "capital", "france", "europe"],
//     "scorers": ["keyword_coverage", "lexical_similarity"]
//   }
//
// 响应体:
//   {
//     "scores": [
//       {"name":"keyword_coverage","value":1.0,"details":{...}},
//       {"name":"lexical_similarity","value":0.8,"details":{...}}
//     ]
//   }
func (s *Server) TextEvalHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req TextEvalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if req.Answer == "" {
			http.Error(w, "answer is required", http.StatusBadRequest)
			return
		}

		// 解析需要运行的 scorer 列表
		scorerNames := req.Scorers
		if len(scorerNames) == 0 {
			scorerNames = []string{"keyword_coverage", "lexical_similarity"}
		}

		for _, name := range scorerNames {
			if name != "keyword_coverage" && name != "lexical_similarity" {
				http.Error(w, "unsupported scorer: "+name, http.StatusBadRequest)
				return
			}
		}

		ctx := r.Context()
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		input := &evals.TextEvalInput{
			Answer:    req.Answer,
			Context:   req.Context,
			Reference: req.Reference,
		}

		scores := make([]evals.ScoreResult, 0, len(scorerNames))

		for _, name := range scorerNames {
			switch name {
			case "keyword_coverage":
				scorer := evals.NewKeywordCoverageScorer(evals.KeywordCoverageConfig{
					Keywords:        req.Keywords,
					CaseInsensitive: true,
				})
				res, err := scorer.Score(ctx, input)
				if err != nil {
					logging.Error(ctx, "http.evals.text.error", map[string]interface{}{
						"scorer":     "keyword_coverage",
						"error":      err.Error(),
						"latency_ms": time.Since(start).Milliseconds(),
					})

					writeJSON(w, http.StatusInternalServerError, &TextEvalResponse{
						ErrorMessage: err.Error(),
					})
					return
				}
				scores = append(scores, *res)
			case "lexical_similarity":
				scorer := evals.NewLexicalSimilarityScorer(evals.LexicalSimilarityConfig{
					MinTokenLength: 2,
				})
				res, err := scorer.Score(ctx, input)
				if err != nil {
					logging.Error(ctx, "http.evals.text.error", map[string]interface{}{
						"scorer":     "lexical_similarity",
						"error":      err.Error(),
						"latency_ms": time.Since(start).Milliseconds(),
					})

					writeJSON(w, http.StatusInternalServerError, &TextEvalResponse{
						ErrorMessage: err.Error(),
					})
					return
				}
				scores = append(scores, *res)
			}
		}

		logging.Info(ctx, "http.evals.text.completed", map[string]interface{}{
			"scorers":    scorerNames,
			"latency_ms": time.Since(start).Milliseconds(),
		})

		writeJSON(w, http.StatusOK, &TextEvalResponse{
			Scores: scores,
		})
	})
}

// SessionEvalHandler 返回一个 HTTP handler, 基于 Session 事件进行评估。
//
// 路径示例:
//   POST /v1/evals/session
//
// 请求体:
//   {
//     "events": [...],                // session.Event JSON 列表
//     "reference": "参考答案",
//     "keywords": ["paris", "capital"],
//     "scorers": ["keyword_coverage", "lexical_similarity"]
//   }
func (s *Server) SessionEvalHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req SessionEvalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if len(req.Events) == 0 {
			http.Error(w, "events is required", http.StatusBadRequest)
			return
		}

		scorerNames := req.Scorers
		if len(scorerNames) == 0 {
			scorerNames = []string{"keyword_coverage", "lexical_similarity"}
		}
		for _, name := range scorerNames {
			if name != "keyword_coverage" && name != "lexical_similarity" {
				http.Error(w, "unsupported scorer: "+name, http.StatusBadRequest)
				return
			}
		}

		ctx := r.Context()
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// 从事件构建 TextEvalInput
		textInput := evals.BuildTextEvalInputFromEvents(req.Events)
		textInput.Reference = req.Reference

		scores := make([]evals.ScoreResult, 0, len(scorerNames))

		for _, name := range scorerNames {
			switch name {
			case "keyword_coverage":
				scorer := evals.NewKeywordCoverageScorer(evals.KeywordCoverageConfig{
					Keywords:        req.Keywords,
					CaseInsensitive: true,
				})
				res, err := scorer.Score(ctx, textInput)
				if err != nil {
					logging.Error(ctx, "http.evals.session.error", map[string]interface{}{
						"scorer":     "keyword_coverage",
						"error":      err.Error(),
						"latency_ms": time.Since(start).Milliseconds(),
					})

					writeJSON(w, http.StatusInternalServerError, &TextEvalResponse{
						ErrorMessage: err.Error(),
					})
					return
				}
				scores = append(scores, *res)
			case "lexical_similarity":
				scorer := evals.NewLexicalSimilarityScorer(evals.LexicalSimilarityConfig{
					MinTokenLength: 2,
				})
				res, err := scorer.Score(ctx, textInput)
				if err != nil {
					logging.Error(ctx, "http.evals.session.error", map[string]interface{}{
						"scorer":     "lexical_similarity",
						"error":      err.Error(),
						"latency_ms": time.Since(start).Milliseconds(),
					})

					writeJSON(w, http.StatusInternalServerError, &TextEvalResponse{
						ErrorMessage: err.Error(),
					})
					return
				}
				scores = append(scores, *res)
			}
		}

		logging.Info(ctx, "http.evals.session.completed", map[string]interface{}{
			"scorers":     scorerNames,
			"event_count": len(req.Events),
			"latency_ms":  time.Since(start).Milliseconds(),
		})

		writeJSON(w, http.StatusOK, &TextEvalResponse{
			Scores: scores,
		})
	})
}

// ======================
// 4. Batch Eval 接口
// ======================

// BatchEvalRequest 表示批量评估请求
type BatchEvalRequest struct {
	// TestCases 测试用例列表
	TestCases []BatchTestCaseRequest `json:"test_cases"`
	// Scorers 要使用的评分器列表
	Scorers []string `json:"scorers,omitempty"`
	// Concurrency 并发数（默认: 1）
	Concurrency int `json:"concurrency,omitempty"`
	// StopOnError 遇到错误时是否停止（默认: false）
	StopOnError bool `json:"stop_on_error,omitempty"`
	// Keywords 关键词列表（用于keyword_coverage scorer）
	Keywords []string `json:"keywords,omitempty"`
	// ProviderConfig 可选的LLM Provider配置（用于LLM-based scorers）
	ProviderConfig *types.ModelConfig `json:"provider_config,omitempty"`
}

// BatchTestCaseRequest 单个测试用例请求
type BatchTestCaseRequest struct {
	ID        string                 `json:"id"`
	Answer    string                 `json:"answer"`
	Context   []string               `json:"context,omitempty"`
	Reference string                 `json:"reference,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// BatchEvalResponse 批量评估响应
type BatchEvalResponse struct {
	Results       []BatchResultResponse `json:"results"`
	Summary       *BatchSummaryResponse `json:"summary"`
	TotalDuration int64                 `json:"total_duration_ms"`
}

// BatchResultResponse 单个测试用例的结果
type BatchResultResponse struct {
	TestCaseID string                 `json:"test_case_id"`
	Scores     []evals.ScoreResult    `json:"scores"`
	Duration   int64                  `json:"duration_ms"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// BatchSummaryResponse 批量评估汇总
type BatchSummaryResponse struct {
	TotalCases      int                `json:"total_cases"`
	SuccessfulCases int                `json:"successful_cases"`
	FailedCases     int                `json:"failed_cases"`
	AverageScores   map[string]float64 `json:"average_scores"`
	AverageDuration int64              `json:"average_duration_ms"`
}

// BatchEvalHandler 返回批量评估的HTTP handler
func (s *Server) BatchEvalHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req BatchEvalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if len(req.TestCases) == 0 {
			http.Error(w, "test_cases is required", http.StatusBadRequest)
			return
		}

		// 默认使用基础 scorers
		scorerNames := req.Scorers
		if len(scorerNames) == 0 {
			scorerNames = []string{"keyword_coverage", "lexical_similarity"}
		}

		ctx := r.Context()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		// 创建 scorers
		scorers := make([]evals.Scorer, 0, len(scorerNames))
		var llmProvider provider.Provider

		for _, name := range scorerNames {
			switch name {
			case "keyword_coverage":
				scorer := evals.NewKeywordCoverageScorer(evals.KeywordCoverageConfig{
					Keywords:        req.Keywords,
					CaseInsensitive: true,
				})
				scorers = append(scorers, scorer)

			case "lexical_similarity":
				scorer := evals.NewLexicalSimilarityScorer(evals.LexicalSimilarityConfig{
					MinTokenLength: 2,
				})
				scorers = append(scorers, scorer)

			case "faithfulness", "hallucination", "answer_relevancy", "context_relevancy",
				"toxicity", "tone_consistency", "coherence", "completeness":
				// LLM-based scorers 需要 Provider
				if llmProvider == nil {
					if req.ProviderConfig == nil {
						http.Error(w, "provider_config is required for LLM-based scorers", http.StatusBadRequest)
						return
					}
					var err error
					llmProvider, err = s.deps.ProviderFactory.Create(req.ProviderConfig)
					if err != nil {
						http.Error(w, "failed to create provider: "+err.Error(), http.StatusInternalServerError)
						return
					}
					defer llmProvider.Close()
				}

				switch name {
				case "faithfulness":
					scorers = append(scorers, evals.NewFaithfulnessScorer(llmProvider))
				case "hallucination":
					scorers = append(scorers, evals.NewHallucinationScorer(llmProvider))
				case "answer_relevancy":
					scorers = append(scorers, evals.NewAnswerRelevancyScorer(llmProvider))
				case "context_relevancy":
					scorers = append(scorers, evals.NewContextRelevancyScorer(llmProvider))
				case "toxicity":
					scorers = append(scorers, evals.NewToxicityScorer(llmProvider))
				case "tone_consistency":
					scorers = append(scorers, evals.NewToneConsistencyScorer(llmProvider))
				case "coherence":
					scorers = append(scorers, evals.NewCoherenceScorer(llmProvider))
				case "completeness":
					scorers = append(scorers, evals.NewCompletenessScorer(llmProvider))
				}

			default:
				http.Error(w, "unsupported scorer: "+name, http.StatusBadRequest)
				return
			}
		}

		// 转换测试用例
		testCases := make([]*evals.BatchTestCase, len(req.TestCases))
		for i, tc := range req.TestCases {
			testCases[i] = &evals.BatchTestCase{
				ID: tc.ID,
				Input: &evals.TextEvalInput{
					Answer:    tc.Answer,
					Context:   tc.Context,
					Reference: tc.Reference,
				},
				Metadata: tc.Metadata,
			}
		}

		// 运行批量评估
		concurrency := req.Concurrency
		if concurrency <= 0 {
			concurrency = 1
		}

		result, err := evals.RunBatch(ctx, &evals.BatchConfig{
			TestCases:   testCases,
			Scorers:     scorers,
			Concurrency: concurrency,
			StopOnError: req.StopOnError,
		})

		if err != nil {
			logging.Error(ctx, "http.evals.batch.error", map[string]interface{}{
				"error":      err.Error(),
				"latency_ms": time.Since(start).Milliseconds(),
			})
			http.Error(w, "batch evaluation failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 转换响应
		response := &BatchEvalResponse{
			Results:       make([]BatchResultResponse, len(result.Results)),
			TotalDuration: result.TotalDuration.Milliseconds(),
		}

		for i, r := range result.Results {
			response.Results[i] = BatchResultResponse{
				TestCaseID: r.TestCaseID,
				Scores:     make([]evals.ScoreResult, len(r.Scores)),
				Duration:   r.Duration.Milliseconds(),
				Error:      r.Error,
				Metadata:   r.Metadata,
			}
			for j, s := range r.Scores {
				response.Results[i].Scores[j] = *s
			}
		}

		if result.Summary != nil {
			response.Summary = &BatchSummaryResponse{
				TotalCases:      result.Summary.TotalCases,
				SuccessfulCases: result.Summary.SuccessfulCases,
				FailedCases:     result.Summary.FailedCases,
				AverageScores:   result.Summary.AverageScores,
				AverageDuration: result.Summary.AverageDuration.Milliseconds(),
			}
		}

		logging.Info(ctx, "http.evals.batch.completed", map[string]interface{}{
			"total_cases":      len(testCases),
			"successful_cases": result.Summary.SuccessfulCases,
			"failed_cases":     result.Summary.FailedCases,
			"concurrency":      concurrency,
			"latency_ms":       time.Since(start).Milliseconds(),
		})

		writeJSON(w, http.StatusOK, response)
	})
}
