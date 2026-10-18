package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
)

// registerFeatureAPIRoutes keeps feature-owned contracts out of the legacy API
// router while preserving the middleware order at each authorization boundary.
func registerFeatureAPIRoutes(apiRouter *gin.RouterGroup) {
	registerExtensionRoutes(apiRouter)
	registerGameRoutes(apiRouter)
	registerNotificationRoutes(apiRouter)
	registerChannelAnalyticsRoutes(apiRouter)
}

func registerExtensionRoutes(apiRouter *gin.RouterGroup) {
	extensionRoute := apiRouter.Group("/extensions")
	extensionRoute.Use(middleware.UserSessionAuth(), middleware.DisableCache())
	{
		extensionRoute.GET("/", middleware.IssueExtensionSessionCookie(), controller.ListExtensions)
		extensionRoute.GET("/host/me", controller.GetExtensionHostContext)
		extensionRoute.GET("/:id/native/:pageKey/:target/:asset", controller.GetExtensionNativeAsset)
		extensionRoute.Any("/:id/proxy/*path", controller.ProxyExtension)
	}

	apiRouter.POST(
		"/extensions/:id/notification-events",
		middleware.RootAuth(),
		middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()),
		middleware.DisableCache(),
		controller.PublishExtensionNotificationEvent,
	)

	extensionAdminRoute := apiRouter.Group("/extension-admin")
	extensionAdminRoute.Use(middleware.RootAuth())
	{
		extensionAdminRoute.GET("/", controller.ListExtensions)
		extensionAdminRoute.GET("/marketplace", middleware.DisableCache(), controller.GetExtensionMarketplace)
		extensionAdminRoute.POST("/refresh", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), controller.RefreshExtensions)
		extensionAdminRoute.POST("/upload", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), controller.UploadExtension)
		extensionAdminRoute.PUT("/:id/enabled", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), controller.SetExtensionEnabled)
		extensionAdminRoute.DELETE("/:id", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), controller.UninstallExtension)
		extensionAdminRoute.GET("/okx-alipay-rate/config", controller.GetOkxAlipayRateConfig)
		extensionAdminRoute.PUT("/okx-alipay-rate/config", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), controller.SaveOkxAlipayRateConfig)
		extensionAdminRoute.GET("/okx-alipay-rate/quote", controller.PreviewOkxAlipayRate)
	}
}

func registerGameRoutes(apiRouter *gin.RouterGroup) {
	mutationBodyLimit := middleware.AnonymousRequestBodyLimit()

	gameRoute := apiRouter.Group("/game")
	gameRoute.Use(middleware.UserAuth())
	{
		gameRoute.GET("/wallet", controller.GetGameWallet)
		gameRoute.GET("/transactions", controller.GetGameWalletTransactions)
		gameRoute.GET("/predictions", controller.ListGamePredictions)
		gameRoute.GET("/predictions/:id", controller.GetGamePrediction)
		gameRoute.POST("/exchange/quota-to-token", middleware.UserCriticalRateLimit("game-exchange"), mutationBodyLimit, controller.ExchangeQuotaToGameTokens)
		gameRoute.POST("/exchange/token-to-quota", middleware.UserCriticalRateLimit("game-exchange"), mutationBodyLimit, controller.ExchangeGameTokensToQuota)
		gameRoute.POST("/predictions/:id/bets", middleware.UserCriticalRateLimit("game-bet"), mutationBodyLimit, controller.PlaceGamePredictionBet)
	}

	gameAdminRoute := apiRouter.Group("/game/admin")
	gameAdminRoute.Use(middleware.AdminAuth())
	{
		gameAdminRoute.GET("/predictions", middleware.RequirePermission(authz.GameAdminRead), controller.AdminListGamePredictions)
		gameAdminRoute.POST("/predictions", middleware.RequirePermission(authz.GameAdminWrite), middleware.AdminRateLimitBypass(middleware.UserCriticalRateLimit("game-admin-create")), mutationBodyLimit, controller.AdminCreateGamePrediction)
		gameAdminRoute.PUT("/predictions/:id/answer", middleware.RequirePermission(authz.GameAdminWrite), middleware.AdminRateLimitBypass(middleware.UserCriticalRateLimit("game-admin-answer")), mutationBodyLimit, controller.AdminSetGamePredictionAnswer)
		gameAdminRoute.POST("/predictions/:id/settle", middleware.RequirePermission(authz.GameAdminWrite), middleware.AdminRateLimitBypass(middleware.UserCriticalRateLimit("game-admin-settle")), controller.AdminSettleGamePrediction)
	}
}

func registerNotificationRoutes(apiRouter *gin.RouterGroup) {
	mutationBodyLimit := middleware.AnonymousRequestBodyLimit()
	notificationRoute := apiRouter.Group("/notification")
	notificationRoute.Use(middleware.RootAuth())
	{
		notificationRoute.GET("/event-types", controller.ListNotificationEventTypes)
		notificationRoute.GET("/bots", controller.ListNotificationBots)
		notificationRoute.GET("/tasks", controller.ListNotificationTasks)
		notificationRoute.GET("/deliveries", controller.ListNotificationDeliveries)

		notificationRoute.POST("/bots", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), mutationBodyLimit, controller.CreateNotificationBot)
		notificationRoute.PUT("/bots/:id", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), mutationBodyLimit, controller.UpdateNotificationBot)
		notificationRoute.DELETE("/bots/:id", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), controller.DisableNotificationBot)
		notificationRoute.POST("/bots/:id/test", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), mutationBodyLimit, controller.TestNotificationBot)
		notificationRoute.POST("/tasks", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), mutationBodyLimit, controller.CreateNotificationTask)
		notificationRoute.PUT("/tasks/:id", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), mutationBodyLimit, controller.UpdateNotificationTask)
		notificationRoute.DELETE("/tasks/:id", middleware.AdminRateLimitBypass(middleware.CriticalRateLimit()), controller.DisableNotificationTask)
	}
}

func registerChannelAnalyticsRoutes(apiRouter *gin.RouterGroup) {
	analyticsRoute := apiRouter.Group("/channel-analytics")
	analyticsRoute.Use(
		middleware.AdminAuth(),
		middleware.RequirePermission(controller.ChannelAnalyticsRequiredPermission()),
	)
	{
		analyticsRoute.GET("/summary", controller.GetChannelAnalyticsSummary)
		analyticsRoute.GET("/trend", controller.GetChannelAnalyticsTrend)
		analyticsRoute.GET("/channels", controller.GetChannelAnalyticsChannels)
		analyticsRoute.GET("/channels/:id/models", controller.GetChannelAnalyticsModels)
		analyticsRoute.GET("/stability", controller.GetChannelAnalyticsStability)
		analyticsRoute.GET("/status-codes", controller.GetChannelAnalyticsStatusCodes)
		analyticsRoute.GET("/failures", controller.GetChannelAnalyticsFailures)
		analyticsRoute.GET("/filters", controller.GetChannelAnalyticsFilters)
		analyticsRoute.GET("/filters/models", controller.GetChannelAnalyticsFilterModels)
	}
}
