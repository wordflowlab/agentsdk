package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// getMessages 获取 Session 消息
func getMessages(st store.Store) gin.HandlerFunc {
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
			"data":    session.Messages,
		})
	}
}

// addMessage 添加消息到 Session
func addMessage(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Role    string `json:"role" binding:"required"`
			Content string `json:"content" binding:"required"`
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

		message := types.Message{
			Role:    types.Role(req.Role),
			Content: req.Content,
		}

		session.Messages = append(session.Messages, message)
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

		logging.Info(ctx, "session.message.added", map[string]interface{}{
			"session_id": id,
			"role":       req.Role,
		})

		c.JSON(201, gin.H{
			"success": true,
			"data":    message,
		})
	}
}

// deleteMessage 删除 Session 中的消息
func deleteMessage(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")
		mid := c.Param("mid")

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

		// 简单实现：mid 是消息索引
		// TODO: 使用消息 ID 进行删除
		c.JSON(200, gin.H{
			"success": true,
			"message": "Message deletion not fully implemented (mid: " + mid + ")",
		})
	}
}

// listCheckpoints 列出 Checkpoints
func listCheckpoints(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		sessionID := c.Param("id")

		records, err := st.List(ctx, "checkpoints")
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to list checkpoints: " + err.Error(),
				},
			})
			return
		}

		checkpoints := make([]*CheckpointRecord, 0)
		for _, record := range records {
			var checkpoint CheckpointRecord
			if err := store.DecodeValue(record, &checkpoint); err != nil {
				continue
			}
			if checkpoint.SessionID == sessionID {
				checkpoints = append(checkpoints, &checkpoint)
			}
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    checkpoints,
		})
	}
}

// createCheckpoint 创建 Checkpoint
func createCheckpoint(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		sessionID := c.Param("id")

		var req struct {
			Label string `json:"label"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			// Label 是可选的
		}

		var session SessionRecord
		if err := st.Get(ctx, "sessions", sessionID, &session); err != nil {
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

		checkpoint := &CheckpointRecord{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			Snapshot:  session,
			CreatedAt: time.Now(),
			Label:     req.Label,
		}

		if err := st.Set(ctx, "checkpoints", checkpoint.ID, checkpoint); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to create checkpoint: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "checkpoint.created", map[string]interface{}{
			"checkpoint_id": checkpoint.ID,
			"session_id":    sessionID,
		})

		c.JSON(201, gin.H{
			"success": true,
			"data":    checkpoint,
		})
	}
}

// restoreCheckpoint 恢复 Checkpoint
func restoreCheckpoint(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		sessionID := c.Param("id")

		var req struct {
			CheckpointID string `json:"checkpoint_id" binding:"required"`
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

		var checkpoint CheckpointRecord
		if err := st.Get(ctx, "checkpoints", req.CheckpointID, &checkpoint); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Checkpoint not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get checkpoint: " + err.Error(),
				},
			})
			return
		}

		if checkpoint.SessionID != sessionID {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "bad_request",
					"message": "Checkpoint does not belong to this session",
				},
			})
			return
		}

		// 恢复快照
		restoredSession := checkpoint.Snapshot
		restoredSession.UpdatedAt = time.Now()

		if err := st.Set(ctx, "sessions", sessionID, &restoredSession); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to restore checkpoint: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "checkpoint.restored", map[string]interface{}{
			"checkpoint_id": req.CheckpointID,
			"session_id":    sessionID,
		})

		c.JSON(200, gin.H{
			"success": true,
			"data":    &restoredSession,
		})
	}
}

// getSessionStats 获取 Session 统计
func getSessionStats(st store.Store) gin.HandlerFunc {
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

		stats := map[string]interface{}{
			"session_id":    id,
			"agent_id":      session.AgentID,
			"status":        session.Status,
			"message_count": len(session.Messages),
			"created_at":    session.CreatedAt,
			"updated_at":    session.UpdatedAt,
			"duration":      time.Since(session.CreatedAt).Seconds(),
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    stats,
		})
	}
}

// exportSession 导出 Session
func exportSession(st store.Store) gin.HandlerFunc {
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

		exportData := map[string]interface{}{
			"session":     session,
			"exported_at": time.Now(),
			"format":      "json",
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    exportData,
		})
	}
}

// forkSession 克隆 Session
func forkSession(st store.Store) gin.HandlerFunc {
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

		// 创建新 Session
		forked := &SessionRecord{
			ID:        uuid.New().String(),
			AgentID:   session.AgentID,
			Status:    "active",
			Messages:  append([]types.Message{}, session.Messages...),
			Context:   session.Context,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata: map[string]interface{}{
				"forked_from": id,
			},
		}

		if err := st.Set(ctx, "sessions", forked.ID, forked); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to fork session: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "session.forked", map[string]interface{}{
			"original_id": id,
			"forked_id":   forked.ID,
		})

		c.JSON(201, gin.H{
			"success": true,
			"data":    forked,
		})
	}
}

// getSessionContext 获取 Session 上下文
func getSessionContext(st store.Store) gin.HandlerFunc {
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
			"data":    session.Context,
		})
	}
}

// batchDeleteSessions 批量删除 Sessions
func batchDeleteSessions(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req struct {
			SessionIDs []string `json:"session_ids" binding:"required"`
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

		results := make(map[string]interface{})
		successCount := 0
		failedIDs := []string{}

		for _, id := range req.SessionIDs {
			if err := st.Delete(ctx, "sessions", id); err != nil {
				failedIDs = append(failedIDs, id)
			} else {
				successCount++
			}
		}

		results["total"] = len(req.SessionIDs)
		results["success"] = successCount
		results["failed"] = len(failedIDs)
		if len(failedIDs) > 0 {
			results["failed_ids"] = failedIDs
		}

		logging.Info(ctx, "sessions.batch.deleted", map[string]interface{}{
			"total":   len(req.SessionIDs),
			"success": successCount,
			"failed":  len(failedIDs),
		})

		c.JSON(200, gin.H{
			"success": true,
			"data":    results,
		})
	}
}

// querySessions 查询 Sessions
func querySessions(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req struct {
			AgentID       string     `json:"agent_id"`
			Status        string     `json:"status"`
			CreatedAfter  *time.Time `json:"created_after"`
			CreatedBefore *time.Time `json:"created_before"`
			Limit         int        `json:"limit"`
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

			// 应用过滤器
			if req.AgentID != "" && session.AgentID != req.AgentID {
				continue
			}
			if req.Status != "" && session.Status != req.Status {
				continue
			}
			if req.CreatedAfter != nil && session.CreatedAt.Before(*req.CreatedAfter) {
				continue
			}
			if req.CreatedBefore != nil && session.CreatedAt.After(*req.CreatedBefore) {
				continue
			}

			sessions = append(sessions, &session)

			// 限制数量
			if req.Limit > 0 && len(sessions) >= req.Limit {
				break
			}
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    sessions,
			"count":   len(sessions),
		})
	}
}
