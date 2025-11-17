package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
)

// executeWorkflow 执行 Workflow
func executeWorkflow(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Context  map[string]interface{} `json:"context"`
			Metadata map[string]interface{} `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			// 允许空请求体
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

		// 创建执行记录
		execution := &WorkflowExecution{
			ID:         uuid.New().String(),
			WorkflowID: id,
			Status:     "pending",
			StartedAt:  time.Now(),
			Context:    req.Context,
			Metadata:   req.Metadata,
			Logs: []ExecutionLog{
				{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "Workflow execution started",
				},
			},
		}

		if err := st.Set(ctx, "workflow_executions", execution.ID, execution); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to create execution: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "workflow.execution.started", map[string]interface{}{
			"workflow_id":  id,
			"execution_id": execution.ID,
		})

		// TODO: 异步执行 workflow
		// 这里应该启动一个 goroutine 来执行 workflow

		c.JSON(202, gin.H{
			"success": true,
			"data":    execution,
		})
	}
}

// listExecutions 列出 Workflow 执行记录
func listExecutions(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		workflowID := c.Param("id")
		status := c.Query("status")

		records, err := st.List(ctx, "workflow_executions")
		if err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to list executions: " + err.Error(),
				},
			})
			return
		}

		executions := make([]*WorkflowExecution, 0)
		for _, record := range records {
			var exec WorkflowExecution
			if err := store.DecodeValue(record, &exec); err != nil {
				continue
			}

			// 过滤
			if exec.WorkflowID != workflowID {
				continue
			}
			if status != "" && exec.Status != status {
				continue
			}

			executions = append(executions, &exec)
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    executions,
		})
	}
}

// getExecution 获取单个执行记录
func getExecution(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		workflowID := c.Param("id")
		executionID := c.Param("eid")

		var execution WorkflowExecution
		if err := st.Get(ctx, "workflow_executions", executionID, &execution); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Execution not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get execution: " + err.Error(),
				},
			})
			return
		}

		if execution.WorkflowID != workflowID {
			c.JSON(404, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Execution does not belong to this workflow",
				},
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    &execution,
		})
	}
}

// cancelExecution 取消执行
func cancelExecution(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		workflowID := c.Param("id")
		executionID := c.Param("eid")

		var execution WorkflowExecution
		if err := st.Get(ctx, "workflow_executions", executionID, &execution); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Execution not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get execution: " + err.Error(),
				},
			})
			return
		}

		if execution.WorkflowID != workflowID {
			c.JSON(404, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Execution does not belong to this workflow",
				},
			})
			return
		}

		if execution.Status == "completed" || execution.Status == "failed" || execution.Status == "cancelled" {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_state",
					"message": "Cannot cancel execution in current state: " + execution.Status,
				},
			})
			return
		}

		execution.Status = "cancelled"
		now := time.Now()
		execution.CompletedAt = &now
		execution.Logs = append(execution.Logs, ExecutionLog{
			Timestamp: now,
			Level:     "info",
			Message:   "Execution cancelled by user",
		})

		if err := st.Set(ctx, "workflow_executions", executionID, &execution); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to cancel execution: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "workflow.execution.cancelled", map[string]interface{}{
			"workflow_id":  workflowID,
			"execution_id": executionID,
		})

		c.JSON(200, gin.H{
			"success": true,
			"data":    &execution,
		})
	}
}

// getExecutionLogs 获取执行日志
func getExecutionLogs(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		workflowID := c.Param("id")
		executionID := c.Param("eid")

		var execution WorkflowExecution
		if err := st.Get(ctx, "workflow_executions", executionID, &execution); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Execution not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get execution: " + err.Error(),
				},
			})
			return
		}

		if execution.WorkflowID != workflowID {
			c.JSON(404, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Execution does not belong to this workflow",
				},
			})
			return
		}

		c.JSON(200, gin.H{
			"success": true,
			"data":    execution.Logs,
		})
	}
}

// retryExecution 重试执行
func retryExecution(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		workflowID := c.Param("id")
		executionID := c.Param("eid")

		var execution WorkflowExecution
		if err := st.Get(ctx, "workflow_executions", executionID, &execution); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "not_found",
						"message": "Execution not found",
					},
				})
				return
			}
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to get execution: " + err.Error(),
				},
			})
			return
		}

		if execution.WorkflowID != workflowID {
			c.JSON(404, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "not_found",
					"message": "Execution does not belong to this workflow",
				},
			})
			return
		}

		if execution.Status != "failed" {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "invalid_state",
					"message": "Can only retry failed executions",
				},
			})
			return
		}

		// 创建新的执行记录
		newExecution := &WorkflowExecution{
			ID:         uuid.New().String(),
			WorkflowID: workflowID,
			Status:     "pending",
			StartedAt:  time.Now(),
			Context:    execution.Context,
			Metadata: map[string]interface{}{
				"retry_of": executionID,
			},
			Logs: []ExecutionLog{
				{
					Timestamp: time.Now(),
					Level:     "info",
					Message:   "Retrying execution " + executionID,
				},
			},
		}

		if err := st.Set(ctx, "workflow_executions", newExecution.ID, newExecution); err != nil {
			c.JSON(500, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "internal_error",
					"message": "Failed to create retry execution: " + err.Error(),
				},
			})
			return
		}

		logging.Info(ctx, "workflow.execution.retried", map[string]interface{}{
			"workflow_id":      workflowID,
			"original_exec_id": executionID,
			"new_execution_id": newExecution.ID,
		})

		c.JSON(202, gin.H{
			"success": true,
			"data":    newExecution,
		})
	}
}
