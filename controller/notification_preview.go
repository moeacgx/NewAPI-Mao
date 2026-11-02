package controller

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// PreviewNotificationTemplate 只渲染样例，不读取工单正文、不发送 Telegram 消息。
func PreviewNotificationTemplate(c *gin.Context) {
	if !requireNotificationRoot(c) {
		return
	}
	var req struct {
		EventType string `json:"event_type"`
		Template  string `json:"template"`
		Target    struct {
			MentionUserId string `json:"mention_user_id"`
			MentionName   string `json:"mention_name"`
		} `json:"target"`
	}
	if !decodeNotificationRequest(c, &req, "无效的消息预览参数") {
		return
	}
	definition, exists := findNotificationEventDefinition(strings.TrimSpace(req.EventType))
	if !exists {
		common.ApiErrorMsg(c, "暂不支持该通知事件")
		return
	}
	if strings.TrimSpace(req.Template) == "" {
		common.ApiErrorMsg(c, "请输入消息模板")
		return
	}
	if utf8.RuneCountInString(req.Target.MentionName) > 128 {
		common.ApiErrorMsg(c, "提及名称最多允许 128 个字符")
		return
	}
	// 使用与保存相同的最坏样例预算，避免预览可用而实际正文无法发送。
	worstTarget := invoicePendingTemplateTargetSample
	limit := 4096
	if definition.Value == model.NotificationEventTypeTicketUserMessage {
		worstTarget.MentionName = strings.Repeat("\"", 128)
		limit -= utf8.RuneCountInString(model.NotificationTicketRepliedPrefix)
	}
	worst, err := service.RenderNotificationTemplate(req.Template, definition.SamplePayload, &worstTarget)
	if err != nil {
		common.ApiErrorMsg(c, fmt.Sprintf("消息模板无效: %v", err))
		return
	}
	if utf8.RuneCountInString(worst) > limit {
		common.ApiErrorMsg(c, "消息模板渲染后不能超过 Telegram 的 4096 字符限制")
		return
	}
	examples := map[string]any{
		"ticket_id": 13, "user_id": 6494, "title": "I can't pay for the USDT", "category": "billing", "priority": "normal",
		"status": "open", "actor_id": 6494, "actor_role": "user", "action": "created", "action_label": "新工单",
		"content":    "I tried different options, but the payment fails.\nPlease let me know when the problem is resolved.",
		"invoice_id": "INV-123", "source_type": "topup", "source_id": "order-123", "total_amount": "99.00",
		"create_time": "2026-11-02 12:00:00", "channel_name": "OpenAI primary", "channel_id": 123,
		"status_code": 403, "error_code": "bad_response_status_code", "error_message": "Insufficient account balance",
		"reason": "Insufficient account balance",
	}
	payload := make(map[string]any, len(definition.SamplePayload))
	for key, value := range definition.SamplePayload {
		if example, ok := examples[key]; ok {
			payload[key] = example
			continue
		}
		text := fmt.Sprint(value)
		if utf8.RuneCountInString(text) > 80 {
			text = string([]rune(text)[:80]) + "…"
		}
		payload[key] = text
	}
	target := model.NotificationTarget{MentionUserId: req.Target.MentionUserId, MentionName: req.Target.MentionName}
	rendered, err := service.RenderNotificationTemplate(req.Template, payload, &target)
	if err != nil {
		common.ApiErrorMsg(c, fmt.Sprintf("消息模板无效: %v", err))
		return
	}
	common.ApiSuccess(c, gin.H{"html": rendered, "text_length": utf8.RuneCountInString(rendered)})
}
