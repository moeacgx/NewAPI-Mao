package service

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service/authz"
)

// 筛选项沿用真实分析路由的管理员身份和渠道读取权限，不开放任意查询。
func mcpFullAnalyticsTools() []MCPTool {
	tools := []MCPTool{
		{Name: "newapi_channel_analytics_filters", Category: "channels", Method: http.MethodGet, Path: "/api/channel-analytics/filters", MinRole: common.RoleAdminUser, Permission: authz.ChannelRead, TypedArguments: true, Description: "查询渠道分析筛选项及统计覆盖信息。分组 name 用于展示、code 用于筛选；模型 value/model_hash 为完整身份。模型较多时使用独立模型筛选分页工具。"},
		{Name: "newapi_channel_analytics_filter_models", Category: "channels", Method: http.MethodGet, Path: "/api/channel-analytics/filters/models", MinRole: common.RoleAdminUser, Permission: authz.ChannelRead, TypedArguments: true, Description: "分页搜索渠道分析模型筛选项；model_dimension 默认 requested，可选 upstream。q 去空格后最多 191 字节；page 默认 1，MCP 上限 10000；page_size 默认 50、最多 100。value/model_hash 是模型完整身份，label 用于展示。"},
	}
	for i := range tools {
		tool := &tools[i]
		tool.InputSchema = mcpToolInputSchema(*tool)
		queryFields := map[string]any{}
		if tool.Name == "newapi_channel_analytics_filter_models" {
			queryFields = map[string]any{
				"model_dimension": map[string]any{"type": "string", "enum": []string{"requested", "upstream"}},
				"q":               map[string]any{"type": "string", "maxLength": 191, "description": "宿主另检查去空格后的 UTF-8 长度不超过 191 字节。"},
				"page":            map[string]any{"type": "string", "pattern": "^([1-9][0-9]{0,3}|10000)$", "maxLength": 5},
				"page_size":       map[string]any{"type": "string", "pattern": "^([1-9][0-9]?|100)$", "maxLength": 3},
			}
		}
		tool.InputSchema["properties"].(map[string]any)["query"] = map[string]any{
			"type": "object", "properties": queryFields, "additionalProperties": false,
		}
	}
	return tools
}

// 使用真实响应 DTO 丢弃未知字段；原始回填错误可能包含数据库或凭据信息。
func projectMCPFullAnalyticsResponse(tool MCPTool, raw []byte) ([]byte, bool, error) {
	var target any
	switch tool.Name {
	case "newapi_channel_analytics_filters":
		target = &dto.ChannelAnalyticsFiltersResponse{}
	case "newapi_channel_analytics_filter_models":
		target = &dto.ChannelAnalyticsFilterModelsResponse{}
	default:
		return nil, false, nil
	}
	if len(raw) > mcpResponseLimit(tool) {
		return nil, true, errors.New("analytics response too large")
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := common.Unmarshal(raw, &envelope); err != nil {
		return nil, true, err
	}
	if common.GetJsonType(envelope.Data) != "object" {
		return nil, true, errors.New("invalid analytics response data")
	}
	if err := common.Unmarshal(envelope.Data, target); err != nil {
		return nil, true, err
	}
	if filters, ok := target.(*dto.ChannelAnalyticsFiltersResponse); ok && filters.Meta.Backfill != nil {
		filters.Meta.Backfill.LastError = ""
	}
	projected, err := common.Marshal(map[string]any{"success": true, "data": target})
	if len(projected) > mcpResponseLimit(tool) {
		return nil, true, errors.New("projected analytics response too large")
	}
	return projected, true, err
}
