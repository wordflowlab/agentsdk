package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
)

// WorkingMemoryRecord Working Memory 记录
type WorkingMemoryRecord struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	Key       string                 `json:"key"`
	Value     interface{}            `json:"value"`
	Type      string                 `json:"type"`          // string, number, object, array
	TTL       int                    `json:"ttl,omitempty"` // seconds, 0 = no expiry
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SemanticMemoryRecord Semantic Memory 记录
type SemanticMemoryRecord struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Embedding []float64              `json:"embedding,omitempty"`
	Source    string                 `json:"source,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	AgentID   string                 `json:"agent_id,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ProvenanceRecord Provenance 溯源记录
type ProvenanceRecord struct {
	ID         string                 `json:"id"`
	EntityType string                 `json:"entity_type"` // memory, session, agent
	EntityID   string                 `json:"entity_id"`
	Operation  string                 `json:"operation"` // create, update, delete, access
	Actor      string                 `json:"actor,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Changes    map[string]interface{} `json:"changes,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ConsolidationJob 整合任务记录
type ConsolidationJob struct {
	ID          string                 `json:"id"`
	Status      string                 `json:"status"` // pending, running, completed, failed
	Type        string                 `json:"type"`   // working_to_semantic, cleanup, optimization
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Progress    float64                `json:"progress"` // 0-100
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// registerMemoryRoutes 注册 Memory 路由
func registerMemoryRoutes(v1 *gin.RouterGroup, st store.Store) {
	memory := v1.Group("/memory")
	{
		// Working Memory
		working := memory.Group("/working")
		{
			working.POST("", createWorkingMemory(st))
			working.GET("", listWorkingMemory(st))
			working.GET("/:id", getWorkingMemory(st))
			working.PATCH("/:id", updateWorkingMemory(st))
			working.DELETE("/:id", deleteWorkingMemory(st))
			working.POST("/clear", clearWorkingMemory(st))
		}

		// Semantic Memory
		semantic := memory.Group("/semantic")
		{
			semantic.POST("", createSemanticMemory(st))
			semantic.GET("", listSemanticMemory(st))
			semantic.POST("/search", searchSemanticMemory(st)) // 已在 serve.go 实现
			semantic.GET("/:id", getSemanticMemory(st))
			semantic.DELETE("/:id", deleteSemanticMemory(st))
			semantic.POST("/consolidate", consolidateSemanticMemory(st))
		}

		// Provenance
		provenance := memory.Group("/provenance")
		{
			provenance.GET("/:id", getProvenance(st))
			provenance.POST("/trace", traceProvenance(st))
		}

		// Consolidation
		memory.POST("/consolidate", startConsolidation(st))
		memory.GET("/consolidation/status", getConsolidationStatus(st))
		memory.GET("/consolidation/history", getConsolidationHistory(st))
		memory.POST("/consolidation/cancel", cancelConsolidation(st))
	}
}

// createWorkingMemory 创建 Working Memory
func createWorkingMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SessionID string                 `json:"session_id"`
			AgentID   string                 `json:"agent_id"`
			Key       string                 `json:"key" binding:"required"`
			Value     interface{}            `json:"value" binding:"required"`
			Type      string                 `json:"type"`
			TTL       int                    `json:"ttl"`
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
		now := time.Now()

		record := &WorkingMemoryRecord{
			ID:        uuid.New().String(),
			SessionID: req.SessionID,
			AgentID:   req.AgentID,
			Key:       req.Key,
			Value:     req.Value,
			Type:      req.Type,
			TTL:       req.TTL,
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		}

		if req.TTL > 0 {
			expiresAt := now.Add(time.Duration(req.TTL) * time.Second)
			record.ExpiresAt = &expiresAt
		}

		if err := st.Set(ctx, "working_memory", record.ID, record); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to create working memory: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "working_memory.created", map[string]interface{}{
			"id":  record.ID,
			"key": req.Key,
		})

		c.JSON(201, gin.H{
			"success": true,
			"data":    record,
		})
	}
}

// listWorkingMemory 列出 Working Memory
func listWorkingMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		sessionID := c.Query("session_id")
		agentID := c.Query("agent_id")

		records, err := st.List(ctx, "working_memory")
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to list working memory: " + err.Error(),
				},
			})
			return
		}

		memories := make([]*WorkingMemoryRecord, 0)
		now := time.Now()

		for _, record := range records {
			var memory WorkingMemoryRecord
			if err := store.DecodeValue(record, &memory); err != nil {
				continue
			}

			// 检查是否过期
			if memory.ExpiresAt != nil && memory.ExpiresAt.Before(now) {
				continue
			}

			// 过滤
			if sessionID != "" && memory.SessionID != sessionID {
				continue
			}
			if agentID != "" && memory.AgentID != agentID {
				continue
			}

			memories = append(memories, &memory)
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    memories,
		})
	}
}

// getWorkingMemory 获取单个 Working Memory
func getWorkingMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var memory WorkingMemoryRecord
		if err := st.Get(ctx, "working_memory", id, &memory); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Working memory not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get working memory: " + err.Error(),
				},
			})
			return
		}

		// 检查是否过期
		if memory.ExpiresAt != nil && memory.ExpiresAt.Before(time.Now()) {
			c.JSON(404, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "expired",
					"message": "Working memory has expired",
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

// updateWorkingMemory 更新 Working Memory
func updateWorkingMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Value    interface{}            `json:"value"`
			TTL      *int                   `json:"ttl"`
			Metadata map[string]interface{} `json:"metadata"`
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

		var memory WorkingMemoryRecord
		if err := st.Get(ctx, "working_memory", id, &memory); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Working memory not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get working memory: " + err.Error(),
				},
			})
			return
		}

		// 更新字段
		if req.Value != nil {
			memory.Value = req.Value
		}

		if req.TTL != nil {
			memory.TTL = *req.TTL
			if *req.TTL > 0 {
				expiresAt := time.Now().Add(time.Duration(*req.TTL) * time.Second)
				memory.ExpiresAt = &expiresAt
			} else {
				memory.ExpiresAt = nil
			}
		}

		if req.Metadata != nil {
			for k, v := range req.Metadata {
				if memory.Metadata == nil {
					memory.Metadata = make(map[string]interface{})
				}
				memory.Metadata[k] = v
			}
		}

		memory.UpdatedAt = time.Now()

		if err := st.Set(ctx, "working_memory", id, &memory); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to update working memory: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "working_memory.updated", map[string]interface{}{
			"id": id,
		})

		c.JSON(200, gin.H{
			"success": true,
			"data":    &memory,
		})
	}
}

// deleteWorkingMemory 删除 Working Memory
func deleteWorkingMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "working_memory", id); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Working memory not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to delete working memory: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "working_memory.deleted", map[string]interface{}{
			"id": id,
		})

		c.Status(204)
	}
}

// clearWorkingMemory 清除 Working Memory
func clearWorkingMemory(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req struct {
			SessionID string `json:"session_id"`
			AgentID   string `json:"agent_id"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体，清除所有
		}

		records, err := st.List(ctx, "working_memory")
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to list working memory: " + err.Error(),
				},
			})
			return
		}

		deleted := 0
		for _, record := range records {
			var memory WorkingMemoryRecord
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

			if err := st.Delete(ctx, "working_memory", memory.ID); err == nil {
				deleted++
			}
		}

		logging.Info(ctx, "working_memory.cleared", map[string]interface{}{
			"deleted": deleted,
		})

		c.JSON(200, gin.H{
			"success": true,
			"data": gin.H{
				"deleted": deleted,
			},
		})
	}
}
