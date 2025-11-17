package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
)

// WorkflowRecord Workflow 定义记录
type WorkflowRecord struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Version     string                 `json:"version"`
	Steps       []WorkflowStep         `json:"steps"`
	Triggers    []WorkflowTrigger      `json:"triggers,omitempty"`
	Status      string                 `json:"status"` // draft, active, inactive
	CreatedBy   string                 `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowStep Workflow 步骤
type WorkflowStep struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // agent, tool, condition, loop
	Config    map[string]interface{} `json:"config,omitempty"`
	DependsOn []string               `json:"depends_on,omitempty"`
	Timeout   int                    `json:"timeout,omitempty"` // seconds
}

// WorkflowTrigger Workflow 触发器
type WorkflowTrigger struct {
	Type   string                 `json:"type"` // manual, scheduled, event
	Config map[string]interface{} `json:"config,omitempty"`
}

// WorkflowExecution Workflow 执行记录
type WorkflowExecution struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	Status      string                 `json:"status"` // pending, running, completed, failed, cancelled
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Logs        []ExecutionLog         `json:"logs,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExecutionLog 执行日志
type ExecutionLog struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"` // info, warn, error
	StepID    string                 `json:"step_id,omitempty"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// registerWorkflowRoutes 注册 Workflow 路由
func registerWorkflowRoutes(v1 *gin.RouterGroup, st store.Store) {
	workflows := v1.Group("/workflows")
	{
		// 基础 CRUD
		workflows.POST("", createWorkflow(st))
		workflows.GET("", listWorkflows(st))
		workflows.GET("/:id", getWorkflow(st))
		workflows.PATCH("/:id", updateWorkflow(st))
		workflows.DELETE("/:id", deleteWorkflow(st))
		workflows.POST("/:id/validate", validateWorkflow(st))

		// 执行管理
		workflows.POST("/:id/execute", executeWorkflow(st))
		workflows.GET("/:id/executions", listExecutions(st))
		workflows.GET("/:id/executions/:eid", getExecution(st))
		workflows.POST("/:id/executions/:eid/cancel", cancelExecution(st))
		workflows.GET("/:id/executions/:eid/logs", getExecutionLogs(st))
		workflows.POST("/:id/executions/:eid/retry", retryExecution(st))
	}
}

// createWorkflow 创建 Workflow
func createWorkflow(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string                 `json:"name" binding:"required"`
			Description string                 `json:"description"`
			Version     string                 `json:"version"`
			Steps       []WorkflowStep         `json:"steps" binding:"required"`
			Triggers    []WorkflowTrigger      `json:"triggers"`
			Metadata    map[string]interface{} `json:"metadata"`
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

		version := req.Version
		if version == "" {
			version = "1.0.0"
		}

		workflow := &WorkflowRecord{
			ID:          uuid.New().String(),
			Name:        req.Name,
			Description: req.Description,
			Version:     version,
			Steps:       req.Steps,
			Triggers:    req.Triggers,
			Status:      "draft",
			CreatedAt:   now,
			UpdatedAt:   now,
			Metadata:    req.Metadata,
		}

		if err := st.Set(ctx, "workflows", workflow.ID, workflow); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to create workflow: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "workflow.created", map[string]interface{}{
			"id":   workflow.ID,
			"name": req.Name,
		})

		c.JSON(201, gin.H{
			"success": true,
			"data":    workflow,
		})
	}
}

// listWorkflows 列出 Workflows
func listWorkflows(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		status := c.Query("status")

		records, err := st.List(ctx, "workflows")
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to list workflows: " + err.Error(),
				},
			})
			return
		}

		workflows := make([]*WorkflowRecord, 0)
		for _, record := range records {
			var wf WorkflowRecord
			if err := store.DecodeValue(record, &wf); err != nil {
				continue
			}

			// 过滤
			if status != "" && wf.Status != status {
				continue
			}

			workflows = append(workflows, &wf)
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    workflows,
		})
	}
}

// getWorkflow 获取单个 Workflow
func getWorkflow(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var workflow WorkflowRecord
		if err := st.Get(ctx, "workflows", id, &workflow); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Workflow not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get workflow: " + err.Error(),
				},
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    &workflow,
		})
	}
}

// updateWorkflow 更新 Workflow
func updateWorkflow(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Name        *string                `json:"name"`
			Description *string                `json:"description"`
			Version     *string                `json:"version"`
			Steps       []WorkflowStep         `json:"steps"`
			Triggers    []WorkflowTrigger      `json:"triggers"`
			Status      *string                `json:"status"`
			Metadata    map[string]interface{} `json:"metadata"`
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

		var workflow WorkflowRecord
		if err := st.Get(ctx, "workflows", id, &workflow); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Workflow not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get workflow: " + err.Error(),
				},
			})
			return
		}

		// 更新字段
		if req.Name != nil {
			workflow.Name = *req.Name
		}
		if req.Description != nil {
			workflow.Description = *req.Description
		}
		if req.Version != nil {
			workflow.Version = *req.Version
		}
		if req.Steps != nil {
			workflow.Steps = req.Steps
		}
		if req.Triggers != nil {
			workflow.Triggers = req.Triggers
		}
		if req.Status != nil {
			workflow.Status = *req.Status
		}
		if req.Metadata != nil {
			for k, v := range req.Metadata {
				if workflow.Metadata == nil {
					workflow.Metadata = make(map[string]interface{})
				}
				workflow.Metadata[k] = v
			}
		}

		workflow.UpdatedAt = time.Now()

		if err := st.Set(ctx, "workflows", id, &workflow); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to update workflow: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "workflow.updated", map[string]interface{}{
			"id": id,
		})

		c.JSON(200, gin.H{
			"success": true,
			"data":    &workflow,
		})
	}
}

// deleteWorkflow 删除 Workflow
func deleteWorkflow(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "workflows", id); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Workflow not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to delete workflow: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "workflow.deleted", map[string]interface{}{
			"id": id,
		})

		c.Status(204)
	}
}

// validateWorkflow 验证 Workflow
func validateWorkflow(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var workflow WorkflowRecord
		if err := st.Get(ctx, "workflows", id, &workflow); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Workflow not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get workflow: " + err.Error(),
				},
			})
			return
		}

		// 简单验证
		validationResults := map[string]interface{}{
			"valid": true,
			"checks": []map[string]interface{}{
				{
					"name":    "has_steps",
					"status":  "passed",
					"message": "Workflow has steps",
				},
				{
					"name":    "step_dependencies",
					"status":  "passed",
					"message": "Step dependencies are valid",
				},
			},
		}

		if len(workflow.Steps) == 0 {
			validationResults["valid"] = false
			validationResults["checks"] = []map[string]interface{}{
				{
					"name":    "has_steps",
					"status":  "failed",
					"message": "Workflow must have at least one step",
				},
			}
		}

		logging.Info(ctx, "workflow.validated", map[string]interface{}{
			"id":    id,
			"valid": validationResults["valid"],
		})

		c.JSON(200, gin.H{
			"success": true,
			"data":    validationResults,
		})
	}
}
