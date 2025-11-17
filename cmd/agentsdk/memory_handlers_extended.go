package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
)

// createSemanticMemory 创建 Semantic Memory
func createSemanticMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Content   string                 `json:"content" binding:"required"`
			Source    string                 `json:"source"`
			SessionID string                 `json:"session_id"`
			AgentID   string                 `json:"agent_id"`
			Tags      []string               `json:"tags"`
			Metadata  map[string]interface{} `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "bad_request",
					"message": err.Error(),
				},
			})
			return
		}

		ctx := c.Request.Context()
		record := &SemanticMemoryRecord{
			ID:        uuid.New().String(),
			Content:   req.Content,
			Source:    req.Source,
			SessionID: req.SessionID,
			AgentID:   req.AgentID,
			Tags:      req.Tags,
			CreatedAt: time.Now(),
			Metadata:  req.Metadata,
			// TODO: 生成 embedding
		}

		if err := st.Set(ctx, "semantic_memory", record.ID, record); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to create semantic memory: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "semantic_memory.created", map[string]interface{}{
			"id": record.ID,
		})

		c.JSON(201, gin.H{
			"success": true,
			"data":    record,
		})
	}
}

// listSemanticMemory 列出 Semantic Memory
func listSemanticMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		sessionID := c.Query("session_id")
		agentID := c.Query("agent_id")
		tag := c.Query("tag")

		records, err := st.List(ctx, "semantic_memory")
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to list semantic memory: " + err.Error(),
				},
			})
			return
		}

		memories := make([]*SemanticMemoryRecord, 0)
		for _, record := range records {
			var memory SemanticMemoryRecord
			if err := store.DecodeValue(record, &memory); err != nil {
				continue
			}

			// 过滤
			if sessionID != "" && memory.SessionID != sessionID {
				continue
			}
			if agentID != "" && memory.AgentID != agentID {
				continue
			}
			if tag != "" {
				found := false
				for _, t := range memory.Tags {
					if t == tag {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			memories = append(memories, &memory)
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    memories,
		})
	}
}

// searchSemanticMemory 搜索 Semantic Memory
func searchSemanticMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Query     string `json:"query" binding:"required"`
			Limit     int    `json:"limit"`
			SessionID string `json:"session_id"`
			AgentID   string `json:"agent_id"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "bad_request",
					"message": err.Error(),
				},
			})
			return
		}

		ctx := c.Request.Context()

		// 简单实现：基于内容匹配
		// TODO: 使用 embedding 进行相似度搜索
		records, err := st.List(ctx, "semantic_memory")
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to search semantic memory: " + err.Error(),
				},
			})
			return
		}

		results := make([]*SemanticMemoryRecord, 0)
		for _, record := range records {
			var memory SemanticMemoryRecord
			if err := store.DecodeValue(record, &memory); err != nil {
				continue
			}

			// 应用过滤
			if req.SessionID != "" && memory.SessionID != req.SessionID {
				continue
			}
			if req.AgentID != "" && memory.AgentID != req.AgentID {
				continue
			}

			// 简单的内容匹配
			// TODO: 使用向量相似度
			results = append(results, &memory)

			if req.Limit > 0 && len(results) >= req.Limit {
				break
			}
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    results,
			"count":   len(results),
		})
	}
}

// getSemanticMemory 获取单个 Semantic Memory
func getSemanticMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var memory SemanticMemoryRecord
		if err := st.Get(ctx, "semantic_memory", id, &memory); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Semantic memory not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get semantic memory: " + err.Error(),
				},
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    &memory,
		})
	}
}

// deleteSemanticMemory 删除 Semantic Memory
func deleteSemanticMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "semantic_memory", id); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Semantic memory not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to delete semantic memory: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "semantic_memory.deleted", map[string]interface{}{
			"id": id,
		})

		c.Status(204)
	}
}

// consolidateSemanticMemory 整合 Semantic Memory
func consolidateSemanticMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SessionID string `json:"session_id"`
			AgentID   string `json:"agent_id"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求
		}

		ctx := c.Request.Context()

		// 创建整合任务
		job := &ConsolidationJob{
			ID:        uuid.New().String(),
			Status:    "pending",
			Type:      "semantic_consolidation",
			Progress:  0,
			CreatedAt: time.Now(),
		}

		if err := st.Set(ctx, "consolidation_jobs", job.ID, job); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to create consolidation job: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "semantic_memory.consolidation.started", map[string]interface{}{
			"job_id": job.ID,
		})

		c.JSON(202, gin.H{
			"success": true,
			"data":    job,
		})
	}
}

// getProvenance 获取溯源信息
func getProvenance(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var provenance ProvenanceRecord
		if err := st.Get(ctx, "provenance", id, &provenance); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Provenance not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get provenance: " + err.Error(),
				},
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    &provenance,
		})
	}
}

// traceProvenance 追踪溯源
func traceProvenance(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			EntityType string `json:"entity_type" binding:"required"`
			EntityID   string `json:"entity_id" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "bad_request",
					"message": err.Error(),
				},
			})
			return
		}

		ctx := c.Request.Context()

		records, err := st.List(ctx, "provenance")
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to list provenance: " + err.Error(),
				},
			})
			return
		}

		traces := make([]*ProvenanceRecord, 0)
		for _, record := range records {
			var prov ProvenanceRecord
			if err := store.DecodeValue(record, &prov); err != nil {
				continue
			}

			if prov.EntityType == req.EntityType && prov.EntityID == req.EntityID {
				traces = append(traces, &prov)
			}
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    traces,
			"count":   len(traces),
		})
	}
}

// startConsolidation 启动整合任务
func startConsolidation(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Type      string                 `json:"type" binding:"required"` // working_to_semantic, cleanup, optimization
			SessionID string                 `json:"session_id"`
			AgentID   string                 `json:"agent_id"`
			Metadata  map[string]interface{} `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "bad_request",
					"message": err.Error(),
				},
			})
			return
		}

		ctx := c.Request.Context()

		job := &ConsolidationJob{
			ID:        uuid.New().String(),
			Status:    "pending",
			Type:      req.Type,
			Progress:  0,
			CreatedAt: time.Now(),
			Metadata:  req.Metadata,
		}

		if err := st.Set(ctx, "consolidation_jobs", job.ID, job); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to create consolidation job: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "consolidation.started", map[string]interface{}{
			"job_id": job.ID,
			"type":   req.Type,
		})

		c.JSON(202, gin.H{
			"success": true,
			"data":    job,
		})
	}
}

// getConsolidationStatus 获取整合任务状态
func getConsolidationStatus(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		jobID := c.Query("job_id")

		if jobID == "" {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "bad_request",
					"message": "job_id is required",
				},
			})
			return
		}

		var job ConsolidationJob
		if err := st.Get(ctx, "consolidation_jobs", jobID, &job); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Consolidation job not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get consolidation job: " + err.Error(),
				},
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    &job,
		})
	}
}

// getConsolidationHistory 获取整合历史
func getConsolidationHistory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		records, err := st.List(ctx, "consolidation_jobs")
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to list consolidation jobs: " + err.Error(),
				},
			})
			return
		}

		jobs := make([]*ConsolidationJob, 0)
		for _, record := range records {
			var job ConsolidationJob
			if err := store.DecodeValue(record, &job); err != nil {
				continue
			}
			jobs = append(jobs, &job)
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    jobs,
			"count":   len(jobs),
		})
	}
}

// cancelConsolidation 取消整合任务
func cancelConsolidation(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			JobID string `json:"job_id" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "bad_request",
					"message": err.Error(),
				},
			})
			return
		}

		ctx := c.Request.Context()

		var job ConsolidationJob
		if err := st.Get(ctx, "consolidation_jobs", req.JobID, &job); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Consolidation job not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get consolidation job: " + err.Error(),
				},
			})
			return
		}

		if job.Status == "completed" || job.Status == "failed" {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_state",
					"message": "Cannot cancel completed or failed job",
				},
			})
			return
		}

		job.Status = "cancelled"
		now := time.Now()
		job.CompletedAt = &now

		if err := st.Set(ctx, "consolidation_jobs", req.JobID, &job); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to cancel consolidation job: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "consolidation.cancelled", map[string]interface{}{
			"job_id": req.JobID,
		})

		c.JSON(200, gin.H{
			"success": true,
			"data":    &job,
		})
	}
}
