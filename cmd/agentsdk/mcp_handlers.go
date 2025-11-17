package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
)

// MCPServerRecord MCP 服务器记录
type MCPServerRecord struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // stdio, sse, http
	Command   string                 `json:"command,omitempty"`
	Args      []string               `json:"args,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
	Status    string                 `json:"status"` // stopped, running, error
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// registerMCPRoutes 注册 MCP 路由
func registerMCPRoutes(v1 *gin.RouterGroup, st store.Store) {
	mcp := v1.Group("/mcp")
	{
		// 服务器管理
		servers := mcp.Group("/servers")
		{
			servers.GET("", listMCPServers(st))
			servers.POST("", createMCPServer(st))
			servers.GET("/:id", getMCPServer(st))
			servers.PATCH("/:id", updateMCPServer(st))
			servers.DELETE("/:id", deleteMCPServer(st))
			servers.POST("/:id/start", startMCPServer(st))
			servers.POST("/:id/stop", stopMCPServer(st))
			servers.GET("/:id/status", getMCPServerStatus(st))
			servers.GET("/:id/tools", getMCPServerTools(st))
			servers.GET("/:id/resources", getMCPServerResources(st))
			servers.GET("/:id/prompts", getMCPServerPrompts(st))
			servers.POST("/:id/execute", executeMCPServer(st))
		}

		// 注册表
		registry := mcp.Group("/registry")
		{
			registry.GET("", listMCPRegistry(st))
			registry.POST("/:id/install", installMCPServer(st))
			registry.DELETE("/:id/uninstall", uninstallMCPServer(st))
			registry.GET("/:id/info", getMCPRegistryInfo(st))
		}
	}
}

// createMCPServer 创建 MCP 服务器
func createMCPServer(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name     string                 `json:"name" binding:"required"`
			Type     string                 `json:"type" binding:"required"`
			Command  string                 `json:"command"`
			Args     []string               `json:"args"`
			Config   map[string]interface{} `json:"config"`
			Metadata map[string]interface{} `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		ctx := c.Request.Context()
		server := &MCPServerRecord{
			ID: uuid.New().String(), Name: req.Name, Type: req.Type,
			Command: req.Command, Args: req.Args, Config: req.Config,
			Status: "stopped", CreatedAt: time.Now(), UpdatedAt: time.Now(), Metadata: req.Metadata,
		}

		if err := st.Set(ctx, "mcp_servers", server.ID, server); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "mcp.server.created", map[string]interface{}{"id": server.ID, "name": req.Name})
		c.JSON(201, gin.H{"success": true, "data": server})
	}
}

// listMCPServers 列出服务器
func listMCPServers(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		records, err := st.List(ctx, "mcp_servers")
		if err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		servers := make([]*MCPServerRecord, 0)
		for _, record := range records {
			var srv MCPServerRecord
			if err := store.DecodeValue(record, &srv); err != nil {
				continue
			}
			servers = append(servers, &srv)
		}

		c.JSON(200, gin.H{"success": true, "data": servers})
	}
}

// getMCPServer 获取服务器
func getMCPServer(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var server MCPServerRecord
		if err := st.Get(ctx, "mcp_servers", id, &server); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "MCP server not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(200, gin.H{"success": true, "data": &server})
	}
}

// updateMCPServer 更新服务器
func updateMCPServer(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Name     *string                `json:"name"`
			Command  *string                `json:"command"`
			Args     []string               `json:"args"`
			Config   map[string]interface{} `json:"config"`
			Metadata map[string]interface{} `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		var server MCPServerRecord
		if err := st.Get(ctx, "mcp_servers", id, &server); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "MCP server not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		if req.Name != nil {
			server.Name = *req.Name
		}
		if req.Command != nil {
			server.Command = *req.Command
		}
		if req.Args != nil {
			server.Args = req.Args
		}
		if req.Config != nil {
			for k, v := range req.Config {
				if server.Config == nil {
					server.Config = make(map[string]interface{})
				}
				server.Config[k] = v
			}
		}
		server.UpdatedAt = time.Now()

		if err := st.Set(ctx, "mcp_servers", id, &server); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "mcp.server.updated", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": &server})
	}
}

// deleteMCPServer 删除服务器
func deleteMCPServer(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "mcp_servers", id); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "MCP server not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "mcp.server.deleted", map[string]interface{}{"id": id})
		c.Status(204)
	}
}

// startMCPServer 启动服务器
func startMCPServer(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var server MCPServerRecord
		if err := st.Get(ctx, "mcp_servers", id, &server); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "MCP server not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		server.Status = "running"
		server.UpdatedAt = time.Now()

		if err := st.Set(ctx, "mcp_servers", id, &server); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "mcp.server.started", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": &server})
	}
}

// stopMCPServer 停止服务器
func stopMCPServer(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var server MCPServerRecord
		if err := st.Get(ctx, "mcp_servers", id, &server); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "MCP server not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		server.Status = "stopped"
		server.UpdatedAt = time.Now()

		if err := st.Set(ctx, "mcp_servers", id, &server); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "mcp.server.stopped", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": &server})
	}
}

// getMCPServerStatus 获取状态
func getMCPServerStatus(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var server MCPServerRecord
		if err := st.Get(ctx, "mcp_servers", id, &server); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "MCP server not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(200, gin.H{"success": true, "data": gin.H{"server_id": id, "status": server.Status}})
	}
}

// getMCPServerTools 获取工具
func getMCPServerTools(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": []string{}, "server_id": id})
	}
}

// getMCPServerResources 获取资源
func getMCPServerResources(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": []string{}, "server_id": id})
	}
}

// getMCPServerPrompts 获取提示
func getMCPServerPrompts(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": []string{}, "server_id": id})
	}
}

// executeMCPServer 执行
func executeMCPServer(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "mcp.server.executed", map[string]interface{}{"id": id})
		c.JSON(202, gin.H{"success": true, "data": gin.H{"server_id": id, "status": "executed"}})
	}
}

// listMCPRegistry 列出注册表
func listMCPRegistry(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": []gin.H{
			{"id": "filesystem", "name": "Filesystem Server", "installed": false},
			{"id": "github", "name": "GitHub Server", "installed": false},
		}})
	}
}

// installMCPServer 安装
func installMCPServer(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "mcp.registry.installed", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": gin.H{"installed": true, "server_id": id}})
	}
}

// uninstallMCPServer 卸载
func uninstallMCPServer(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "mcp.registry.uninstalled", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": gin.H{"uninstalled": true, "server_id": id}})
	}
}

// getMCPRegistryInfo 获取信息
func getMCPRegistryInfo(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": gin.H{
			"id": id, "name": id + " Server", "version": "1.0.0", "description": "MCP Server",
		}})
	}
}
