package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PlaygroundTokenAuth 在登录鉴权之后，将本人选中的 API 令牌交给正式令牌鉴权链路。
// 完整密钥只在服务端使用，空头或失效选择不得降级为账号模式。
func PlaygroundTokenAuth() gin.HandlerFunc {
	tokenAuth := TokenAuth()
	return func(c *gin.Context) {
		values, selected := c.Request.Header[http.CanonicalHeaderKey("X-Playground-Token-Id")]
		if !selected {
			c.Next()
			return
		}
		if c.GetInt("id") <= 0 || c.GetBool("use_access_token") {
			abortWithOpenAiMessage(c, http.StatusUnauthorized, common.TranslateMessage(c, i18n.MsgAuthAccessTokenInvalid))
			return
		}
		if len(values) != 1 {
			abortWithOpenAiMessage(c, http.StatusBadRequest, common.TranslateMessage(c, i18n.MsgInvalidParams))
			return
		}
		id, err := strconv.Atoi(values[0])
		if err != nil || id <= 0 {
			abortWithOpenAiMessage(c, http.StatusBadRequest, common.TranslateMessage(c, i18n.MsgInvalidParams))
			return
		}
		token, err := model.GetTokenByIds(id, c.GetInt("id"))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				abortWithOpenAiMessage(c, http.StatusNotFound, common.TranslateMessage(c, i18n.MsgTokenInvalid))
			} else {
				common.SysLog("PlaygroundTokenAuth token lookup failed: " + err.Error())
				abortWithOpenAiMessage(c, http.StatusInternalServerError, common.TranslateMessage(c, i18n.MsgDatabaseError))
			}
			return
		}
		// 不允许备用 WebSocket 鉴权头覆盖已核验归属的令牌；保留原请求头供外层收尾使用。
		originalHeaders := c.Request.Header
		c.Request.Header = originalHeaders.Clone()
		defer func() { c.Request.Header = originalHeaders }()
		c.Request.Header.Del("Sec-WebSocket-Protocol")
		c.Request.Header.Set("Authorization", "Bearer sk-"+token.Key)
		tokenAuth(c)
	}
}
