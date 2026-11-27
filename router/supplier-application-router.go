package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

func registerSupplierApplicationRoutes(apiRouter *gin.RouterGroup) {
	personalAPI := controller.SupplierApplicationAPI{}
	personal := apiRouter.Group("/supplier-applications", middleware.UserAuth(), middleware.DisableCache())
	personal.GET("", personalAPI.List)
	personal.POST("", middleware.UserCriticalRateLimit("supplier-application-write"), personalAPI.Create)
	personal.GET("/:id", personalAPI.Detail)
	personal.POST("/:id/messages", middleware.UserCriticalRateLimit("supplier-application-write"), personalAPI.CreateMessage)

	adminAPI := controller.SupplierApplicationAPI{Admin: true}
	admin := apiRouter.Group("/supplier-application-admin", middleware.AdminAuth(), middleware.DisableCache())
	admin.GET("", middleware.RequirePermission(authz.SupplierApplicationAdminRead), adminAPI.List)
	admin.GET("/:id", middleware.RequirePermission(authz.SupplierApplicationAdminRead), adminAPI.Detail)
	admin.PUT("/:id", middleware.RequirePermission(authz.SupplierApplicationAdminWrite), middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), adminAPI.Review)
	admin.POST("/:id/messages", middleware.RequirePermission(authz.SupplierApplicationAdminWrite), middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), adminAPI.CreateMessage)
}
