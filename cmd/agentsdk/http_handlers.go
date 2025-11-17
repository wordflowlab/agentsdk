package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wordflowlab/agentsdk/pkg/agent"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
	"github.com/wordflowlab/agentsdk/pkg/types"
)

// AgentRecord Agent 持久化记录
type AgentRecord struct {
	ID        string                 `json:"id"`
	Config    *types.AgentConfig     `json:"config"`
	Status    string                 `json:"status"` // active, disabled, archived
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// registerAgentRoutes 注册 Agent 路由
func registerAgentRoutes(v1 *gin.RouterGroup, deps *agent.Dependencies, st store.Store) {
	agents := v1.Group("/agents")
	{
		// CRUD
		agents.POST("", createAgent(deps, st))
		agents.GET("", listAgents(st))
		agents.GET("/:id", getAgent(st))
		agents.PATCH("/:id", updateAgent(st))
		agents.DELETE("/:id", deleteAgent(st))

		// Templates
		agents.GET("/templates", listTemplates(deps))
		agents.GET("/templates/:id", getTemplate(deps))

		// Agent operations
		agents.POST("/:id/activate", activateAgent(st))
		agents.POST("/:id/disable", disableAgent(st))
		agents.POST("/:id/archive", archiveAgent(st))
		agents.POST("/:id/validate", validateAgent(deps, st))
		agents.POST("/:id/clone", cloneAgent(deps, st))

		// Stats
		agents.GET("/:id/stats", getAgentStats(st))
		agents.GET("/stats/aggregated", getAggregatedStats(st))

		// Batch operations
		batch := agents.Group("/batch")
		{
			batch.POST("/delete", batchDeleteAgents(st))
			batch.POST("/activate", batchActivateAgents(st))
			batch.POST("/disable", batchDisableAgents(st))
		}
	}
}

// createAgent 创建 Agent
func createAgent(deps *agent.Dependencies, st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TemplateID    string                     `json:"template_id" binding:"required"`
			Name          string                     `json:"name"`
			ModelConfig   *types.ModelConfig         `json:"model_config"`
			Sandbox       *types.SandboxConfig       `json:"sandbox"`
			Middlewares   []string                   `json:"middlewares"`
			Metadata      map[string]interface{}     `json:"metadata"`
			SkillsPackage *types.SkillsPackageConfig `json:"skills_package"`
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

		// 创建 Agent 配置
		config := &types.AgentConfig{
			TemplateID:    req.TemplateID,
			ModelConfig:   req.ModelConfig,
			Sandbox:       req.Sandbox,
			Middlewares:   req.Middlewares,
			Metadata:      req.Metadata,
			SkillsPackage: req.SkillsPackage,
		}

		// 创建 Agent 实例
		ag, err := agent.Create(ctx, config, deps)
		if err != nil {
			logging.Error(ctx, "agent.create.error", map[string]interface{}{
				"template_id": req.TemplateID,
				"error":       err.Error(),
			})
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to create agent: " + err.Error(),
				},
			})
			return
		}
		defer ag.Close()

		// 保存 Agent 记录
		record := &AgentRecord{
			ID:        ag.ID(),
			Config:    config,
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata: map[string]interface{}{
				"name": req.Name,
			},
		}

		if err := st.Set(ctx, "agents", ag.ID(), record); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to save agent: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "agent.created", map[string]interface{}{
			"agent_id":    ag.ID(),
			"template_id": req.TemplateID,
		})

		c.JSON(201, gin.H{
			"success": true,
			"data":    record,
		})
	}
}

// listAgents 列出所有 Agents
func listAgents(st store.Store) gin.HandlerFunc {
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

		agents := make([]*AgentRecord, 0, len(records))
		for _, record := range records {
			var agent AgentRecord
			if err := store.DecodeValue(record, &agent); err != nil {
				continue
			}
			agents = append(agents, &agent)
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    agents,
		})
	}
}

// getAgent 获取单个 Agent
func getAgent(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var agent AgentRecord
		if err := st.Get(ctx, "agents", id, &agent); err != nil {
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

		c.JSON(200, gin.H{
			"success": true,
			"data":    &agent,
		})
	}
}

// deleteAgent 删除 Agent
func deleteAgent(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "agents", id); err != nil {
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
					"message": "Failed to delete agent: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "agent.deleted", map[string]interface{}{
			"agent_id": id,
		})

		c.Status(204)
	}
}

// listTemplates 列出模板
func listTemplates(deps *agent.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		templates := deps.TemplateRegistry.List()
		c.JSON(200, gin.H{
			"success": true,
			"data":    templates,
		})
	}
}

// getTemplate 获取模板
func getTemplate(deps *agent.Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		template, err := deps.TemplateRegistry.Get(id)
		if err != nil {
			c.JSON(404, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Template not found",
				},
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    template,
		})
	}
}
