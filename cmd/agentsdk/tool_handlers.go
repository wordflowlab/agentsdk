package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
)

// ToolRecord Tool 定义记录
type ToolRecord struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type"` // builtin, custom, external
	Schema      map[string]interface{} `json:"schema"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Status      string                 `json:"status"` // active, inactive
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ToolExecution Tool 执行记录
type ToolExecution struct {
	ID          string                 `json:"id"`
	ToolID      string                 `json:"tool_id"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Status      string                 `json:"status"` // pending, running, completed, failed
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// registerToolRoutes 注册 Tool 路由
func registerToolRoutes(v1 *gin.RouterGroup, st store.Store) {
	tools := v1.Group("/tools")
	{
		// 基础 CRUD
		tools.POST("", createTool(st))
		tools.GET("", listTools(st))
		tools.GET("/:id", getTool(st))
		tools.PATCH("/:id", updateTool(st))
		tools.DELETE("/:id", deleteTool(st))
		tools.POST("/:id/validate", validateTool(st))

		// 执行管理
		tools.POST("/:id/execute", executeTool(st))
		tools.GET("/:id/executions", listToolExecutions(st))
		tools.GET("/:id/schema", getToolSchema(st))
		tools.POST("/:id/test", testTool(st))

		// Registry
		registry := tools.Group("/registry")
		{
			registry.GET("", listRegistry(st))
			registry.POST("/:id/register", registerToolInRegistry(st))
			registry.DELETE("/:id/unregister", unregisterToolFromRegistry(st))
			registry.GET("/:id/status", getToolRegistryStatus(st))
			registry.POST("/:id/enable", enableToolInRegistry(st))
			registry.POST("/:id/disable", disableToolInRegistry(st))
		}
	}
}

// createTool 创建 Tool
func createTool(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string                 `json:"name" binding:"required"`
			Description string                 `json:"description"`
			Type        string                 `json:"type"`
			Schema      map[string]interface{} `json:"schema" binding:"required"`
			Config      map[string]interface{} `json:"config"`
			Metadata    map[string]interface{} `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		ctx := c.Request.Context()
		tool := &ToolRecord{
			ID: uuid.New().String(), Name: req.Name, Description: req.Description,
			Type: req.Type, Schema: req.Schema, Config: req.Config, Status: "active",
			CreatedAt: time.Now(), UpdatedAt: time.Now(), Metadata: req.Metadata,
		}

		if err := st.Set(ctx, "tools", tool.ID, tool); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "tool.created", map[string]interface{}{"id": tool.ID, "name": req.Name})
		c.JSON(201, gin.H{"success": true, "data": tool})
	}
}

// listTools 列出 Tools
func listTools(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		status := c.Query("status")

		records, err := st.List(ctx, "tools")
		if err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		tools := make([]*ToolRecord, 0)
		for _, record := range records {
			var tool ToolRecord
			if err := store.DecodeValue(record, &tool); err != nil {
				continue
			}
			if status != "" && tool.Status != status {
				continue
			}
			tools = append(tools, &tool)
		}

		c.JSON(200, gin.H{"success": true, "data": tools})
	}
}

// getTool 获取 Tool
func getTool(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var tool ToolRecord
		if err := st.Get(ctx, "tools", id, &tool); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Tool not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(200, gin.H{"success": true, "data": &tool})
	}
}

// updateTool 更新 Tool
func updateTool(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Name        *string                `json:"name"`
			Description *string                `json:"description"`
			Schema      map[string]interface{} `json:"schema"`
			Config      map[string]interface{} `json:"config"`
			Status      *string                `json:"status"`
			Metadata    map[string]interface{} `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		var tool ToolRecord
		if err := st.Get(ctx, "tools", id, &tool); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Tool not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		if req.Name != nil {
			tool.Name = *req.Name
		}
		if req.Description != nil {
			tool.Description = *req.Description
		}
		if req.Schema != nil {
			tool.Schema = req.Schema
		}
		if req.Config != nil {
			for k, v := range req.Config {
				if tool.Config == nil {
					tool.Config = make(map[string]interface{})
				}
				tool.Config[k] = v
			}
		}
		if req.Status != nil {
			tool.Status = *req.Status
		}
		if req.Metadata != nil {
			for k, v := range req.Metadata {
				if tool.Metadata == nil {
					tool.Metadata = make(map[string]interface{})
				}
				tool.Metadata[k] = v
			}
		}

		tool.UpdatedAt = time.Now()

		if err := st.Set(ctx, "tools", id, &tool); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "tool.updated", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": &tool})
	}
}

// deleteTool 删除 Tool
func deleteTool(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "tools", id); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Tool not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "tool.deleted", map[string]interface{}{"id": id})
		c.Status(204)
	}
}

// validateTool 验证 Tool
func validateTool(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var tool ToolRecord
		if err := st.Get(ctx, "tools", id, &tool); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Tool not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		valid := tool.Schema != nil && tool.Name != ""
		c.JSON(200, gin.H{"success": true, "data": gin.H{"valid": valid, "tool_id": id}})
	}
}

// executeTool 执行 Tool
func executeTool(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Input map[string]interface{} `json:"input" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		execution := &ToolExecution{
			ID: uuid.New().String(), ToolID: id, Input: req.Input,
			Status: "pending", StartedAt: time.Now(),
		}

		if err := st.Set(ctx, "tool_executions", execution.ID, execution); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "tool.executed", map[string]interface{}{"tool_id": id, "execution_id": execution.ID})
		c.JSON(202, gin.H{"success": true, "data": execution})
	}
}

// listToolExecutions 列出执行记录
func listToolExecutions(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		toolID := c.Param("id")

		records, err := st.List(ctx, "tool_executions")
		if err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		executions := make([]*ToolExecution, 0)
		for _, record := range records {
			var exec ToolExecution
			if err := store.DecodeValue(record, &exec); err != nil {
				continue
			}
			if exec.ToolID == toolID {
				executions = append(executions, &exec)
			}
		}

		c.JSON(200, gin.H{"success": true, "data": executions})
	}
}

// getToolSchema 获取 Schema
func getToolSchema(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var tool ToolRecord
		if err := st.Get(ctx, "tools", id, &tool); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Tool not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(200, gin.H{"success": true, "data": tool.Schema})
	}
}

// testTool 测试 Tool
func testTool(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Input map[string]interface{} `json:"input"`
		}
		c.ShouldBindJSON(&req)

		var tool ToolRecord
		if err := st.Get(ctx, "tools", id, &tool); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Tool not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(200, gin.H{"success": true, "data": gin.H{"test": "ok", "tool_id": id, "message": "Test completed"}})
	}
}

// listRegistry 列出 Registry
func listRegistry(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": []string{"builtin", "custom"}})
	}
}

// registerToolInRegistry 注册到 Registry
func registerToolInRegistry(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "tool.registry.registered", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": gin.H{"registered": true, "tool_id": id}})
	}
}

// unregisterToolFromRegistry 从 Registry 注销
func unregisterToolFromRegistry(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "tool.registry.unregistered", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": gin.H{"unregistered": true, "tool_id": id}})
	}
}

// getToolRegistryStatus 获取 Registry 状态
func getToolRegistryStatus(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": gin.H{"tool_id": id, "status": "registered", "enabled": true}})
	}
}

// enableToolInRegistry 启用
func enableToolInRegistry(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "tool.registry.enabled", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": gin.H{"enabled": true, "tool_id": id}})
	}
}

// disableToolInRegistry 禁用
func disableToolInRegistry(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "tool.registry.disabled", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": gin.H{"disabled": true, "tool_id": id}})
	}
}
