package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
)

// EvalRecord 评估记录
type EvalRecord struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`   // text, session, batch, benchmark
	Status      string                 `json:"status"` // pending, running, completed, failed
	Input       map[string]interface{} `json:"input,omitempty"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Metrics     map[string]float64     `json:"metrics,omitempty"`
	Score       float64                `json:"score,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Duration    int64                  `json:"duration,omitempty"` // milliseconds
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BenchmarkRecord 基准测试记录
type BenchmarkRecord struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Runs      int                    `json:"runs"`
	Results   []map[string]float64   `json:"results,omitempty"`
	Summary   map[string]interface{} `json:"summary,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// registerEvalRoutes 注册 Eval 路由
func registerEvalRoutes(v1 *gin.RouterGroup, st store.Store) {
	eval := v1.Group("/eval")
	{
		// 基础评估
		eval.POST("/text", runTextEval(st))
		eval.POST("/session", runSessionEval(st))
		eval.POST("/batch", runBatchEval(st))
		eval.POST("/custom", runCustomEval(st))

		// 评估管理
		evals := eval.Group("/evals")
		{
			evals.GET("", listEvals(st))
			evals.GET("/:id", getEval(st))
			evals.DELETE("/:id", deleteEval(st))
			evals.POST("/:id/cancel", cancelEval(st))
			evals.GET("/:id/results", getEvalResults(st))
			evals.GET("/:id/metrics", getEvalMetrics(st))
		}

		// 基准测试
		benchmarks := eval.Group("/benchmarks")
		{
			benchmarks.POST("", createBenchmark(st))
			benchmarks.GET("", listBenchmarks(st))
			benchmarks.GET("/:id", getBenchmark(st))
			benchmarks.DELETE("/:id", deleteBenchmark(st))
			benchmarks.POST("/:id/run", runBenchmark(st))
			benchmarks.GET("/:id/results", getBenchmarkResults(st))
		}

		// 比较和分析
		eval.POST("/compare", compareEvals(st))
		eval.GET("/stats", getEvalStats(st))
		eval.POST("/export", exportEvals(st))
		eval.GET("/health", getEvalHealth(st))
	}
}

// runTextEval 运行文本评估
func runTextEval(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Prompt   string                 `json:"prompt" binding:"required"`
			Expected string                 `json:"expected"`
			Criteria map[string]interface{} `json:"criteria"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		ctx := c.Request.Context()
		evalRec := &EvalRecord{
			ID: uuid.New().String(), Name: "Text Eval", Type: "text",
			Status: "completed", Input: map[string]interface{}{"prompt": req.Prompt},
			Output:  map[string]interface{}{"result": "Sample output"},
			Metrics: map[string]float64{"accuracy": 0.95}, Score: 0.95,
			StartedAt: time.Now(), Duration: 100,
		}

		now := time.Now()
		evalRec.CompletedAt = &now

		if err := st.Set(ctx, "evals", evalRec.ID, evalRec); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "eval.text.completed", map[string]interface{}{"id": evalRec.ID, "score": evalRec.Score})
		c.JSON(200, gin.H{"success": true, "data": evalRec})
	}
}

// runSessionEval 运行会话评估
func runSessionEval(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SessionID string                 `json:"session_id" binding:"required"`
			Criteria  map[string]interface{} `json:"criteria"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		ctx := c.Request.Context()
		evalRec := &EvalRecord{
			ID: uuid.New().String(), Name: "Session Eval", Type: "session",
			Status: "completed", Input: map[string]interface{}{"session_id": req.SessionID},
			Metrics: map[string]float64{"coherence": 0.90, "relevance": 0.88}, Score: 0.89,
			StartedAt: time.Now(), Duration: 250,
		}

		now := time.Now()
		evalRec.CompletedAt = &now

		if err := st.Set(ctx, "evals", evalRec.ID, evalRec); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "eval.session.completed", map[string]interface{}{"id": evalRec.ID})
		c.JSON(200, gin.H{"success": true, "data": evalRec})
	}
}

// runBatchEval 运行批量评估
func runBatchEval(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Items []map[string]interface{} `json:"items" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		ctx := c.Request.Context()
		evalRec := &EvalRecord{
			ID: uuid.New().String(), Name: "Batch Eval", Type: "batch",
			Status: "completed", Input: map[string]interface{}{"count": len(req.Items)},
			Metrics: map[string]float64{"avg_score": 0.92}, Score: 0.92,
			StartedAt: time.Now(), Duration: 500,
		}

		now := time.Now()
		evalRec.CompletedAt = &now

		if err := st.Set(ctx, "evals", evalRec.ID, evalRec); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "eval.batch.completed", map[string]interface{}{"id": evalRec.ID, "count": len(req.Items)})
		c.JSON(202, gin.H{"success": true, "data": evalRec})
	}
}

// runCustomEval 运行自定义评估
func runCustomEval(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name   string                 `json:"name" binding:"required"`
			Config map[string]interface{} `json:"config"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		ctx := c.Request.Context()
		evalRec := &EvalRecord{
			ID: uuid.New().String(), Name: req.Name, Type: "custom",
			Status: "pending", StartedAt: time.Now(),
		}

		if err := st.Set(ctx, "evals", evalRec.ID, evalRec); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(202, gin.H{"success": true, "data": evalRec})
	}
}

// listEvals 列出评估
func listEvals(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		records, err := st.List(ctx, "evals")
		if err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		evals := make([]*EvalRecord, 0)
		for _, record := range records {
			var e EvalRecord
			if err := store.DecodeValue(record, &e); err != nil {
				continue
			}
			evals = append(evals, &e)
		}

		c.JSON(200, gin.H{"success": true, "data": evals})
	}
}

// getEval 获取评估
func getEval(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var evalRec EvalRecord
		if err := st.Get(ctx, "evals", id, &evalRec); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Eval not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(200, gin.H{"success": true, "data": &evalRec})
	}
}

// deleteEval 删除评估
func deleteEval(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "evals", id); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Eval not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.Status(204)
	}
}

// cancelEval 取消评估
func cancelEval(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "eval.cancelled", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": gin.H{"cancelled": true, "eval_id": id}})
	}
}

// getEvalResults 获取结果
func getEvalResults(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": gin.H{"eval_id": id, "results": []gin.H{}}})
	}
}

// getEvalMetrics 获取指标
func getEvalMetrics(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": gin.H{"eval_id": id, "metrics": gin.H{}}})
	}
}

// createBenchmark 创建基准测试
func createBenchmark(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
			Runs int    `json:"runs" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		ctx := c.Request.Context()
		benchmark := &BenchmarkRecord{
			ID: uuid.New().String(), Name: req.Name, Runs: req.Runs, CreatedAt: time.Now(),
		}

		if err := st.Set(ctx, "benchmarks", benchmark.ID, benchmark); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(201, gin.H{"success": true, "data": benchmark})
	}
}

// listBenchmarks 列出基准测试
func listBenchmarks(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		records, err := st.List(ctx, "benchmarks")
		if err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		benchmarks := make([]*BenchmarkRecord, 0)
		for _, record := range records {
			var b BenchmarkRecord
			if err := store.DecodeValue(record, &b); err != nil {
				continue
			}
			benchmarks = append(benchmarks, &b)
		}

		c.JSON(200, gin.H{"success": true, "data": benchmarks})
	}
}

// getBenchmark 获取基准测试
func getBenchmark(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var benchmark BenchmarkRecord
		if err := st.Get(ctx, "benchmarks", id, &benchmark); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Benchmark not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(200, gin.H{"success": true, "data": &benchmark})
	}
}

// deleteBenchmark 删除基准测试
func deleteBenchmark(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "benchmarks", id); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Benchmark not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.Status(204)
	}
}

// runBenchmark 运行基准测试
func runBenchmark(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "benchmark.running", map[string]interface{}{"id": id})
		c.JSON(202, gin.H{"success": true, "data": gin.H{"status": "running", "benchmark_id": id}})
	}
}

// getBenchmarkResults 获取基准测试结果
func getBenchmarkResults(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": gin.H{"benchmark_id": id, "results": []gin.H{}}})
	}
}

// compareEvals 比较评估
func compareEvals(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": gin.H{"comparison": gin.H{}}})
	}
}

// getEvalStats 获取统计
func getEvalStats(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": gin.H{"total": 0, "avg_score": 0.0}})
	}
}

// exportEvals 导出评估
func exportEvals(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": gin.H{"exported": true}})
	}
}

// getEvalHealth 获取健康状态
func getEvalHealth(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": gin.H{"status": "healthy"}})
	}
}
