package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func registerTaskPluginManagement(api *gin.RouterGroup) {
	group := api.Group("/plugin/task", middleware.DisableCache(), middleware.GlobalAPIRateLimit(), middleware.AdminAuth())
	group.GET("", controller.ListTaskPlugins)
	group.GET("/runtime/status", controller.GetTaskPluginRuntime)
	group.GET("/:key", controller.GetTaskPlugin)
	group.GET("/:key/versions", controller.GetTaskPluginVersions)
	root := group.Group("", middleware.RootAuth(), middleware.CriticalRateLimit())
	root.PUT("/runtime/status", controller.SetTaskPluginRuntime)
	root.POST("/:key/status", controller.SetTaskPluginStatus)
	root.POST("/:key/activate", controller.ActivateTaskPlugin)
	root.POST("", controller.UnsupportedTaskPluginOperation)
	root.PUT("", controller.UnsupportedTaskPluginOperation)
	root.DELETE("/:key/versions/:version", controller.UnsupportedTaskPluginOperation)
	root.POST("/:key/dryrun", controller.UnsupportedTaskPluginOperation)
	api.GET("/task_plugin_options", middleware.AdminAuth(), controller.GetTaskPluginOptions)
}
