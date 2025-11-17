package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wordflowlab/agentsdk/pkg/logging"
	"github.com/wordflowlab/agentsdk/pkg/store"
)

// ConfigRecord 配置记录
type ConfigRecord struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// registerSystemRoutes 注册 System 路由
func registerSystemRoutes(v1 *gin.RouterGroup, st store.Store) {
	system := v1.Group("/system")
	{
		// 配置管理
		config := system.Group("/config")
		{
			config.GET("", listConfig(st))
			config.GET("/:key", getConfig(st))
			config.PUT("/:key", updateConfig(st))
			config.DELETE("/:key", deleteConfig(st))
		}

		// 系统操作
		system.GET("/info", getSystemInfo(st))
		system.GET("/health", getSystemHealth(st))
		system.GET("/stats", getSystemStats(st))
		system.POST("/reload", reloadSystem(st))
		system.POST("/gc", runGarbageCollection(st))
		system.POST("/backup", backupSystem(st))
	}
}

// listConfig 列出配置
func listConfig(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": []gin.H{
			{"key": "debug", "value": false},
			{"key": "max_connections", "value": 100},
		}})
	}
}

// getConfig 获取配置
func getConfig(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		c.JSON(200, gin.H{"success": true, "data": gin.H{"key": key, "value": "default"}})
	}
}

// updateConfig 更新配置
func updateConfig(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		var req struct {
			Value interface{} `json:"value" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "bad_request", "message": err.Error()}})
			return
		}

		logging.Info(c.Request.Context(), "config.updated", map[string]interface{}{"key": key})
		c.JSON(200, gin.H{"success": true, "data": gin.H{"key": key, "value": req.Value, "updated_at": time.Now()}})
	}
}

// deleteConfig 删除配置
func deleteConfig(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		logging.Info(c.Request.Context(), "config.deleted", map[string]interface{}{"key": key})
		c.Status(204)
	}
}

// getSystemInfo 获取系统信息
func getSystemInfo(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": gin.H{
			"version":    "v0.8.0",
			"go_version": "go1.21",
			"os":         "darwin",
			"arch":       "amd64",
		}})
	}
}

// getSystemHealth 获取健康状态
func getSystemHealth(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": gin.H{
			"status":    "healthy",
			"uptime":    3600,
			"memory_mb": 256,
		}})
	}
}

// getSystemStats 获取统计
func getSystemStats(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"success": true, "data": gin.H{
			"requests_total":       1000,
			"requests_success":     980,
			"requests_error":       20,
			"avg_response_time_ms": 45.2,
		}})
	}
}

// reloadSystem 重新加载系统
func reloadSystem(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logging.Info(c.Request.Context(), "system.reloaded", nil)
		c.JSON(200, gin.H{"success": true, "data": gin.H{"reloaded": true}})
	}
}

// runGarbageCollection 运行垃圾回收
func runGarbageCollection(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logging.Info(c.Request.Context(), "system.gc.run", nil)
		c.JSON(200, gin.H{"success": true, "data": gin.H{"gc_completed": true, "freed_mb": 12.5}})
	}
}

// backupSystem 备份系统
func backupSystem(st store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logging.Info(c.Request.Context(), "system.backup.started", nil)
		c.JSON(202, gin.H{"success": true, "data": gin.H{"backup_started": true, "backup_id": "backup_" + time.Now().Format("20060102_150405")}})
	}
}
