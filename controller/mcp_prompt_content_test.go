package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupMCPPromptContentDB(t *testing.T) *gorm.DB {
	t.Helper()
	mockDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := mockDB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	previousDB, previousLogDB, previousRedis := model.DB, model.LOG_DB, common.RedisEnabled
	model.DB, model.LOG_DB, common.RedisEnabled = mockDB, mockDB, false
	t.Cleanup(func() {
		model.DB, model.LOG_DB, common.RedisEnabled = previousDB, previousLogDB, previousRedis
		_ = sqlDB.Close()
	})
	require.NoError(t, mockDB.AutoMigrate(&model.Group{}, &model.PromptCategory{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.PromptSkillRecord{}, &model.PromptSkillFile{}, &model.PromptSkillChunk{}))
	return mockDB
}

func callMCPPromptContent(query string, internal bool) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/api/prompt-center/admin/"+query, nil)
	request.Header.Set("X-MCP-Key-ID", "1")
	if internal {
		request = request.WithContext(service.MCPInternalContext(request.Context(), 1))
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = request
	GetAdminPromptTemplates(c)
	return recorder
}

func TestMCPPromptContentPaginationAndExactID(t *testing.T) {
	mockDB := setupMCPPromptContentDB(t)
	group := model.Group{Code: "internal-prompt-group", Name: "提示词显示分组", Status: model.GroupStatusActive}
	require.NoError(t, mockDB.Create(&group).Error)
	var saved []*model.PromptTemplate
	for _, order := range []int{30, 10, 20} {
		item, err := model.SavePromptTemplate(model.PromptTemplate{
			Title: fmt.Sprintf("模板 %d", order), Description: "完整说明", Content: fmt.Sprintf("正文 %d", order),
			SortOrder: order, LoadMode: model.PromptLoadModeProgressive, GroupIDs: []int{group.Id},
			Skills: []model.PromptSkill{{Path: "skills/example/SKILL.md", Resources: []model.PromptSkillResource{{Path: "references/guide.md", Content: "引用正文"}}}},
		})
		require.NoError(t, err)
		saved = append(saved, item)
	}
	ordered := []int{saved[1].ID, saved[2].ID, saved[0].ID}
	for _, tc := range []struct {
		query string
		ids   []int
		page  int
		size  int
	}{
		{"", ordered, 1, 5},
		{"?p=2&page_size=1", ordered[1:2], 2, 1},
		{"?p=2&page_size=2", ordered[2:], 2, 2},
		{"?p=10000&page_size=100", []int{}, 10000, 100},
	} {
		t.Run("分页"+tc.query, func(t *testing.T) {
			recorder := callMCPPromptContent(tc.query, true)
			require.Equal(t, http.StatusOK, recorder.Code)
			var response struct {
				Success bool `json:"success"`
				Data    struct {
					Items    []model.PromptTemplate `json:"items"`
					Total    int                    `json:"total"`
					Page     int                    `json:"page"`
					PageSize int                    `json:"page_size"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			require.True(t, response.Success)
			assert.Equal(t, 3, response.Data.Total)
			assert.Equal(t, tc.page, response.Data.Page)
			assert.Equal(t, tc.size, response.Data.PageSize)
			ids := make([]int, 0, len(response.Data.Items))
			for _, item := range response.Data.Items {
				ids = append(ids, item.ID)
				require.Len(t, item.Skills, 1)
				require.Len(t, item.Skills[0].Resources, 1)
				assert.Equal(t, "引用正文", item.Skills[0].Resources[0].Content)
				require.Len(t, item.GroupDetails, 1)
				assert.Equal(t, "提示词显示分组", item.GroupDetails[0].Name)
				assert.Equal(t, "internal-prompt-group", item.GroupDetails[0].Code)
			}
			assert.Equal(t, tc.ids, ids)
		})
	}
	for _, internal := range []bool{false, true} {
		recorder := callMCPPromptContent("?id="+strconv.Itoa(saved[0].ID), internal)
		require.Equal(t, http.StatusOK, recorder.Code)
		if !internal {
			var response struct {
				Data []model.PromptTemplate `json:"data"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.Len(t, response.Data, 3, "普通后台和伪造请求头仍返回原数组")
			continue
		}
		var response struct {
			Data model.PromptTemplate `json:"data"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.Equal(t, *saved[0], response.Data)
		// 原摘要工具即使请求同一个精确 ID，也不能获得正文授权。
		out, success, err := service.ProjectMCPResponse(service.MCPTool{Name: "newapi_prompts_list", Category: "prompts"}, recorder.Body.Bytes(), recorder.Code)
		require.NoError(t, err)
		require.True(t, success)
		assert.NotContains(t, string(out), "正文")
		assert.NotContains(t, string(out), `"content"`)
		assert.NotContains(t, string(out), `"resources"`)
	}
	for _, query := range []string{"?id=", "?id=0", "?id=2147483648", "?id=1&id=2", "?id=1&p=1", "?p=0", "?p=10001", "?page_size=0", "?page_size=101", "?p=bad", "?page_size="} {
		recorder := callMCPPromptContent(query, true)
		assert.Equal(t, http.StatusBadRequest, recorder.Code, query)
	}
	assert.Equal(t, http.StatusNotFound, callMCPPromptContent("?id=2147483647", true).Code)
}

func TestMCPPromptContentLargeCatalogReadsPagesBeforeHydration(t *testing.T) {
	setupMCPPromptContentDB(t)
	body := strings.Repeat("正文", 120000)
	var saved []*model.PromptTemplate
	for _, title := range []string{"甲", "乙", "丙"} {
		item, err := model.SavePromptTemplate(model.PromptTemplate{Title: title, Content: body, LoadMode: model.PromptLoadModeProgressive})
		require.NoError(t, err)
		saved = append(saved, item)
	}
	whole := callMCPPromptContent("", false)
	require.Greater(t, whole.Body.Len(), 4*1024*1024, "真实渐进式目录超过单次 MCP 响应上限")
	for _, tool := range service.MCPTools() {
		if tool.Name != "newapi_prompts_content_list" && tool.Name != "newapi_prompt_content_get" {
			continue
		}
		for index, item := range saved {
			query := fmt.Sprintf("?p=%d&page_size=1", index+1)
			if tool.Name == "newapi_prompt_content_get" {
				query = "?id=" + strconv.Itoa(item.ID)
			}
			recorder := callMCPPromptContent(query, true)
			out, success, err := service.ProjectMCPResponse(tool, recorder.Body.Bytes(), recorder.Code)
			require.NoError(t, err)
			require.True(t, success)
			assert.Contains(t, string(out), body)
			assert.Less(t, len(out), 4*1024*1024)
		}
	}
	// 页外损坏的正文不能使当前页失败，验证先分页、后装配，而非全量读取再切片。
	require.NoError(t, model.DB.Model(&model.PromptTemplate{}).Where("id = ?", saved[2].ID).Update("load_mode", "invalid-mode").Error)
	recorder := callMCPPromptContent("?p=1&page_size=1", true)
	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.True(t, response.Success)
	recorder = callMCPPromptContent("?id="+strconv.Itoa(saved[2].ID), true)
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
}

func TestMCPFullAccountOAuthDiscoveryControllerProjection(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/.well-known/openid-configuration", r.URL.Path)
		assert.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"issuer":"https://issuer.example","authorization_endpoint":"https://issuer.example/authorize","token_endpoint":"https://issuer.example/token","userinfo_endpoint":"https://issuer.example/userinfo","jwks_uri":"https://issuer.example/jwks","client_secret":"private-fixture","unknown":{"key":"private-fixture"}}`))
		assert.NoError(t, err)
	}))
	defer upstream.Close()
	var tool service.MCPTool
	for _, candidate := range service.MCPTools() {
		if candidate.Name == "newapi_oauth_discovery" {
			tool = candidate
			break
		}
	}
	require.NotEmpty(t, tool.Name)
	arguments, err := common.Marshal(map[string]any{"body": map[string]string{"well_known_url": upstream.URL + "/.well-known/openid-configuration"}})
	require.NoError(t, err)
	_, path, body, err := service.PrepareMCPRequest(tool, arguments)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = request
	FetchCustomOAuthDiscovery(c)
	out, success, err := service.ProjectMCPResponse(tool, recorder.Body.Bytes(), recorder.Code)
	require.NoError(t, err)
	require.True(t, success)
	assert.Contains(t, string(out), `"token_endpoint":"https://issuer.example/token"`)
	assert.Contains(t, string(out), `"jwks_uri":"https://issuer.example/jwks"`)
	assert.NotContains(t, string(out), "private-fixture")
	assert.NotContains(t, string(out), "client_secret")
	assert.NotContains(t, string(out), "unknown")
}

func TestMCPFullAccountOAuthDiscoveryRootAndReadReceipt(t *testing.T) {
	mockDB := setupMCPPromptContentDB(t)
	require.NoError(t, mockDB.AutoMigrate(&model.User{}, &model.MCPKey{}, &model.MCPCallLog{}, &model.MCPWriteRequest{}))
	group := model.Group{Code: "discovery-fixture", Name: "发现配置测试分组", Status: model.GroupStatusActive}
	require.NoError(t, mockDB.Create(&group).Error)
	user := model.User{Username: "discovery-root", Role: common.RoleRootUser, Status: common.UserStatusEnabled, Group: group.Code}
	require.NoError(t, mockDB.Create(&user).Error)
	key := model.MCPKey{UserID: user.Id, KeyHash: "discovery-fixture-hash", Tools: []string{"newapi_oauth_discovery"}, ExpiresAt: time.Now().Unix() + 3600}
	require.NoError(t, mockDB.Create(&key).Error)
	identity := &service.MCPIdentity{User: user.ToBaseUser(), Key: &key}
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"issuer":"https://issuer.example","token_endpoint":"https://issuer.example/token","client_secret":"private-discovery-fixture"}`))
		assert.NoError(t, err)
	}))
	defer upstream.Close()
	engine := gin.New()
	// 隔离用例保留执行器的同步 MCP 审计；不启动通用后台审计的异步数据库写入。
	engine.Use(func(c *gin.Context) {
		c.Set(string(constant.ContextKeyAuditLogged), true)
		c.Next()
	})
	engine.POST("/api/custom-oauth-provider/discovery", middleware.RootAuth(), FetchCustomOAuthDiscovery)
	args, err := common.Marshal(map[string]any{"body": map[string]string{"well_known_url": upstream.URL}})
	require.NoError(t, err)
	for range 2 {
		result := service.ExecuteMCPTool(context.Background(), identity, "newapi_oauth_discovery", args, engine)
		require.False(t, result.IsError, "%+v", result)
		require.Len(t, result.Content, 1)
		assert.Contains(t, result.Content[0].Text, `"token_endpoint":"https://issuer.example/token"`)
		assert.NotContains(t, result.Content[0].Text, "private-discovery-fixture")
	}
	assert.EqualValues(t, 2, calls.Load(), "只读重试重新查询，不能复用写回执")
	var writes int64
	require.NoError(t, mockDB.Model(&model.MCPWriteRequest{}).Count(&writes).Error)
	assert.Zero(t, writes)
	var logs []model.MCPCallLog
	require.NoError(t, mockDB.Find(&logs).Error)
	require.Len(t, logs, 2)
	rawLogs, err := common.Marshal(logs)
	require.NoError(t, err)
	assert.NotContains(t, string(rawLogs), "private-discovery-fixture")
	assert.NotContains(t, string(rawLogs), upstream.URL)
	// 即使入口身份还是 Root 快照，执行时和真实路由都会重新检查当前角色。
	require.NoError(t, mockDB.Model(&user).Update("role", common.RoleAdminUser).Error)
	result := service.ExecuteMCPTool(context.Background(), identity, "newapi_oauth_discovery", args, engine)
	assert.True(t, result.IsError)
	body, err := common.Marshal(map[string]string{"well_known_url": upstream.URL})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/custom-oauth-provider/discovery", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(service.MCPInternalContext(request.Context(), key.ID))
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.EqualValues(t, 2, calls.Load(), "降权后不得再访问发现地址")
}
