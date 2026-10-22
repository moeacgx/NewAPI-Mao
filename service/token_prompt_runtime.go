package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

const tokenPromptAuditKey = "token_prompt_templates"
const tokenPromptManagedKey = "token_prompt_managed"

// ApplySelectedTokenPrompts 在每次实际选组后读取当前发布内容，不缓存撤销状态。
func ApplySelectedTokenPrompts(c *gin.Context, format types.RelayFormat, body []byte) ([]byte, error) {
	c.Set(tokenPromptAuditKey, nil)
	tokenID := common.GetContextKeyInt(c, constant.ContextKeyTokenId)
	if tokenID <= 0 || c.Request == nil || c.Request.URL == nil {
		return body, nil
	}
	path := c.Request.URL.Path
	supported := (format == types.RelayFormatOpenAI && strings.HasSuffix(path, "/chat/completions")) ||
		(format == types.RelayFormatOpenAIResponses && strings.HasSuffix(path, "/responses")) ||
		(format == types.RelayFormatClaude && strings.HasSuffix(path, "/messages")) ||
		(format == types.RelayFormatGemini && (strings.HasSuffix(path, ":generateContent") || strings.HasSuffix(path, ":streamGenerateContent")))
	if !supported {
		return body, nil
	}
	prompts, err := model.GetActiveTokenPrompts(tokenID, common.GetContextKeyString(c, constant.ContextKeyUsingGroup))
	if err != nil {
		return nil, err
	}
	if len(prompts) == 0 {
		return body, nil
	}
	texts := make([]string, 0, len(prompts))
	audit := make([]map[string]any, 0, len(prompts))
	for _, prompt := range prompts {
		texts = append(texts, prompt.Content)
		audit = append(audit, map[string]any{"id": prompt.ID, "title": prompt.Title, "version": prompt.Version})
	}
	modified, err := ApplyTokenPromptText(body, format, common.GetContextKeyString(c, constant.ContextKeyOriginalModel), texts)
	if err != nil {
		return nil, err
	}
	c.Set(tokenPromptAuditKey, audit)
	c.Set(tokenPromptManagedKey, true)
	return modified, nil
}

// TokenPromptManaged 表示本请求已装载过模板；换组不命中时仍需恢复计数与计费快照。
func TokenPromptManaged(c *gin.Context) bool { return c.GetBool(tokenPromptManagedKey) }

func AppendTokenPromptLog(c *gin.Context, other map[string]interface{}) {
	if c == nil || other == nil {
		return
	}
	if audit, ok := c.Get(tokenPromptAuditKey); ok && audit != nil {
		other["prompt_templates"] = audit
	}
}
