package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// SessionRecord Session 持久化记录
type SessionRecord struct {
	ID        string                 `json:"id"`
	AgentID   string                 `json:"agent_id"`
	Status    string                 `json:"status"` // active, completed, abandoned
	Messages  []types.Message        `json:"messages,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CheckpointRecord Checkpoint 记录
type CheckpointRecord struct {
	ID        string        `json:"id"`
	SessionID string        `json:"session_id"`
	Snapshot  SessionRecord `json:"snapshot"`
	CreatedAt time.Time     `json:"created_at"`
	Label     string        `json:"label,omitempty"`
}

// registerSessionRoutes 注册 Session 路由
func registerSessionRoutes(v1 *gin.RouterGroup, st store.Store) {
	sessions := v1.Group("/sessions")
	{
		// 基础 CRUD
		sessions.POST("", createSession(st))
		sessions.GET("", listSessions(st))
		sessions.GET("/:id", getSession(st))
		sessions.PATCH("/:id", updateSession(st))
		sessions.DELETE("/:id", deleteSession(st))
		sessions.POST("/:id/complete", completeSession(st))

		// 消息管理
		sessions.GET("/:id/messages", getMessages(st))
		sessions.POST("/:id/messages", addMessage(st))
		sessions.DELETE("/:id/messages/:mid", deleteMessage(st))

		// Checkpoint
		sessions.GET("/:id/checkpoints", listCheckpoints(st))
		sessions.POST("/:id/checkpoints", createCheckpoint(st))
		sessions.POST("/:id/restore", restoreCheckpoint(st))

		// 其他功能
		sessions.GET("/:id/stats", getSessionStats(st))
		sessions.POST("/:id/export", exportSession(st))
		sessions.POST("/:id/fork", forkSession(st))
		sessions.GET("/:id/context", getSessionContext(st))

		// 批量和查询
		sessions.POST("/batch/delete", batchDeleteSessions(st))
		sessions.POST("/query", querySessions(st))
	}
}

// createSession 创建 Session
func createSession(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			AgentID  string                 `json:"agent_id" binding:"required"`
			Context  map[string]interface{} `json:"context"`
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

		ctx := c.Request.Context()
		record := &SessionRecord{
			ID:        uuid.New().String(),
			AgentID:   req.AgentID,
			Status:    "active",
			Messages:  []types.Message{},
			Context:   req.Context,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata:  req.Metadata,
		}

		if err := st.Set(ctx, "sessions", record.ID, record); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to create session: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "session.created", map[string]interface{}{
			"session_id": record.ID,
			"agent_id":   req.AgentID,
		})

		c.JSON(201, gin.H{
			"success": true,
			"data":    record,
		})
	}
}

// listSessions 列出所有 Sessions
func listSessions(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		agentID := c.Query("agent_id")
		status := c.Query("status")

		records, err := st.List(ctx, "sessions")
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to list sessions: " + err.Error(),
				},
			})
			return
		}

		sessions := make([]*SessionRecord, 0)
		for _, record := range records {
			var session SessionRecord
			if err := store.DecodeValue(record, &session); err != nil {
				continue
			}

			// 过滤
			if agentID != "" && session.AgentID != agentID {
				continue
			}
			if status != "" && session.Status != status {
				continue
			}

			sessions = append(sessions, &session)
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    sessions,
		})
	}
}

// getSession 获取单个 Session
func getSession(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var session SessionRecord
		if err := st.Get(ctx, "sessions", id, &session); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Session not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get session: " + err.Error(),
				},
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    &session,
		})
	}
}

// updateSession 更新 Session
func updateSession(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Context  map[string]interface{} `json:"context"`
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

		var session SessionRecord
		if err := st.Get(ctx, "sessions", id, &session); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Session not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get session: " + err.Error(),
				},
			})
			return
		}

		// 更新字段
		if req.Context != nil {
			for k, v := range req.Context {
				if session.Context == nil {
					session.Context = make(map[string]interface{})
				}
				session.Context[k] = v
			}
		}

		if req.Metadata != nil {
			for k, v := range req.Metadata {
				if session.Metadata == nil {
					session.Metadata = make(map[string]interface{})
				}
				session.Metadata[k] = v
			}
		}

		session.UpdatedAt = time.Now()

		if err := st.Set(ctx, "sessions", id, &session); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to update session: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "session.updated", map[string]interface{}{
			"session_id": id,
		})

		c.JSON(200, gin.H{
			"success": true,
			"data":    &session,
		})
	}
}

// deleteSession 删除 Session
func deleteSession(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "sessions", id); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Session not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to delete session: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "session.deleted", map[string]interface{}{
			"session_id": id,
		})

		c.Status(204)
	}
}

// completeSession 完成 Session
func completeSession(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		updateSessionStatus(c, st, "completed")
	}
}

// updateSessionStatus 更新 Session 状态（辅助函数）
func updateSessionStatus(c *gin.Context, st store.Store, status string) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var session SessionRecord
	if err := st.Get(ctx, "sessions", id, &session); err != nil {
		if err == store.ErrNotFound {
			c.JSON(404, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Session not found",
				},
			})
			return
		}
		c.JSON(500, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get session: " + err.Error(),
			},
		})
		return
	}

	session.Status = status
	session.UpdatedAt = time.Now()

	if err := st.Set(ctx, "sessions", id, &session); err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to update session status: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "session.status.updated", map[string]interface{}{
		"session_id": id,
		"status":     status,
	})

	c.JSON(200, gin.H{
		"success": true,
		"data":    &session,
	})
}
