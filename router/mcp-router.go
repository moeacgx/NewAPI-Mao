package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func registerMCPRoutes(engine *gin.Engine, api *gin.RouterGroup) {
	engine.Any("/mcp", middleware.MCPTransport(), controller.MCPHandler(engine))
	management := api.Group("/mcp", middleware.DisableCache(), middleware.AdminAuth())
	management.GET("/tools", controller.GetMCPTools)
	management.GET("/keys", controller.GetMCPKeys)
	management.POST("/keys", controller.CreateMCPKey)
	management.DELETE("/keys/:id", controller.RevokeMCPKey)
	management.GET("/logs", controller.GetMCPLogs)
}
