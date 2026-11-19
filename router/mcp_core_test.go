package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var mcpTestKeyID atomic.Int64

type mcpCoreFixture struct {
	engine  *gin.Engine
	key     *model.MCPKey
	user    *model.User
	secret  string
	session string
}

func TestMCPCoreWriteAuditRecordsMCPIdentity(t *testing.T) {
	f := setupMCPCore(t)
	written := make(chan int, 1)
	require.NoError(t, model.DB.Callback().Create().After("gorm:create").Register("observe_mcp_operation_audit", func(tx *gorm.DB) {
		if entry, ok := tx.Statement.Dest.(*model.Log); ok && entry.Type == model.LogTypeManage && tx.Error == nil {
			select {
			case written <- entry.Id:
			default:
			}
		}
	}))
	r := f.rpc(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"newapi_vendor_create","arguments":{"body":{"name":"审计供应商"},"request_id":"audit-mcp"}}}`, nil)
	require.Contains(t, r.Body.String(), `"isError":false`, r.Body.String())
	var logID int
	select {
	case logID = <-written:
	case <-time.After(5 * time.Second):
		t.Fatal("管理路由未写入操作审计")
	}
	var entry model.Log
	require.NoError(t, model.DB.First(&entry, logID).Error)
	var metadata struct {
		AdminInfo struct {
			AuthMethod string `json:"auth_method"`
		} `json:"admin_info"`
	}
	require.NoError(t, common.UnmarshalJsonStr(entry.Other, &metadata))
	assert.Equal(t, "mcp", metadata.AdminInfo.AuthMethod)
	assert.NotContains(t, entry.Other, f.secret)
	var callLog model.MCPCallLog
	require.NoError(t, model.DB.Where("key_id = ? AND request_id = ?", f.key.ID, "audit-mcp").First(&callLog).Error)
	assert.Equal(t, f.key.ID, callLog.KeyID)
	assert.True(t, callLog.Success)
}

func setupMCPCore(t *testing.T) mcpCoreFixture {
	t.Helper()
	db := setupFeatureRouterAuthTest(t)
	require.NoError(t, db.AutoMigrate(&model.MCPKey{}, &model.MCPCallLog{}, &model.MCPWriteRequest{}, &model.CasbinRule{}, &model.AuthzRole{}, &model.Channel{}, &model.Log{}, &model.Vendor{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	oldGlobal, oldCritical, oldLog, oldMaster := common.GlobalApiRateLimitEnable, common.CriticalRateLimitEnable, model.LOG_DB, common.IsMasterNode
	common.GlobalApiRateLimitEnable, common.CriticalRateLimitEnable, common.IsMasterNode = false, false, true
	model.LOG_DB = db
	t.Cleanup(func() {
		common.GlobalApiRateLimitEnable, common.CriticalRateLimitEnable, model.LOG_DB, common.IsMasterNode = oldGlobal, oldCritical, oldLog, oldMaster
	})
	require.NoError(t, authz.Init(db))
	user, session := issueFeatureRouterSession(t, common.RoleAdminUser, "mcp")
	key, secret, err := service.CreateMCPKey(user.ToBaseUser(), "MCP 测试", []string{"newapi_context", "newapi_channel_status_set", "newapi_channel_update", "newapi_channel_get", "newapi_channel_models", "newapi_vendor_create", "newapi_vendor_get", "newapi_vendor_update"}, time.Now().Unix()+3600)
	require.NoError(t, err)
	newID := int(mcpTestKeyID.Add(1) + 1000)
	require.NoError(t, db.Model(key).Update("id", newID).Error)
	key.ID = newID
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)
	return mcpCoreFixture{engine: engine, key: key, user: user, secret: secret, session: session}
}

func TestMCPCoreVendorReadModifyWritePreservesMetadata(t *testing.T) {
	f := setupMCPCore(t)
	vendor := model.Vendor{Name: "旧名称", Description: "公开说明\n第二行", Icon: "OpenAI", Status: 1}
	require.NoError(t, vendor.Insert())
	call := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"newapi_vendor_get","arguments":{"path":{"id":"%d"}}}}`, vendor.Id)
	r := f.rpc(call, nil)
	var rpc struct {
		Result service.MCPToolResult `json:"result"`
	}
	require.NoError(t, common.Unmarshal(r.Body.Bytes(), &rpc))
	require.False(t, rpc.Result.IsError, r.Body.String())
	var projected struct {
		Data model.Vendor `json:"data"`
	}
	require.NoError(t, common.UnmarshalJsonStr(rpc.Result.Content[0].Text, &projected))
	assert.Equal(t, vendor.Description, projected.Data.Description)
	assert.Equal(t, vendor.Icon, projected.Data.Icon)
	projected.Data.Name = "新名称"
	body, err := common.Marshal(map[string]any{
		"id": projected.Data.Id, "name": projected.Data.Name, "description": projected.Data.Description,
		"icon": projected.Data.Icon, "status": projected.Data.Status,
	})
	require.NoError(t, err)
	call = fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"newapi_vendor_update","arguments":{"body":%s,"request_id":"metadata-update"}}}`, body)
	r = f.rpc(call, nil)
	require.Contains(t, r.Body.String(), `"isError":false`, r.Body.String())
	saved, err := model.GetVendorByID(vendor.Id)
	require.NoError(t, err)
	assert.Equal(t, "新名称", saved.Name)
	assert.Equal(t, vendor.Description, saved.Description)
	assert.Equal(t, vendor.Icon, saved.Icon)
}

func TestMCPCoreWriteReplayAndUncertainState(t *testing.T) {
	f := setupMCPCore(t)
	invalid := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"newapi_vendor_create","arguments":{"body":{"name":"仅创建一次","description":"公开说明","key":"must-not-persist-secret"},"request_id":"create-once"}}}`
	rejected := f.rpc(invalid, nil)
	require.Contains(t, rejected.Body.String(), `"isError":true`)
	assert.NotContains(t, rejected.Body.String(), "must-not-persist-secret")
	var count int64
	require.NoError(t, model.DB.Model(&model.Vendor{}).Count(&count).Error)
	require.Zero(t, count, "未通过参数校验时不得写入业务数据")
	require.NoError(t, model.DB.Model(&model.MCPWriteRequest{}).Count(&count).Error)
	require.Zero(t, count, "未执行的错误参数不得占用重试标识")
	call := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"newapi_vendor_create","arguments":{"body":{"name":"仅创建一次","description":"公开说明"},"request_id":"create-once"}}}`
	first := f.rpc(call, nil)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	require.Contains(t, first.Body.String(), `"isError":false`, first.Body.String())
	assert.NotContains(t, first.Body.String(), "must-not-persist-secret")
	replayed := f.rpc(call, nil)
	assert.JSONEq(t, first.Body.String(), replayed.Body.String())
	changed := f.rpc(strings.Replace(call, "仅创建一次", "不能第二次", 1), nil)
	assert.Contains(t, changed.Body.String(), "different arguments")
	require.NoError(t, model.DB.Model(&model.Vendor{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
	var stored model.MCPWriteRequest
	require.NoError(t, model.DB.Where("key_id = ? AND request_id = ?", f.key.ID, "create-once").First(&stored).Error)
	assert.Equal(t, "done", stored.State)
	assert.NotContains(t, stored.Result, "must-not-persist-secret")
	require.NoError(t, model.DB.Model(&stored).Update("state", "pending").Error)
	assert.Contains(t, f.rpc(call, nil).Body.String(), "pending or uncertain")
	require.NoError(t, model.DB.Model(&model.Vendor{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestMCPCoreManagementContractAndIsolation(t *testing.T) {
	f := setupMCPCore(t)
	other, otherSession := issueFeatureRouterSession(t, common.RoleAdminUser, "other")
	requestBody := fmt.Sprintf(`{"name":"一次性展示","tools":["newapi_context"],"expires_at":%d}`, time.Now().Unix()+3600)
	response := managementRequest(f.engine, http.MethodPost, "/api/mcp/keys", f.session, "192.0.2.10", requestBody)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var created struct {
		Success bool `json:"success"`
		Data    struct {
			Key    model.MCPKey `json:"key"`
			Secret string       `json:"secret"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &created))
	require.True(t, created.Success)
	assert.Len(t, created.Data.Secret, 47)
	assert.Contains(t, response.Header().Get("Cache-Control"), "no-store")
	var saved model.MCPKey
	require.NoError(t, model.DB.First(&saved, created.Data.Key.ID).Error)
	assert.Equal(t, service.MCPKeyDigest(created.Data.Secret), saved.KeyHash)
	assert.NotContains(t, response.Body.String(), saved.KeyHash)
	response = managementRequest(f.engine, http.MethodGet, "/api/mcp/keys?p=1&page_size=20", otherSession, "192.0.2.10", "")
	assert.Contains(t, response.Body.String(), `"total":0`)
	assert.NotContains(t, response.Body.String(), created.Data.Secret)
	response = managementRequest(f.engine, http.MethodDelete, fmt.Sprintf("/api/mcp/keys/%d", saved.ID), otherSession, "192.0.2.10", "")
	assert.Equal(t, http.StatusNotFound, response.Code)
	response = managementRequest(f.engine, http.MethodGet, "/api/mcp/keys?page_size=101", f.session, "192.0.2.10", "")
	assert.Equal(t, http.StatusBadRequest, response.Code)
	require.NoError(t, model.DB.Create(&model.MCPCallLog{UserID: f.user.Id, KeyID: saved.ID, ToolName: "newapi_context"}).Error)
	response = managementRequest(f.engine, http.MethodGet, "/api/mcp/logs", otherSession, "192.0.2.10", "")
	assert.Contains(t, response.Body.String(), `"total":0`)
	assert.NotEqual(t, f.user.Id, other.Id)
	response = managementRequest(f.engine, http.MethodDelete, fmt.Sprintf("/api/mcp/keys/%d", saved.ID), f.session, "192.0.2.10", "")
	assert.Equal(t, http.StatusOK, response.Code)
	_, err := service.AuthenticateMCP(created.Data.Secret)
	assert.Error(t, err)
}

func TestMCPCoreManagementRejectsUnrepresentableExpiry(t *testing.T) {
	f := setupMCPCore(t)
	for _, expiresAt := range []string{"253402300800", "8640000000001", "9223372036854775807", "9223372036854775808"} {
		body := fmt.Sprintf(`{"name":"越界时间","tools":["newapi_context"],"expires_at":%s}`, expiresAt)
		response := managementRequest(f.engine, http.MethodPost, "/api/mcp/keys", f.session, "192.0.2.10", body)
		assert.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
		assert.NotContains(t, response.Body.String(), `"secret"`)
	}
	var count int64
	require.NoError(t, model.DB.Model(&model.MCPKey{}).Where("user_id = ?", f.user.Id).Count(&count).Error)
	assert.EqualValues(t, 1, count, "拒绝请求不能创建测试初始密钥之外的记录")
}

func (f mcpCoreFixture) rpc(body string, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Authorization", "Bearer "+f.secret)
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	f.engine.ServeHTTP(recorder, request)
	return recorder
}

func TestMCPCoreTransport(t *testing.T) {
	f := setupMCPCore(t)
	initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"unknown","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`
	r := f.rpc(initialize, nil)
	require.Equal(t, http.StatusOK, r.Code, r.Body.String())
	assert.Contains(t, r.Body.String(), `"protocolVersion":"2025-11-25"`)
	for _, version := range []string{"2025-11-25", "2025-06-18", "2025-03-26"} {
		r = f.rpc(strings.Replace(initialize, "unknown", version, 1), map[string]string{"MCP-Protocol-Version": version})
		assert.Equal(t, http.StatusOK, r.Code)
		assert.Contains(t, r.Body.String(), `"protocolVersion":"`+version+`"`)
	}
	for _, tc := range []struct {
		name, body string
		headers    map[string]string
		status     int
	}{
		{"未知协议头", `{"jsonrpc":"2.0","id":2,"method":"ping"}`, map[string]string{"MCP-Protocol-Version": "unsupported"}, 400},
		{"恶意来源", `{"jsonrpc":"2.0","id":2,"method":"ping"}`, map[string]string{"Origin": "https://evil.example"}, 403},
		{"缺少SSE", `{"jsonrpc":"2.0","id":2,"method":"ping"}`, map[string]string{"Accept": "application/json"}, 406},
		{"拒绝SSE", `{"jsonrpc":"2.0","id":2,"method":"ping"}`, map[string]string{"Accept": "application/json,text/event-stream;q=0"}, 406},
		{"数组批处理", `[{"jsonrpc":"2.0","id":2,"method":"ping"}]`, nil, 400},
		{"空ID", `{"jsonrpc":"2.0","id":null,"method":"ping"}`, nil, 400},
		{"小数ID", `{"jsonrpc":"2.0","id":1.5,"method":"ping"}`, nil, 400},
		{"数组参数", `{"jsonrpc":"2.0","id":2,"method":"ping","params":[]}`, nil, 400},
		{"解析错误", `{`, nil, 400},
		{"初始化通知", `{"jsonrpc":"2.0","method":"notifications/initialized"}`, nil, 202},
		{"工具通知不执行", `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"newapi_channel_status_set"}}`, nil, 202},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := f.rpc(tc.body, tc.headers)
			assert.Equal(t, tc.status, response.Code, response.Body.String())
			if tc.status == 202 {
				assert.Empty(t, response.Body.String())
			}
		})
	}
	r = httptest.NewRecorder()
	f.engine.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/mcp", nil))
	assert.Equal(t, http.StatusMethodNotAllowed, r.Code)
	r = f.rpc(strings.Repeat("x", service.MCPMaxUploadBytes+1), nil)
	assert.Equal(t, http.StatusRequestEntityTooLarge, r.Code)
}

func TestMCPCoreUncertainWritesNeverReexecute(t *testing.T) {
	f := setupMCPCore(t)
	identity, err := service.AuthenticateMCP(f.secret)
	require.NoError(t, err)
	for _, outcome := range []string{"panic", "oversize", "server-error"} {
		t.Run(outcome, func(t *testing.T) {
			calls := 0
			dispatch := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				calls++
				switch outcome {
				case "panic":
					panic("must-not-leak-secret")
				case "oversize":
					_, _ = writer.Write([]byte(strings.Repeat("x", service.MCPMaxResponseBytes+1)))
				case "server-error":
					writer.WriteHeader(500)
					_, _ = writer.Write([]byte(`{"success":false,"message":"must-not-leak-secret"}`))
				}
			})
			arguments := []byte(fmt.Sprintf(`{"body":{"name":"测试"},"request_id":"%s"}`, outcome))
			result := service.ExecuteMCPTool(t.Context(), identity, "newapi_vendor_create", arguments, dispatch)
			assert.True(t, result.IsError)
			encoded, err := common.Marshal(result)
			require.NoError(t, err)
			assert.NotContains(t, string(encoded), "must-not-leak-secret")
			result = service.ExecuteMCPTool(t.Context(), identity, "newapi_vendor_create", arguments, dispatch)
			assert.True(t, result.IsError)
			assert.Contains(t, result.Content[0].Text, "pending or uncertain")
			assert.Equal(t, 1, calls)
		})
	}
}

func TestMCPCoreCurrentIdentityAndKeyBoundary(t *testing.T) {
	f := setupMCPCore(t)
	call := `{"jsonrpc":"2.0","id":"context","method":"tools/call","params":{"name":"newapi_context","arguments":{}}}`
	r := f.rpc(call, nil)
	require.Equal(t, http.StatusOK, r.Code)
	assert.Contains(t, r.Body.String(), "NewAPI-Mao management")
	assert.NotContains(t, r.Body.String(), f.secret)
	request := httptest.NewRequest(http.MethodGet, "/api/mcp/keys", nil)
	request.Header.Set("Authorization", "Bearer "+f.secret)
	r = httptest.NewRecorder()
	f.engine.ServeHTTP(r, request)
	assert.Equal(t, http.StatusUnauthorized, r.Code, "MCP key 不得变为后台 API token")
	for _, tc := range []struct {
		field string
		value any
	}{
		{"status", common.UserStatusDisabled}, {"role", common.RoleCommonUser},
	} {
		require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", f.user.Id).Update(tc.field, tc.value).Error)
		r = f.rpc(call, nil)
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", f.user.Id).Updates(map[string]any{"role": common.RoleAdminUser, "status": common.UserStatusEnabled}).Error)
	}
	require.NoError(t, model.DB.Model(f.key).Update("expires_at", time.Now().Unix()-1).Error)
	assert.Equal(t, http.StatusUnauthorized, f.rpc(call, nil).Code)
	require.NoError(t, model.DB.Model(f.key).Update("expires_at", time.Now().Unix()+3600).Error)
	require.NoError(t, model.RevokeMCPKey(f.user.Id, f.key.ID))
	assert.Equal(t, http.StatusUnauthorized, f.rpc(call, nil).Code)
}

func TestMCPCorePermissionAndSchemaBeforeRealChannelRoute(t *testing.T) {
	f := setupMCPCore(t)
	channel := model.Channel{Name: "MCP 渠道", Status: 1, Key: "never-return", Models: "model-a", Group: "default"}
	require.NoError(t, model.DB.Create(&channel).Error)
	// 直接写数据库而不 reload，验证 MCP 不受 authz 的旧内存授权影响。
	rule := model.CasbinRule{Ptype: "p", V0: authz.UserSubject(f.user.Id), V1: "channel", V2: "operate", V3: "deny"}
	require.NoError(t, model.DB.Create(&rule).Error)
	r := f.rpc(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, nil)
	assert.NotContains(t, r.Body.String(), "newapi_channel_status_set")
	assert.Contains(t, r.Body.String(), "newapi_channel_update")
	call := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"newapi_channel_status_set","arguments":{"path":{"id":"%d"},"body":{"status":2},"request_id":"denied-1"}}}`, channel.Id)
	r = f.rpc(call, nil)
	assert.Contains(t, r.Body.String(), `"isError":true`)
	for i, field := range []string{`"status":2`, `"key":"must-not-log"`, `"base_url":"https://invalid.example"`, `"header_override":"must-not-log"`, `"param_override":"must-not-log"`} {
		call = fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"newapi_channel_update","arguments":{"body":{"id":%d,%s},"request_id":"schema-%d"}}}`, channel.Id, field, i)
		r = f.rpc(call, nil)
		assert.Contains(t, r.Body.String(), `"isError":true`, r.Body.String())
	}
	var saved model.Channel
	require.NoError(t, model.DB.First(&saved, channel.Id).Error)
	assert.Equal(t, 1, saved.Status)
	assert.Equal(t, "never-return", saved.Key)
	var logs []model.MCPCallLog
	require.NoError(t, model.DB.Find(&logs).Error)
	require.Len(t, logs, 6)
	for _, log := range logs {
		assert.False(t, log.Success)
	}
	// 直接复用内部身份请求真实路由，仍必须经过 operate 中间件。
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/channel/%d/status", channel.Id), strings.NewReader(`{"status":2}`))
	request = request.WithContext(service.MCPInternalContext(request.Context(), f.key.ID))
	request.Header.Set("Content-Type", "application/json")
	r = httptest.NewRecorder()
	f.engine.ServeHTTP(r, request)
	assert.Equal(t, http.StatusForbidden, r.Code, r.Body.String())
}
