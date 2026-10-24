package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

func registerTicketRoutes(apiRouter *gin.RouterGroup) {
	personalAPI := controller.TicketAPI{}
	personal := apiRouter.Group("/tickets", middleware.UserAuth(), middleware.DisableCache())
	personal.GET("", personalAPI.List)
	personal.POST("", middleware.UserCriticalRateLimit("ticket-write"), personalAPI.Create)
	personal.GET("/:id", personalAPI.Detail)
	personal.POST("/:id/messages", middleware.UserCriticalRateLimit("ticket-write"), personalAPI.Reply)
	personal.PUT("/:id/status", middleware.UserCriticalRateLimit("ticket-write"), personalAPI.Status)

	adminAPI := controller.TicketAPI{Admin: true}
	admin := apiRouter.Group("/ticket-admin", middleware.AdminAuth(), middleware.DisableCache())
	admin.GET("", middleware.RequirePermission(authz.TicketAdminRead), adminAPI.List)
	admin.PUT("/batch/status", middleware.RequirePermission(authz.TicketAdminWrite), middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), adminAPI.BatchStatus)
	admin.POST("/batch/delete", middleware.RequirePermission(authz.TicketAdminWrite), middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), adminAPI.BatchDelete)
	admin.GET("/automation/auto-archive", middleware.RequirePermission(authz.TicketAdminRead), adminAPI.AutoArchive)
	admin.PUT("/automation/auto-archive", middleware.RequirePermission(authz.TicketAdminWrite), middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), adminAPI.UpdateAutoArchive)
	admin.GET("/automation/auto-reply", middleware.RequirePermission(authz.TicketAdminRead), adminAPI.AutoReply)
	admin.PUT("/automation/auto-reply", middleware.RequirePermission(authz.TicketAdminWrite), middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), adminAPI.UpdateAutoReply)
	admin.GET("/:id", middleware.RequirePermission(authz.TicketAdminRead), adminAPI.Detail)
	admin.POST("/:id/messages", middleware.RequirePermission(authz.TicketAdminWrite), middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), adminAPI.Reply)
	admin.PUT("/:id/status", middleware.RequirePermission(authz.TicketAdminWrite), middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), adminAPI.Status)
}
