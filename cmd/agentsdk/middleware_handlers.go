package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
)

// MiddlewareRecord 中间件记录
type MiddlewareRecord struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // builtin, custom
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"` // 执行顺序
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// registerMiddlewareRoutes 注册 Middleware 路由
func registerMiddlewareRoutes(v1 *gin.RouterGroup, st store.Store) {
	mw := v1.Group("/middlewares")
	{
		// 基础 CRUD
		mw.POST("", createMiddleware(st))
		mw.GET("", listMiddlewares(st))
		mw.GET("/:id", getMiddleware(st))
		mw.PATCH("/:id", updateMiddleware(st))
		mw.DELETE("/:id", deleteMiddleware(st))

		// 管理操作
		mw.POST("/:id/enable", enableMiddleware(st))
		mw.POST("/:id/disable", disableMiddleware(st))
		mw.POST("/:id/reload", reloadMiddleware(st))
		mw.GET("/:id/stats", getMiddlewareStats(st))

		// Registry
		registry := mw.Group("/registry")
		{
			registry.GET("", listMiddlewareRegistry(st))
			registry.POST("/:id/install", installMiddleware(st))
			registry.DELETE("/:id/uninstall", uninstallMiddleware(st))
			registry.GET("/:id/info", getMiddlewareInfo(st))
			registry.POST("/reload-all", reloadAllMiddlewares(st))
		}
	}
}

// createMiddleware 创建中间件
func createMiddleware(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string                 `json:"name" binding:"required"`
			Type        string                 `json:"type"`
			Description string                 `json:"description"`
			Config      map[string]interface{} `json:"config"`
			Priority    int                    `json:"priority"`
			Metadata    map[string]interface{} `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		ctx := c.Request.Context()
		mw := &MiddlewareRecord{
			ID: uuid.New().String(), Name: req.Name, Type: req.Type,
			Description: req.Description, Config: req.Config, Enabled: true,
			Priority: req.Priority, CreatedAt: time.Now(), UpdatedAt: time.Now(), Metadata: req.Metadata,
		}

		if err := st.Set(ctx, "middlewares", mw.ID, mw); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "middleware.created", map[string]interface{}{"id": mw.ID, "name": req.Name})
		c.JSON(201, gin.H{"success": true, "data": mw})
	}
}

// listMiddlewares 列出中间件
func listMiddlewares(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		records, err := st.List(ctx, "middlewares")
		if err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		middlewares := make([]*MiddlewareRecord, 0)
		for _, record := range records {
			var mw MiddlewareRecord
			if err := store.DecodeValue(record, &mw); err != nil {
				continue
			}
			middlewares = append(middlewares, &mw)
		}

		c.JSON(200, gin.H{"success": true, "data": middlewares})
	}
}

// getMiddleware 获取中间件
func getMiddleware(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var mw MiddlewareRecord
		if err := st.Get(ctx, "middlewares", id, &mw); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Middleware not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		c.JSON(200, gin.H{"success": true, "data": &mw})
	}
}

// updateMiddleware 更新中间件
func updateMiddleware(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var req struct {
			Name        *string                `json:"name"`
			Description *string                `json:"description"`
			Config      map[string]interface{} `json:"config"`
			Priority    *int                   `json:"priority"`
			Metadata    map[string]interface{} `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		var mw MiddlewareRecord
		if err := st.Get(ctx, "middlewares", id, &mw); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Middleware not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		if req.Name != nil {
			mw.Name = *req.Name
		}
		if req.Description != nil {
			mw.Description = *req.Description
		}
		if req.Config != nil {
			for k, v := range req.Config {
				if mw.Config == nil {
					mw.Config = make(map[string]interface{})
				}
				mw.Config[k] = v
			}
		}
		if req.Priority != nil {
			mw.Priority = *req.Priority
		}
		mw.UpdatedAt = time.Now()

		if err := st.Set(ctx, "middlewares", id, &mw); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "middleware.updated", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": &mw})
	}
}

// deleteMiddleware 删除中间件
func deleteMiddleware(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		if err := st.Delete(ctx, "middlewares", id); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Middleware not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "middleware.deleted", map[string]interface{}{"id": id})
		c.Status(204)
	}
}

// enableMiddleware 启用中间件
func enableMiddleware(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var mw MiddlewareRecord
		if err := st.Get(ctx, "middlewares", id, &mw); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Middleware not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		mw.Enabled = true
		mw.UpdatedAt = time.Now()

		if err := st.Set(ctx, "middlewares", id, &mw); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "middleware.enabled", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": &mw})
	}
}

// disableMiddleware 禁用中间件
func disableMiddleware(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		var mw MiddlewareRecord
		if err := st.Get(ctx, "middlewares", id, &mw); err != nil {
			if err == store.ErrNotFound {
				c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "not_found", "message": "Middleware not found"}})
				return
			}
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		mw.Enabled = false
		mw.UpdatedAt = time.Now()

		if err := st.Set(ctx, "middlewares", id, &mw); err != nil {
			c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "internal_error", "message": err.Error()}})
			return
		}

		logging.Info(ctx, "middleware.disabled", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": &mw})
	}
}

// reloadMiddleware 重新加载中间件
func reloadMiddleware(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "middleware.reloaded", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": gin.H{"reloaded": true, "middleware_id": id}})
	}
}

// getMiddlewareStats 获取统计
func getMiddlewareStats(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": gin.H{
			"middleware_id": id, "calls": 0, "errors": 0, "avg_time_ms": 0.0,
		}})
	}
}

// listMiddlewareRegistry 列出注册表
func listMiddlewareRegistry(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": []gin.H{
			{"id": "logging", "name": "Logging Middleware", "builtin": true},
			{"id": "auth", "name": "Auth Middleware", "builtin": true},
			{"id": "ratelimit", "name": "Rate Limit Middleware", "builtin": false},
		}})
	}
}

// installMiddleware 安装中间件
func installMiddleware(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "middleware.installed", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": gin.H{"installed": true, "middleware_id": id}})
	}
}

// uninstallMiddleware 卸载中间件
func uninstallMiddleware(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		logging.Info(c.Request.Context(), "middleware.uninstalled", map[string]interface{}{"id": id})
		c.JSON(200, gin.H{"success": true, "data": gin.H{"uninstalled": true, "middleware_id": id}})
	}
}

// getMiddlewareInfo 获取信息
func getMiddlewareInfo(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(200, gin.H{"success": true, "data": gin.H{
			"id": id, "name": id + " Middleware", "version": "1.0.0", "builtin": false,
		}})
	}
}

// reloadAllMiddlewares 重新加载所有
func reloadAllMiddlewares(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logging.Info(c.Request.Context(), "middleware.all.reloaded", nil)
		c.JSON(200, gin.H{"success": true, "data": gin.H{"reloaded": true, "count": 0}})
	}
}
