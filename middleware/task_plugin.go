package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// TaskPluginRequest 只标记显式插件入口，模型、令牌分组和渠道选择仍由本地中间件处理。
func TaskPluginRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("plugin_key")
		if !jsplugin.ValidPluginKey(key) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
			c.AbortWithStatus(http.StatusUnsupportedMediaType)
			return
		}
		loaded, pin, err := service.ActiveTaskPlugin(key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "官方任务插件未启用或不可用"})
			return
		}
		var body map[string]any
		if err := common.UnmarshalBodyReusable(c, &body); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		modelName, _ := body["model"].(string)
		if strings.TrimSpace(modelName) == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.Set("official_task_plugin", loaded)
		c.Set("task_plugin_snapshot", pin)
		c.Set("platform", key)
		c.Set("relay_mode", relayconstant.RelayModeVideoSubmit)
		c.Next()
	}
}
