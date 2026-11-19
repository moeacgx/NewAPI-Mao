package router

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 复用实际路由器与 MCP 传输夹具；数据库仅为本用例的内存 SQLite。
func setupMCPAccountRoutes(t *testing.T) mcpCoreFixture {
	t.Helper()
	f := setupMCPCore(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Token{}, &model.Group{}, &model.GroupAlias{}, &model.TokenGroupBinding{}, &model.PromptTemplate{}, &model.TokenPromptBinding{}))
	group := model.Group{Code: "default", Name: "令牌测试显示分组", Status: model.GroupStatusActive}
	require.NoError(t, model.DB.Create(&group).Error)
	require.NoError(t, model.DB.Model(f.user).Updates(map[string]any{"role": common.RoleRootUser, "group_id": group.Id}).Error)
	f.user.Role = common.RoleRootUser
	f.user.GroupId = group.Id
	key, secret, err := service.CreateMCPKey(f.user.ToBaseUser(), "账号管理回归", []string{
		"newapi_my_token_create", "newapi_my_token_edit_get", "newapi_my_token_update", "newapi_my_token_key",
		"newapi_user_edit_get", "newapi_user_permissions_get", "newapi_user_quota_adjust",
		"newapi_my_token_auto_groups",
	}, time.Now().Unix()+3600)
	require.NoError(t, err)
	f.key, f.secret = key, secret
	return f
}

func callMCPAccountRoute(t *testing.T, f mcpCoreFixture, name string, arguments map[string]any) []byte {
	t.Helper()
	request, err := common.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": arguments}})
	require.NoError(t, err)
	recorder := f.rpc(string(request), nil)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Result service.MCPToolResult `json:"result"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Result.IsError, recorder.Body.String())
	require.Len(t, response.Result.Content, 1)
	return []byte(response.Result.Content[0].Text)
}

func mcpTokenCreateFixtureBody() map[string]any {
	return map[string]any{
		"name": "允许重复名称", "expired_time": -1, "remain_quota": 100, "unlimited_quota": false,
		"model_limits_enabled": false, "model_limits": "", "allow_ips": "", "group": "",
		"group_mode": "inherit", "group_ids": []int{}, "group_ratio_limits": "",
		"cross_group_retry": false, "auto_groups": []string{}, "group_channel_ids": map[string]int{},
	}
}

func TestMCPAccountTokenCreateReturnsIDAndKeepsSecretSeparate(t *testing.T) {
	f := setupMCPAccountRoutes(t)
	body := mcpTokenCreateFixtureBody()
	var created []model.Token
	for _, requestID := range []string{"token-a", "token-b"} {
		out := callMCPAccountRoute(t, f, "newapi_my_token_create", map[string]any{"request_id": requestID, "body": body})
		var response struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		require.NoError(t, common.Unmarshal(out, &response))
		require.Positive(t, response.Data.ID)
		var token model.Token
		require.NoError(t, model.DB.First(&token, response.Data.ID).Error)
		assert.Equal(t, f.user.Id, token.UserId)
		assert.Equal(t, body["name"], token.Name)
		require.NotEmpty(t, token.Key)
		assert.NotContains(t, string(out), token.Key)
		assert.NotContains(t, string(out), `"key"`)
		created = append(created, token)
	}
	assert.NotEqual(t, created[0].Id, created[1].Id, "同名创建由实际新 ID 唯一定位")
	replay := callMCPAccountRoute(t, f, "newapi_my_token_create", map[string]any{"request_id": "token-a", "body": body})
	var receipt struct {
		Receipt struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		} `json:"receipt"`
	}
	require.NoError(t, common.Unmarshal(replay, &receipt))
	assert.Equal(t, created[0].Id, receipt.Receipt.Data.ID)
	var count int64
	require.NoError(t, model.DB.Model(&model.Token{}).Count(&count).Error)
	assert.EqualValues(t, 2, count, "幂等重试不得重复创建")
	for _, token := range created {
		out := callMCPAccountRoute(t, f, "newapi_my_token_key", map[string]any{"path": map[string]string{"id": strconv.Itoa(token.Id)}})
		var response struct {
			Data struct {
				Key string `json:"key"`
			} `json:"data"`
		}
		require.NoError(t, common.Unmarshal(out, &response))
		assert.Equal(t, token.Key, response.Data.Key, "按创建 ID 读取的是对应本人令牌密钥")
	}
	var writes []model.MCPWriteRequest
	require.NoError(t, model.DB.Find(&writes).Error)
	require.Len(t, writes, 2, "密钥只读调用不产生写回执")
	var logs []model.MCPCallLog
	require.NoError(t, model.DB.Find(&logs).Error)
	saved, err := common.Marshal(map[string]any{"writes": writes, "logs": logs})
	require.NoError(t, err)
	for _, token := range created {
		assert.NotContains(t, string(saved), token.Key)
	}
	// JWT 后台调用即使伪造 MCP 请求头，也保持原响应且不返回 ID 或密钥。
	rawBody, err := common.Marshal(body)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/token/", strings.NewReader(string(rawBody)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+f.session)
	request.Header.Set("X-MCP-Key-ID", strconv.Itoa(f.key.ID))
	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"success":true,"message":""}`, recorder.Body.String())
}

func TestMCPAccountTokenReadThenUpdateNormalizesEmptyCollections(t *testing.T) {
	f := setupMCPAccountRoutes(t)
	created := callMCPAccountRoute(t, f, "newapi_my_token_create", map[string]any{"request_id": "create-edit", "body": mcpTokenCreateFixtureBody()})
	var response struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(created, &response))
	id := response.Data.ID
	require.Positive(t, id)
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", id).Update("allow_ips", nil).Error)
	// 先确认真实控制器仍返回历史 null/省略契约，再经过工具投影归一化。
	request := httptest.NewRequest(http.MethodGet, "/api/token/"+strconv.Itoa(id), nil)
	request = request.WithContext(service.MCPInternalContext(request.Context(), f.key.ID))
	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, request)
	require.Contains(t, recorder.Body.String(), `"auto_groups":null`)
	require.Contains(t, recorder.Body.String(), `"allow_ips":null`)
	require.NotContains(t, recorder.Body.String(), `"group_ids"`)
	require.NotContains(t, recorder.Body.String(), `"group_channel_ids"`)
	out := callMCPAccountRoute(t, f, "newapi_my_token_edit_get", map[string]any{"path": map[string]string{"id": strconv.Itoa(id)}})
	var editable struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, common.Unmarshal(out, &editable))
	assert.JSONEq(t, `[]`, string(editable.Data["auto_groups"]))
	assert.JSONEq(t, `[]`, string(editable.Data["group_ids"]))
	assert.JSONEq(t, `{}`, string(editable.Data["group_channel_ids"]))
	assert.Equal(t, `""`, string(editable.Data["allow_ips"]))
	// 只选更新工具声明的编辑字段，读取附带的统计与展示字段不能当作输入。
	body := map[string]json.RawMessage{}
	for _, tool := range service.MCPTools() {
		if tool.Name != "newapi_my_token_update" {
			continue
		}
		schema := tool.InputSchema["properties"].(map[string]any)["body"].(map[string]any)
		for field := range schema["properties"].(map[string]any) {
			require.Contains(t, editable.Data, field)
			body[field] = editable.Data[field]
		}
	}
	body["name"] = json.RawMessage(`"读取后保存"`)
	callMCPAccountRoute(t, f, "newapi_my_token_update", map[string]any{"request_id": "save-edit", "body": body})
	var token model.Token
	require.NoError(t, model.DB.First(&token, id).Error)
	assert.Equal(t, "读取后保存", token.Name)
	assert.Equal(t, model.TokenGroupModeInherit, token.GroupMode)
	assert.Equal(t, 100, token.RemainQuota)
}

func TestMCPAccountUserReadsPreserveInt64Quota(t *testing.T) {
	f := setupMCPAccountRoutes(t)
	target := model.User{Username: "debt-user", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: -1}
	require.NoError(t, model.DB.Create(&target).Error)
	for _, quota := range []int64{-1, 2147483648, -2147483649, math.MaxInt64, math.MinInt64} {
		require.NoError(t, model.DB.Model(&target).Update("quota", quota).Error)
		for _, name := range []string{"newapi_user_edit_get", "newapi_user_permissions_get"} {
			t.Run(name+"/"+strconv.FormatInt(quota, 10), func(t *testing.T) {
				out := callMCPAccountRoute(t, f, name, map[string]any{"path": map[string]string{"id": strconv.Itoa(target.Id)}})
				var response struct {
					Data struct {
						ID    int             `json:"id"`
						Quota json.RawMessage `json:"quota"`
					} `json:"data"`
				}
				require.NoError(t, common.Unmarshal(out, &response))
				assert.Equal(t, target.Id, response.Data.ID)
				assert.Equal(t, strconv.FormatInt(quota, 10), string(response.Data.Quota), "实际 GetUser 返回的 BIGINT 必须保留完整原文")
				var actual int64
				require.NoError(t, common.Unmarshal(response.Data.Quota, &actual))
				assert.Equal(t, quota, actual)
			})
		}
	}
	for _, tool := range service.MCPTools() {
		if tool.Name != "newapi_user_quota_adjust" {
			continue
		}
		for _, value := range []int64{-1, 2147483648} {
			input, err := common.Marshal(map[string]any{"request_id": "invalid-adjust", "body": map[string]any{"id": target.Id, "action": "add_quota", "mode": "subtract", "value": value}})
			require.NoError(t, err)
			_, _, _, err = service.PrepareMCPRequest(tool, input)
			require.Error(t, err, "读取支持完整 int64 不应扩大调整输入范围")
		}
	}
}

func TestMCPAccountTokenAutoGroupsUsesCurrentUserScope(t *testing.T) {
	f := setupMCPAccountRoutes(t)
	previousAuto, previousUsable := setting.AutoGroups2JsonString(), setting.UserUsableGroups2JSONString()
	previousRatio, previousMax := ratio_setting.GroupRatio2JSONString(), setting.GetMaxTokenAutoGroups()
	special := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup
	previousSpecial, err := common.Marshal(special.ReadAll())
	require.NoError(t, err)
	special.Set("default", map[string]string{})
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(previousAuto))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(previousUsable))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(previousRatio))
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(strconv.Itoa(previousMax)))
		require.NoError(t, common.Unmarshal(previousSpecial, special))
	})
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`["private","default","default","no-ratio","auto"]`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"no-ratio":"没有倍率"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"private":1}`))
	for _, limit := range []int{0, 2} {
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(strconv.Itoa(limit)))
		out := callMCPAccountRoute(t, f, "newapi_my_token_auto_groups", map[string]any{})
		expected, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{
			"groups": []string{"default"}, "group_names": []string{"令牌测试显示分组"}, "max_count": limit,
		}})
		require.NoError(t, err)
		assert.JSONEq(t, string(expected), string(out), "保留本人可选范围、原有顺序与数量限制")
		assert.NotContains(t, string(out), "private")
		assert.NotContains(t, string(out), "no-ratio")
	}
	require.NoError(t, model.DB.Model(&model.Group{}).Where("code = ?", "default").Update("name", "改名后的分组").Error)
	out := callMCPAccountRoute(t, f, "newapi_my_token_auto_groups", map[string]any{})
	assert.Contains(t, string(out), "改名后的分组")
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`[]`))
	out = callMCPAccountRoute(t, f, "newapi_my_token_auto_groups", map[string]any{})
	assert.JSONEq(t, `{"success":true,"data":{"groups":[],"group_names":[],"max_count":2}}`, string(out))
}

func TestMCPAccountToolsListRequiredSchemaContract(t *testing.T) {
	f := setupMCPAccountRoutes(t)
	// 通过实际 MCP tools/list 获取序列化后的契约，不检查 Go 内存中的切片类型。
	names := make([]string, 0)
	for _, tool := range service.MCPTools() {
		switch tool.Category {
		case "users", "tokens", "tickets", "prompts", "oauth", "authz":
			names = append(names, tool.Name)
		}
	}
	key, secret, err := service.CreateMCPKey(f.user.ToBaseUser(), "账号 schema 发现回归", names, time.Now().Unix()+3600)
	require.NoError(t, err)
	f.key, f.secret = key, secret
	recorder := f.rpc(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, nil)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Result struct {
			Tools []struct {
				Name        string         `json:"name"`
				InputSchema map[string]any `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Result.Tools, len(names))
	for _, tool := range response.Result.Tools {
		t.Run(tool.Name, func(t *testing.T) {
			require.Equal(t, "object", tool.InputSchema["type"])
			assertMCPAccountRequiredSchema(t, tool.InputSchema)
		})
	}
}

// required 是 JSON Schema 的数组关键字：允许空数组或省略，不允许 null、重复或非字符串项。
func assertMCPAccountRequiredSchema(t *testing.T, schema map[string]any) {
	t.Helper()
	if value, exists := schema["required"]; exists {
		required, ok := value.([]any)
		require.True(t, ok, "required 必须序列化为数组，实际为 %#v", value)
		seen := map[string]bool{}
		for _, entry := range required {
			field, ok := entry.(string)
			require.True(t, ok)
			assert.False(t, seen[field], "required 字段不能重复")
			seen[field] = true
			properties, ok := schema["properties"].(map[string]any)
			require.True(t, ok)
			assert.Contains(t, properties, field, "固定对象 schema 的必需字段必须已声明")
		}
	}
	if properties, ok := schema["properties"].(map[string]any); ok {
		for _, value := range properties {
			child, ok := value.(map[string]any)
			require.True(t, ok)
			assertMCPAccountRequiredSchema(t, child)
		}
	}
	for _, keyword := range []string{"items", "additionalProperties"} {
		if child, ok := schema[keyword].(map[string]any); ok {
			assertMCPAccountRequiredSchema(t, child)
		}
	}
}
