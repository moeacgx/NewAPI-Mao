package service

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPAnalyticsFilterRequestContracts(t *testing.T) {
	tools := mcpFullAnalyticsTools()
	for _, tc := range []struct {
		name, input string
		index       int
		valid       bool
	}{
		{"基础筛选", `{}`, 0, true},
		{"模型默认值", `{}`, 1, true},
		{"上限分页与转义", `{"query":{"model_dimension":"upstream","q":"模型/a+b","page":"10000","page_size":"100"}}`, 1, true},
		{"基础接口拒绝过滤字段", `{"query":{"q":"x"}}`, 0, false},
		{"拒绝任意请求体", `{"body":{"url":"https://example.invalid"}}`, 1, false},
		{"拒绝未知参数", `{"query":{"channel_id":"1"}}`, 1, false},
		{"拒绝错误维度", `{"query":{"model_dimension":"both"}}`, 1, false},
		{"拒绝零页码", `{"query":{"page":"0"}}`, 1, false},
		{"拒绝超核心页码", `{"query":{"page":"10001"}}`, 1, false},
		{"拒绝零页大小", `{"query":{"page_size":"0"}}`, 1, false},
		{"拒绝超页大小", `{"query":{"page_size":"101"}}`, 1, false},
		{"拒绝数值参数类型", `{"query":{"page":1}}`, 1, false},
		{"拒绝多行查询", `{"query":{"q":"x\ny"}}`, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tool := tools[tc.index]
			_, path, body, err := PrepareMCPRequest(tool, []byte(tc.input))
			if !tc.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Empty(t, body)
			assert.False(t, tool.Write)
			address, err := url.Parse(path)
			require.NoError(t, err)
			assert.Equal(t, tool.Path, address.Path)
			if tc.index == 1 {
				query, err := ParseChannelAnalyticsFilterModelsQuery(address.Query())
				require.NoError(t, err)
				if tc.input == `{}` {
					assert.Equal(t, "requested", query.ModelDimension)
					assert.Equal(t, 1, query.Page)
					assert.Equal(t, 50, query.PageSize)
				} else {
					assert.Equal(t, "模型/a+b", query.Query)
					assert.Equal(t, 10000, query.Page)
					assert.Equal(t, 100, query.PageSize)
				}
			}
		})
	}
	// JSON Schema 长度按字符，宿主另限制 UTF-8 字节，不能放宽真实解析器。
	_, err := ParseChannelAnalyticsFilterModelsQuery(url.Values{"q": {strings.Repeat("中", 64)}})
	require.Error(t, err)
	_, err = ParseChannelAnalyticsFilterModelsQuery(url.Values{"q": {strings.Repeat("x", 191)}})
	require.NoError(t, err)
}

func TestMCPAnalyticsFilterProjectionPreservesDTOAndDropsSecrets(t *testing.T) {
	filters := dto.ChannelAnalyticsFiltersResponse{
		Channels:              []dto.ChannelAnalyticsFilterChannel{{ChannelId: 7, ChannelName: "渠道七", ChannelType: 1}},
		ChannelTypes:          []dto.ChannelAnalyticsFilterType{{Value: 1, Label: "OpenAI"}},
		Groups:                []dto.ChannelAnalyticsFilterGroup{{Code: "stable-code", Name: "当前分组显示名"}},
		RequestedModels:       []string{"model-a"},
		UpstreamModels:        []string{"upstream-a"},
		RequestedModelOptions: []dto.ChannelAnalyticsFilterModel{{Value: "full-hash-a", Label: "model-a", Model: "model-a", ModelHash: "full-hash-a"}},
		UpstreamModelOptions:  []dto.ChannelAnalyticsFilterModel{{Value: "full-hash-b", Label: "upstream-a", Model: "upstream-a", ModelHash: "full-hash-b"}},
		Outcomes:              []string{"success"},
		TrafficSources:        []string{"relay"},
		DataOrigins:           []string{"live"},
		Meta: dto.ChannelAnalyticsMeta{
			Partial: true, DimensionOverflowCount: 2, BucketLevel: "5m", RetentionDays: 7,
			Backfill: &dto.ChannelAnalyticsBackfillMeta{Status: "failed", LastError: "private-database-error", ScannedRows: 3},
		},
	}
	models := dto.ChannelAnalyticsFilterModelsResponse{
		ModelDimension: "upstream", Query: "模型",
		Items: []dto.ChannelAnalyticsFilterModel{{Value: "full-hash-b", Label: "模型", Model: "模型", ModelHash: "full-hash-b"}},
		Total: 1200, Page: 2, PageSize: 100,
	}
	for i, expected := range []any{filters, models} {
		t.Run(mcpFullAnalyticsTools()[i].Name, func(t *testing.T) {
			raw, err := common.Marshal(map[string]any{"success": true, "data": expected})
			require.NoError(t, err)
			var injected map[string]any
			require.NoError(t, common.Unmarshal(raw, &injected))
			data := injected["data"].(map[string]any)
			data["private_key"] = "private-key-value"
			if i == 0 {
				data["channels"].([]any)[0].(map[string]any)["key"] = "private-channel-key"
				filters.Meta.Backfill.LastError = ""
				expected = filters
			} else {
				data["items"].([]any)[0].(map[string]any)["payload"] = "private-payload"
			}
			raw, err = common.Marshal(injected)
			require.NoError(t, err)
			projected, success, err := ProjectMCPResponse(mcpFullAnalyticsTools()[i], raw, http.StatusOK)
			require.NoError(t, err)
			require.True(t, success)
			want, err := common.Marshal(map[string]any{"success": true, "data": expected})
			require.NoError(t, err)
			assert.JSONEq(t, string(want), string(projected))
			assert.NotContains(t, string(projected), "private-")
		})
	}
}

func TestMCPAnalyticsFilterResponseBoundAndFailure(t *testing.T) {
	tool := mcpFullAnalyticsTools()[1]
	raw, err := common.Marshal(map[string]any{"success": true, "data": dto.ChannelAnalyticsFilterModelsResponse{
		Items: []dto.ChannelAnalyticsFilterModel{{Label: strings.Repeat("模型", 7000), ModelHash: "complete-hash"}},
		Total: 1, Page: 1, PageSize: 50,
	}})
	require.NoError(t, err)
	require.Greater(t, len(raw), 32*1024)
	projected, success, err := ProjectMCPResponse(tool, raw, http.StatusOK)
	require.NoError(t, err)
	require.True(t, success)
	assert.JSONEq(t, string(raw), string(projected), "不能套旧工具上限或静默截断")
	_, _, err = ProjectMCPResponse(tool, []byte(strings.Repeat(" ", 4*1024*1024+1)), http.StatusOK)
	require.Error(t, err)
	_, _, err = ProjectMCPResponse(tool, []byte(`{"success":true,"data":{"items":"wrong-type"}}`), http.StatusOK)
	require.Error(t, err)
	projected, success, err = ProjectMCPResponse(tool, []byte(`{"success":false,"message":"private-provider-error"}`), http.StatusServiceUnavailable)
	require.NoError(t, err)
	assert.False(t, success)
	assert.NotContains(t, string(projected), "private-provider-error")
	_, handled, err := projectMCPFullAnalyticsResponse(MCPTool{Name: "newapi_channel_analytics_summary"}, []byte(`{}`))
	require.NoError(t, err)
	assert.False(t, handled, "旧分析工具不经新投影升级授权")
}

func TestMCPAnalyticsFilterCatalogAndPermission(t *testing.T) {
	db := mcpCommerceMockDB(t, &model.CasbinRule{})
	key := &model.MCPKey{Tools: []string{"newapi_channel_analytics_filters", "newapi_channel_analytics_filter_models"}}
	user := &model.UserBase{Id: 8, Role: common.RoleAdminUser}
	assert.Empty(t, MCPVisibleTools(user, key), "管理员也必须具有 ChannelRead")
	require.NoError(t, db.Create(&model.CasbinRule{Ptype: "p", V0: authz.UserSubject(8), V1: authz.ChannelRead.Resource, V2: authz.ChannelRead.Action, V3: "allow"}).Error)
	visible := MCPVisibleTools(user, key)
	require.Len(t, visible, 2, "必须接入共享目录")
	for _, tool := range visible {
		assert.Equal(t, authz.ChannelRead, tool.Permission)
		assert.Equal(t, common.RoleAdminUser, tool.MinRole)
		assert.True(t, tool.TypedArguments)
		assert.True(t, tool.Annotations["readOnlyHint"])
	}
	user.Role = common.RoleCommonUser
	assert.Empty(t, MCPVisibleTools(user, key))
	user.Role = common.RoleRootUser
	assert.Len(t, MCPVisibleTools(user, key), 2)
	assert.Empty(t, MCPVisibleTools(user, &model.MCPKey{Tools: []string{}}), "Root 仍必须显式授予工具")
}
