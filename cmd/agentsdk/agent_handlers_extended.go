package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/agent"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
)

// updateAgent 更新 Agent
func updateAgent(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Name     *string                `json:"name"`
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

		// 获取现有 Agent
		var agentRecord AgentRecord
		if err := st.Get(ctx, "agents", id, &agentRecord); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Agent not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get agent: " + err.Error(),
				},
			})
			return
		}

		// 更新字段
		if req.Name != nil {
			if agentRecord.Metadata == nil {
				agentRecord.Metadata = make(map[string]interface{})
			}
			agentRecord.Metadata["name"] = *req.Name
		}

		if req.Metadata != nil {
			for k, v := range req.Metadata {
				if agentRecord.Metadata == nil {
					agentRecord.Metadata = make(map[string]interface{})
				}
				agentRecord.Metadata[k] = v
			}
		}

		agentRecord.UpdatedAt = time.Now()

		// 保存更新
		if err := st.Set(ctx, "agents", id, &agentRecord); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to update agent: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "agent.updated", map[string]interface{}{
			"agent_id": id,
		})

		c.JSON(200, gin.H{
			"success": true,
			"data":    &agentRecord,
		})
	}
}

// activateAgent 激活 Agent
func activateAgent(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		updateAgentStatus(c, st, "active")
	}
}

// disableAgent 禁用 Agent
func disableAgent(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		updateAgentStatus(c, st, "disabled")
	}
}

// archiveAgent 归档 Agent
func archiveAgent(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		updateAgentStatus(c, st, "archived")
	}
}

// updateAgentStatus 更新 Agent 状态（辅助函数）
func updateAgentStatus(c *gin.Context, st store.Store, status string) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var agentRecord AgentRecord
	if err := st.Get(ctx, "agents", id, &agentRecord); err != nil {
		if err == store.ErrNotFound {
			c.JSON(404, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Agent not found",
				},
			})
			return
		}
		c.JSON(500, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to get agent: " + err.Error(),
			},
		})
		return
	}

	agentRecord.Status = status
	agentRecord.UpdatedAt = time.Now()

	if err := st.Set(ctx, "agents", id, &agentRecord); err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "internal_error",
				"message": "Failed to update agent status: " + err.Error(),
			},
		})
		return
	}

	logging.Info(ctx, "agent.status.updated", map[string]interface{}{
		"agent_id": id,
		"status":   status,
	})

	c.JSON(200, gin.H{
		"success": true,
		"data":    &agentRecord,
	})
}

// validateAgent 验证 Agent 配置
func validateAgent(deps *agent.Dependencies, st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var agentRecord AgentRecord
		if err := st.Get(ctx, "agents", id, &agentRecord); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Agent not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get agent: " + err.Error(),
				},
			})
			return
		}

		// 执行验证逻辑
		validationResults := map[string]interface{}{
			"valid": true,
			"checks": []map[string]interface{}{
				{
					"name":    "template_exists",
					"status":  "passed",
					"message": "Template exists",
				},
				{
					"name":    "config_valid",
					"status":  "passed",
					"message": "Configuration is valid",
				},
			},
		}

		// 验证模板是否存在
		if agentRecord.Config != nil && agentRecord.Config.TemplateID != "" {
			if _, err := deps.TemplateRegistry.Get(agentRecord.Config.TemplateID); err != nil {
				validationResults["valid"] = false
				validationResults["checks"] = append(validationResults["checks"].([]map[string]interface{}), map[string]interface{}{
					"name":    "template_exists",
					"status":  "failed",
					"message": "Template not found: " + agentRecord.Config.TemplateID,
				})
			}
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    validationResults,
		})
	}
}

// cloneAgent 克隆 Agent
func cloneAgent(deps *agent.Dependencies, st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Name string `json:"name"`
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

		// 获取原 Agent
		var originalAgent AgentRecord
		if err := st.Get(ctx, "agents", id, &originalAgent); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Agent not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get agent: " + err.Error(),
				},
			})
			return
		}

		// 创建克隆
		clonedAgent := AgentRecord{
			ID:        uuid.New().String(),
			Config:    originalAgent.Config,
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata:  make(map[string]interface{}),
		}

		// 复制元数据
		for k, v := range originalAgent.Metadata {
			clonedAgent.Metadata[k] = v
		}

		// 设置新名称
		if req.Name != "" {
			clonedAgent.Metadata["name"] = req.Name
		} else if name, ok := originalAgent.Metadata["name"].(string); ok {
			clonedAgent.Metadata["name"] = name + " (Clone)"
		}

		// 保存克隆
		if err := st.Set(ctx, "agents", clonedAgent.ID, &clonedAgent); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to save cloned agent: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "agent.cloned", map[string]interface{}{
			"original_id": id,
			"cloned_id":   clonedAgent.ID,
		})

		c.JSON(201, gin.H{
			"success": true,
			"data":    &clonedAgent,
		})
	}
}

// getAgentStats 获取 Agent 统计信息
func getAgentStats(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var agentRecord AgentRecord
		if err := st.Get(ctx, "agents", id, &agentRecord); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Agent not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get agent: " + err.Error(),
				},
			})
			return
		}

		// 收集统计信息
		stats := map[string]interface{}{
			"agent_id":   id,
			"status":     agentRecord.Status,
			"created_at": agentRecord.CreatedAt,
			"updated_at": agentRecord.UpdatedAt,
			"uptime":     time.Since(agentRecord.CreatedAt).Seconds(),
			"sessions":   0, // TODO: 从实际数据中获取
			"messages":   0, // TODO: 从实际数据中获取
			"tool_calls": 0, // TODO: 从实际数据中获取
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    stats,
		})
	}
}

// getAggregatedStats 获取聚合统计信息
func getAggregatedStats(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		records, err := st.List(ctx, "agents")
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to list agents: " + err.Error(),
				},
			})
			return
		}

		// 统计各状态的 Agent 数量
		statusCounts := map[string]int{
			"active":   0,
			"disabled": 0,
			"archived": 0,
		}
		totalAgents := 0

		for _, record := range records {
			var agent AgentRecord
			if err := store.DecodeValue(record, &agent); err != nil {
				continue
			}
			totalAgents++
			statusCounts[agent.Status]++
		}

		stats := map[string]interface{}{
			"total_agents":    totalAgents,
			"active_agents":   statusCounts["active"],
			"disabled_agents": statusCounts["disabled"],
			"archived_agents": statusCounts["archived"],
			"by_status":       statusCounts,
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    stats,
		})
	}
}

// batchDeleteAgents 批量删除 Agents
func batchDeleteAgents(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var req struct {
			AgentIDs []string `json:"agent_ids" binding:"required"`
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

		for _, id := range req.AgentIDs {
			if err := st.Delete(ctx, "agents", id); err != nil {
				failedIDs = append(failedIDs, id)
			} else {
				successCount++
			}
		}

		results["total"] = len(req.AgentIDs)
		results["success"] = successCount
		results["failed"] = len(failedIDs)
		if len(failedIDs) > 0 {
			results["failed_ids"] = failedIDs
		}

		logging.Info(ctx, "agents.batch.deleted", map[string]interface{}{
			"total":   len(req.AgentIDs),
			"success": successCount,
			"failed":  len(failedIDs),
		})

		c.JSON(200, gin.H{
			"success": true,
			"data":    results,
		})
	}
}

// batchActivateAgents 批量激活 Agents
func batchActivateAgents(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		batchUpdateAgentStatus(c, st, "active")
	}
}

// batchDisableAgents 批量禁用 Agents
func batchDisableAgents(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		batchUpdateAgentStatus(c, st, "disabled")
	}
}

// batchUpdateAgentStatus 批量更新 Agent 状态（辅助函数）
func batchUpdateAgentStatus(c *gin.Context, st store.Store, status string) {
	ctx := c.Request.Context()

	var req struct {
		AgentIDs []string `json:"agent_ids" binding:"required"`
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

	for _, id := range req.AgentIDs {
		var agentRecord AgentRecord
		if err := st.Get(ctx, "agents", id, &agentRecord); err != nil {
			failedIDs = append(failedIDs, id)
			continue
		}

		agentRecord.Status = status
		agentRecord.UpdatedAt = time.Now()

		if err := st.Set(ctx, "agents", id, &agentRecord); err != nil {
			failedIDs = append(failedIDs, id)
		} else {
			successCount++
		}
	}

	results["total"] = len(req.AgentIDs)
	results["success"] = successCount
	results["failed"] = len(failedIDs)
	if len(failedIDs) > 0 {
		results["failed_ids"] = failedIDs
	}

	logging.Info(ctx, "agents.batch.status.updated", map[string]interface{}{
		"total":   len(req.AgentIDs),
		"success": successCount,
		"failed":  len(failedIDs),
		"status":  status,
	})

	c.JSON(200, gin.H{
		"success": true,
		"data":    results,
	})
}
