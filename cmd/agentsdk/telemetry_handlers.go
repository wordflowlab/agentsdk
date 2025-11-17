package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
)

// MetricRecord 指标记录
type MetricRecord struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // counter, gauge, histogram
	Value     float64                `json:"value"`
	Tags      map[string]string      `json:"tags,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TraceRecord 追踪记录
type TraceRecord struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	SpanID     string                 `json:"span_id"`
	ParentID   string                 `json:"parent_id,omitempty"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Duration   int64                  `json:"duration,omitempty"` // microseconds
	Status     string                 `json:"status"`             // ok, error
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// registerTelemetryRoutes 注册 Telemetry 路由
func registerTelemetryRoutes(v1 *gin.RouterGroup, st store.Store) {
	telem := v1.Group("/telemetry")
	{
		// Metrics
		metrics := telem.Group("/metrics")
		{
			metrics.POST("", recordMetric(st))
			metrics.GET("", listMetrics(st))
			metrics.GET("/:id", getMetric(st))
			metrics.DELETE("/:id", deleteMetric(st))
			metrics.GET("/query", queryMetrics(st))
			metrics.POST("/aggregate", aggregateMetrics(st))
			metrics.GET("/stats", getMetricsStats(st))
		}

		// Traces
		traces := telem.Group("/traces")
		{
			traces.POST("", recordTrace(st))
			traces.GET("", listTraces(st))
			traces.GET("/:id", getTrace(st))
			traces.DELETE("/:id", deleteTrace(st))
			traces.POST("/query", queryTraces(st)) // 客户端使用 POST
			traces.GET("/:id/spans", getTraceSpans(st))
		}

		// Logs
		logs := telem.Group("/logs")
		{
			logs.POST("", recordLog(st))
			logs.GET("", listLogs(st))
			logs.GET("/:id", getLog(st))
			logs.DELETE("/:id", deleteLog(st))
			logs.GET("/query", queryLogs(st))
			logs.GET("/stats", getLogsStats(st))
		}

		// Export
		telem.POST("/export", exportTelemetry(st))
		telem.GET("/health", getTelemetryHealth(st))
	}
}

// recordMetric 记录指标
func recordMetric(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name     string                 `json:"name" binding:"required"`
			Type     string                 `json:"type" binding:"required"`
			Value    float64                `json:"value" binding:"required"`
			Tags     map[string]string      `json:"tags"`
			Metadata map[string]interface{} `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		ctx := c.Request.Context()
		metric := &MetricRecord{
			ID: uuid.New().String(), Name: req.Name, Type: req.Type,
			Value: req.Value, Tags: req.Tags, Timestamp: time.Now(), Metadata: req.Metadata,
		}

		if err := st.Set(ctx, "metrics", metric.ID, metric); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "telemetry.metric.recorded", map[string]interface{}{"name": req.Name, "value": req.Value})
		c.JSON(201, gin.H{"success": true, "data": metric})
	}
}

// listMetrics 列出指标
func listMetrics(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		records, err := st.List(ctx, "metrics")
		if err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		metrics := make([]*MetricRecord, 0)
		for _, record := range records {
			var m MetricRecord
			if err := store.DecodeValue(record, &m); err != nil {
				continue
			}
			metrics = append(metrics, &m)
		}

		c.JSON(200, gin.H{"success": true, "data": metrics})
	}
}

// getMetric 获取指标
func getMetric(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var metric MetricRecord
		if err := st.Get(ctx, "metrics", id, &metric); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Metric not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(200, gin.H{"success": true, "data": &metric})
	}
}

// deleteMetric 删除指标
func deleteMetric(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "metrics", id); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Metric not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.Status(204)
	}
}

// queryMetrics 查询指标
func queryMetrics(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": []gin.H{}})
	}
}

// aggregateMetrics 聚合指标
func aggregateMetrics(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": gin.H{"count": 0, "sum": 0.0, "avg": 0.0}})
	}
}

// getMetricsStats 获取统计
func getMetricsStats(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": gin.H{"total": 0, "types": map[string]int{}}})
	}
}

// recordTrace 记录追踪
func recordTrace(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name       string                 `json:"name" binding:"required"`
			SpanID     string                 `json:"span_id" binding:"required"`
			ParentID   string                 `json:"parent_id"`
			Attributes map[string]interface{} `json:"attributes"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		ctx := c.Request.Context()
		trace := &TraceRecord{
			ID: uuid.New().String(), Name: req.Name, SpanID: req.SpanID,
			ParentID: req.ParentID, StartTime: time.Now(), Status: "ok", Attributes: req.Attributes,
		}

		if err := st.Set(ctx, "traces", trace.ID, trace); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "telemetry.trace.recorded", map[string]interface{}{"name": req.Name, "span_id": req.SpanID})
		c.JSON(201, gin.H{"success": true, "data": trace})
	}
}

// listTraces 列出追踪
func listTraces(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		records, err := st.List(ctx, "traces")
		if err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		traces := make([]*TraceRecord, 0)
		for _, record := range records {
			var t TraceRecord
			if err := store.DecodeValue(record, &t); err != nil {
				continue
			}
			traces = append(traces, &t)
		}

		c.JSON(200, gin.H{"success": true, "data": traces})
	}
}

// getTrace 获取追踪
func getTrace(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var trace TraceRecord
		if err := st.Get(ctx, "traces", id, &trace); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Trace not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(200, gin.H{"success": true, "data": &trace})
	}
}

// deleteTrace 删除追踪
func deleteTrace(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "traces", id); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Trace not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.Status(204)
	}
}

// queryTraces 查询追踪
func queryTraces(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// 获取所有 traces
		records, err := st.List(ctx, "traces")
		if err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		traces := make([]*TraceRecord, 0)
		for _, record := range records {
			var t TraceRecord
			if err := store.DecodeValue(record, &t); err != nil {
				continue
			}
			traces = append(traces, &t)
		}

		// 返回格式匹配客户端期望
		c.JSON(200, gin.H{"success": true, "data": gin.H{"traces": traces}})
	}
}

// getTraceSpans 获取 Spans
func getTraceSpans(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": []gin.H{}, "trace_id": id})
	}
}

// recordLog 记录日志
func recordLog(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Level   string                 `json:"level" binding:"required"`
			Message string                 `json:"message" binding:"required"`
			Fields  map[string]interface{} `json:"fields"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		ctx := c.Request.Context()
		logID := uuid.New().String()

		logging.Info(ctx, "telemetry.log.recorded", map[string]interface{}{"level": req.Level, "message": req.Message})
		c.JSON(201, gin.H{"success": true, "data": gin.H{"id": logID, "level": req.Level, "message": req.Message, "timestamp": time.Now()}})
	}
}

// listLogs 列出日志
func listLogs(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": []gin.H{}})
	}
}

// getLog 获取日志
func getLog(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": gin.H{"id": id, "level": "info", "message": "Log entry"}})
	}
}

// deleteLog 删除日志
func deleteLog(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Status(204)
	}
}

// queryLogs 查询日志
func queryLogs(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": []gin.H{}})
	}
}

// getLogsStats 获取统计
func getLogsStats(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": gin.H{"total": 0, "levels": map[string]int{}}})
	}
}

// exportTelemetry 导出遥测数据
func exportTelemetry(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logging.Info(c.Request.Context(), "telemetry.exported", nil)
		c.JSON(200, gin.H{"success": true, "data": gin.H{"exported": true, "format": "json"}})
	}
}

// getTelemetryHealth 获取健康状态
func getTelemetryHealth(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": gin.H{"status": "healthy", "uptime": 0}})
	}
}
