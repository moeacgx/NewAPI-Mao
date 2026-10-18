package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

func registerTaskPluginManagement(api *gin.RouterGroup) {
	group := api.Group("/plugin/task", middleware.DisableCache(), middleware.GlobalAPIRateLimitWithAdminBypass(), middleware.AdminAuth())
	group.GET("", controller.ListTaskPlugins)
	group.GET("/runtime/status", controller.GetTaskPluginRuntime)
	// 市场源由 Admin 读取，Root 才能通过 PUT 更新；必须放在 /:key 之前避免被参数路由吞掉。
	group.GET("/marketplace/sources", controller.GetTaskPluginMarketplaceSources)
	group.GET("/:key", controller.GetTaskPlugin)
	group.GET("/:key/versions", controller.GetTaskPluginVersions)
	root := group.Group("", middleware.RootAuth(), middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()))
	root.PUT("/runtime/status", controller.SetTaskPluginRuntime)
	root.PUT("/marketplace/sources", controller.SetTaskPluginMarketplaceSources)
	root.POST("/:key/status", controller.SetTaskPluginStatus)
	root.POST("/:key/activate", controller.ActivateTaskPlugin)
	root.POST("", controller.UploadTaskPlugin)
	root.PUT("", controller.UploadTaskPlugin)
	root.DELETE("/:key/versions/:version", controller.DeleteTaskPluginVersion)
	root.POST("/:key/dryrun", controller.UnsupportedTaskPluginOperation)
	api.GET("/task_plugin_options", middleware.GlobalAPIRateLimitWithAdminBypass(), middleware.AdminAuth(), controller.GetTaskPluginOptions)
}
