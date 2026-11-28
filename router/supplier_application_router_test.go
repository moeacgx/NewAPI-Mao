package router

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func supplierApplicationBody(t *testing.T, mutate func(payload map[string]any)) string {
	t.Helper()
	payload := map[string]any{
		"site_name":    "测试供货站",
		"site_url":     "https://supplier.example.com/panel",
		"contact_type": "wechat",
		"contact":      "wx-supplier-001",
		"remark":       "按月结算，充值比例 1:1",
		"groups": []any{
			map[string]any{"group_name": "官方直连", "platform": "openai", "ratio": 0.35, "models": []any{"gpt-5", "gpt-5-mini"}, "description": "并发 50，可外接"},
			map[string]any{"group_name": "逆向备用", "platform": "anthropic", "ratio": 1, "models": []any{}, "description": ""},
		},
	}
	if mutate != nil {
		mutate(payload)
	}
	data, err := common.Marshal(payload)
	require.NoError(t, err)
	return string(data)
}

func TestSupplierApplicationRoutesIsolatePersonalDataAndAdminProcessing(t *testing.T) {
	fixture := setupTokenKeyFixture(t, false, common.RoleAdminUser, 1)
	common.GlobalApiRateLimitEnable = false
	common.CriticalRateLimitEnable = false
	wasMaster := common.IsMasterNode
	common.IsMasterNode = true
	t.Cleanup(func() { common.IsMasterNode = wasMaster })
	require.NoError(t, fixture.db.AutoMigrate(&model.SupplierApplication{}, &model.SupplierApplicationQuote{}, &model.SupplierApplicationMessage{}, &model.SupplierApplicationRead{}, &model.Log{}, &model.CasbinRule{}, &model.AuthzRole{}))
	require.NoError(t, authz.Init(fixture.db))
	oldLogDB := model.LOG_DB
	model.LOG_DB = fixture.db
	t.Cleanup(func() { model.LOG_DB = oldLogDB })
	fixture.engine = newManagementRouter(t)
	ip := "192.0.2.61"
	internalNote := "内部评估：报价偏高，暂缓"

	// 未登录不能提交或读取。
	assert.Equal(t, http.StatusUnauthorized, managementRequest(fixture.engine, http.MethodPost, "/api/supplier-applications", "", ip, supplierApplicationBody(t, nil)).Code)
	assert.Equal(t, http.StatusUnauthorized, managementRequest(fixture.engine, http.MethodGet, "/api/supplier-applications", "", ip, "").Code)

	// 客户端注入的归属、状态、备注和处理人被忽略。
	injected := supplierApplicationBody(t, func(payload map[string]any) {
		payload["user_id"] = fixture.owner.user.Id
		payload["status"] = "cooperating"
		payload["admin_note"] = "伪造备注"
		payload["handled_by"] = fixture.owner.user.Id
	})
	created := managementRequest(fixture.engine, http.MethodPost, "/api/supplier-applications", fixture.peer.session, ip, injected)
	require.Equal(t, http.StatusOK, created.Code, created.Body.String())
	var createdPayload struct {
		Data model.SupplierApplicationUserView `json:"data"`
	}
	require.NoError(t, common.Unmarshal(created.Body.Bytes(), &createdPayload))
	applicationID := createdPayload.Data.Id
	assert.Equal(t, model.SupplierApplicationStatusPending, createdPayload.Data.Status)
	require.Len(t, createdPayload.Data.Groups, 2)
	assert.Equal(t, 0.35, createdPayload.Data.Groups[0].Ratio)
	assert.Equal(t, []string{"gpt-5", "gpt-5-mini"}, createdPayload.Data.Groups[0].Models)
	var stored model.SupplierApplication
	require.NoError(t, fixture.db.First(&stored, applicationID).Error)
	assert.Equal(t, fixture.peer.user.Id, stored.UserId)
	assert.Zero(t, stored.HandledBy)
	assert.Empty(t, stored.AdminNote)

	// 相同内容重复提交被拒绝，且只保存一份。
	duplicate := managementRequest(fixture.engine, http.MethodPost, "/api/supplier-applications", fixture.peer.session, ip, supplierApplicationBody(t, nil))
	assert.Equal(t, http.StatusConflict, duplicate.Code, duplicate.Body.String())
	var count int64
	require.NoError(t, fixture.db.Model(&model.SupplierApplication{}).Where("user_id = ?", fixture.peer.user.Id).Count(&count).Error)
	assert.Equal(t, int64(1), count)

	// 管理员自己的申请与申请人的申请互相隔离，个人接口忽略 user_id。
	own := managementRequest(fixture.engine, http.MethodPost, "/api/supplier-applications", fixture.owner.session, ip, supplierApplicationBody(t, func(payload map[string]any) {
		payload["site_name"] = "管理员自己的站点"
		payload["contact"] = "wx-owner-002"
		payload["groups"] = []any{map[string]any{"group_name": "自有分组", "platform": "gemini", "ratio": 2}}
	}))
	require.Equal(t, http.StatusOK, own.Code, own.Body.String())
	list := managementRequest(fixture.engine, http.MethodGet, fmt.Sprintf("/api/supplier-applications?user_id=%d", fixture.peer.user.Id), fixture.owner.session, ip, "")
	require.Equal(t, http.StatusOK, list.Code)
	assert.Contains(t, list.Body.String(), "管理员自己的站点")
	assert.NotContains(t, list.Body.String(), "测试供货站")
	assert.Equal(t, http.StatusNotFound, managementRequest(fixture.engine, http.MethodGet, fmt.Sprintf("/api/supplier-applications/%d", applicationID), fixture.owner.session, ip, "").Code)
	assert.Equal(t, http.StatusOK, managementRequest(fixture.engine, http.MethodGet, fmt.Sprintf("/api/supplier-applications/%d", applicationID), fixture.peer.session, ip, "").Code)

	// 普通用户不能访问管理接口。
	adminPath := fmt.Sprintf("/api/supplier-application-admin/%d", applicationID)
	for _, credential := range []string{fixture.peer.session, fixture.peer.pat} {
		assert.Equal(t, http.StatusForbidden, managementRequest(fixture.engine, http.MethodGet, "/api/supplier-application-admin", credential, ip, "").Code)
		assert.Equal(t, http.StatusForbidden, managementRequest(fixture.engine, http.MethodGet, adminPath, credential, ip, "").Code)
		assert.Equal(t, http.StatusForbidden, managementRequest(fixture.engine, http.MethodPut, adminPath, credential, ip, `{"status":"cooperating","version":1}`).Code)
	}
	assert.Equal(t, http.StatusUnauthorized, managementRequest(fixture.engine, http.MethodGet, "/api/supplier-application-admin", "", ip, "").Code)

	// 管理员看到完整资料，可按分组名搜索、按状态筛选。
	adminList := managementRequest(fixture.engine, http.MethodGet, "/api/supplier-application-admin?keyword="+"%E5%AE%98%E6%96%B9%E7%9B%B4%E8%BF%9E&status=pending", fixture.owner.session, ip, "")
	require.Equal(t, http.StatusOK, adminList.Code, adminList.Body.String())
	assert.Contains(t, adminList.Body.String(), "wx-supplier-001")
	assert.Contains(t, adminList.Body.String(), "官方直连")
	assert.NotContains(t, adminList.Body.String(), "管理员自己的站点")
	assert.Contains(t, adminList.Body.String(), `"username":"`+fixture.peer.user.Username+`"`)
	none := managementRequest(fixture.engine, http.MethodGet, "/api/supplier-application-admin?keyword=%E4%B8%8D%E5%AD%98%E5%9C%A8", fixture.owner.session, ip, "")
	require.Equal(t, http.StatusOK, none.Code)
	assert.Contains(t, none.Body.String(), `"total":0`)
	assert.Equal(t, http.StatusBadRequest, managementRequest(fixture.engine, http.MethodGet, "/api/supplier-application-admin?status=bogus", fixture.owner.session, ip, "").Code)

	// 处理：记录状态、内部备注、处理人和处理时间，并写入管理审计。
	review := managementRequest(fixture.engine, http.MethodPut, adminPath, fixture.owner.session, ip, fmt.Sprintf(`{"status":"contacted","admin_note":%q,"version":1}`, internalNote))
	require.Equal(t, http.StatusOK, review.Code, review.Body.String())
	var reviewPayload struct {
		Data model.SupplierApplicationAdminView `json:"data"`
	}
	require.NoError(t, common.Unmarshal(review.Body.Bytes(), &reviewPayload))
	assert.Equal(t, model.SupplierApplicationStatusContacted, reviewPayload.Data.Status)
	assert.Equal(t, internalNote, reviewPayload.Data.AdminNote)
	assert.Equal(t, fixture.owner.user.Id, reviewPayload.Data.HandledBy)
	assert.Equal(t, fixture.owner.user.Username, reviewPayload.Data.HandledByName)
	assert.Positive(t, reviewPayload.Data.HandledAt)
	assert.Equal(t, int64(2), reviewPayload.Data.Version)
	var auditCount int64
	require.NoError(t, fixture.db.Model(&model.Log{}).Where("user_id = ? AND content LIKE ?", fixture.owner.user.Id, "%supplier_application%").Or("user_id = ? AND content LIKE ?", fixture.owner.user.Id, "Reviewed supplier application%").Count(&auditCount).Error)
	assert.Positive(t, auditCount)
	var auditContents []string
	require.NoError(t, fixture.db.Model(&model.Log{}).Where("user_id = ?", fixture.owner.user.Id).Pluck("content", &auditContents).Error)
	for _, content := range auditContents {
		assert.NotContains(t, content, internalNote)
		assert.NotContains(t, content, "wx-supplier-001")
	}

	// 过期版本不能覆盖；非法状态、超长备注被拒绝。
	assert.Equal(t, http.StatusConflict, managementRequest(fixture.engine, http.MethodPut, adminPath, fixture.owner.session, ip, `{"status":"rejected","admin_note":"x","version":1}`).Code)
	assert.Equal(t, http.StatusBadRequest, managementRequest(fixture.engine, http.MethodPut, adminPath, fixture.owner.session, ip, `{"status":"bogus","version":2}`).Code)
	assert.Equal(t, http.StatusBadRequest, managementRequest(fixture.engine, http.MethodPut, adminPath, fixture.owner.session, ip, fmt.Sprintf(`{"status":"rejected","admin_note":%q,"version":2}`, strings.Repeat("备", 2001))).Code)
	assert.Equal(t, http.StatusNotFound, managementRequest(fixture.engine, http.MethodPut, "/api/supplier-application-admin/999999", fixture.owner.session, ip, `{"status":"rejected","version":1}`).Code)

	// 申请人能看到状态，但看不到内部备注、处理人和处理时间。
	for _, path := range []string{fmt.Sprintf("/api/supplier-applications/%d", applicationID), "/api/supplier-applications"} {
		personal := managementRequest(fixture.engine, http.MethodGet, path, fixture.peer.session, ip, "")
		require.Equal(t, http.StatusOK, personal.Code)
		assert.Contains(t, personal.Body.String(), `"status":"contacted"`)
		for _, forbidden := range []string{internalNote, "admin_note", "handled_by", "handled_at", "user_id", "version"} {
			assert.NotContains(t, personal.Body.String(), forbidden, path)
		}
	}

	// 处理不创建渠道、不改动分组和计费配置。
	var channelCount int64
	if fixture.db.Migrator().HasTable(&model.Channel{}) {
		require.NoError(t, fixture.db.Model(&model.Channel{}).Count(&channelCount).Error)
		assert.Zero(t, channelCount)
	}
	var groupCount int64
	require.NoError(t, fixture.db.Model(&model.Group{}).Count(&groupCount).Error)
	assert.Equal(t, int64(1), groupCount)

	// 撤销写权限后仍可读取，不可处理；撤销读权限后不可访问。
	require.NoError(t, authz.SetUserPermissions(fixture.owner.user.Id, authz.PermissionsMap{authz.ResourceSupplierApplications: {"read": true, "write": false}}))
	assert.Equal(t, http.StatusOK, managementRequest(fixture.engine, http.MethodGet, "/api/supplier-application-admin", fixture.owner.session, ip, "").Code)
	assert.Equal(t, http.StatusForbidden, managementRequest(fixture.engine, http.MethodPut, adminPath, fixture.owner.session, ip, `{"status":"cooperating","version":2}`).Code)
	require.NoError(t, authz.SetUserPermissions(fixture.owner.user.Id, authz.PermissionsMap{authz.ResourceSupplierApplications: {"read": false, "write": false}}))
	assert.Equal(t, http.StatusForbidden, managementRequest(fixture.engine, http.MethodGet, "/api/supplier-application-admin", fixture.owner.session, ip, "").Code)
	assert.Equal(t, http.StatusOK, managementRequest(fixture.engine, http.MethodGet, "/api/supplier-applications", fixture.owner.session, ip, "").Code)
	require.NoError(t, authz.ClearUserPermissions(fixture.owner.user.Id))
}

func TestSupplierApplicationRoutesRejectInvalidSubmissions(t *testing.T) {
	fixture := setupTokenKeyFixture(t, false, common.RoleAdminUser, 1)
	common.GlobalApiRateLimitEnable = false
	common.CriticalRateLimitEnable = false
	require.NoError(t, fixture.db.AutoMigrate(&model.SupplierApplication{}, &model.SupplierApplicationQuote{}))
	fixture.engine = newManagementRouter(t)
	ip := "192.0.2.62"
	group := func(mutate func(map[string]any)) []any {
		item := map[string]any{"group_name": "g", "platform": "openai", "ratio": 1}
		if mutate != nil {
			mutate(item)
		}
		return []any{item}
	}
	manyGroups := make([]any, model.SupplierApplicationMaxGroups+1)
	for index := range manyGroups {
		manyGroups[index] = map[string]any{"group_name": fmt.Sprintf("g%d", index), "platform": "openai", "ratio": 1}
	}
	manyModels := make([]any, model.SupplierQuoteMaxModels+1)
	for index := range manyModels {
		manyModels[index] = fmt.Sprintf("m%d", index)
	}
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"站点名称缺失", func(p map[string]any) { p["site_name"] = "  " }},
		{"站点名称过长", func(p map[string]any) { p["site_name"] = strings.Repeat("站", 101) }},
		{"站点名称含换行", func(p map[string]any) { p["site_name"] = "a\nb" }},
		{"地址使用 javascript 协议", func(p map[string]any) { p["site_url"] = "javascript:alert(1)" }},
		{"地址使用 ftp 协议", func(p map[string]any) { p["site_url"] = "ftp://example.com" }},
		{"地址缺少主机", func(p map[string]any) { p["site_url"] = "https://" }},
		{"地址内嵌账号密码", func(p map[string]any) { p["site_url"] = "https://user:pw@example.com" }},
		{"地址过长", func(p map[string]any) { p["site_url"] = "https://example.com/" + strings.Repeat("a", 500) }},
		{"联系方式类型未知", func(p map[string]any) { p["contact_type"] = "fax" }},
		{"联系方式缺失", func(p map[string]any) { p["contact"] = "" }},
		{"备注过长", func(p map[string]any) { p["remark"] = strings.Repeat("备", 1001) }},
		{"备注包含接入密钥", func(p map[string]any) { p["remark"] = "我的 api_key: fake-value" }},
		{"备注包含 sk 形态密钥", func(p map[string]any) { p["remark"] = "sk-" + strings.Repeat("a", 24) }},
		{"说明包含密码", func(p map[string]any) {
			p["groups"] = group(func(g map[string]any) { g["description"] = "password: hunter2" })
		}},
		{"没有分组", func(p map[string]any) { p["groups"] = []any{} }},
		{"分组缺失", func(p map[string]any) { delete(p, "groups") }},
		{"分组超过上限", func(p map[string]any) { p["groups"] = manyGroups }},
		{"分组名称缺失", func(p map[string]any) { p["groups"] = group(func(g map[string]any) { g["group_name"] = "" }) }},
		{"平台未知", func(p map[string]any) { p["groups"] = group(func(g map[string]any) { g["platform"] = "unknown" }) }},
		{"倍率缺失", func(p map[string]any) { p["groups"] = group(func(g map[string]any) { delete(g, "ratio") }) }},
		{"倍率为 0", func(p map[string]any) { p["groups"] = group(func(g map[string]any) { g["ratio"] = 0 }) }},
		{"倍率为负数", func(p map[string]any) { p["groups"] = group(func(g map[string]any) { g["ratio"] = -0.5 }) }},
		{"倍率超过上限", func(p map[string]any) { p["groups"] = group(func(g map[string]any) { g["ratio"] = 1000.01 }) }},
		{"倍率小数位过多", func(p map[string]any) { p["groups"] = group(func(g map[string]any) { g["ratio"] = 0.12345 }) }},
		{"倍率为字符串", func(p map[string]any) { p["groups"] = group(func(g map[string]any) { g["ratio"] = "abc" }) }},
		{"模型过多", func(p map[string]any) { p["groups"] = group(func(g map[string]any) { g["models"] = manyModels }) }},
		{"模型重复", func(p map[string]any) {
			p["groups"] = group(func(g map[string]any) { g["models"] = []any{"gpt-5", "gpt-5"} })
		}},
		{"模型为空串", func(p map[string]any) { p["groups"] = group(func(g map[string]any) { g["models"] = []any{" "} }) }},
		{"模型名过长", func(p map[string]any) {
			p["groups"] = group(func(g map[string]any) { g["models"] = []any{strings.Repeat("m", 101)} })
		}},
		{"分组说明过长", func(p map[string]any) {
			p["groups"] = group(func(g map[string]any) { g["description"] = strings.Repeat("说", 501) })
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := managementRequest(fixture.engine, http.MethodPost, "/api/supplier-applications", fixture.peer.session, ip, supplierApplicationBody(t, tc.mutate))
			assert.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
		})
	}
	for name, body := range map[string]string{"非 JSON": "not-json", "尾随第二段 JSON": supplierApplicationBody(t, nil) + `{"x":1}`, "空请求": ""} {
		response := managementRequest(fixture.engine, http.MethodPost, "/api/supplier-applications", fixture.peer.session, ip, body)
		assert.Equal(t, http.StatusBadRequest, response.Code, name)
	}
	oversized := supplierApplicationBody(t, func(p map[string]any) { p["remark"] = strings.Repeat("x", 70<<10) })
	assert.Equal(t, http.StatusRequestEntityTooLarge, managementRequest(fixture.engine, http.MethodPost, "/api/supplier-applications", fixture.peer.session, ip, oversized).Code)
	var count int64
	require.NoError(t, fixture.db.Model(&model.SupplierApplication{}).Count(&count).Error)
	assert.Zero(t, count, "任何非法提交都不得落库")

	// 边界值合法：20 个分组、20 个模型、倍率下限/上限、纯 http 地址、省略地址。
	boundary := supplierApplicationBody(t, func(p map[string]any) {
		groups := make([]any, model.SupplierApplicationMaxGroups)
		for index := range groups {
			groups[index] = map[string]any{"group_name": fmt.Sprintf("g%d", index), "platform": "other", "ratio": 1000}
		}
		groups[0] = map[string]any{"group_name": "min", "platform": "gemini", "ratio": 0.0001, "models": manyModels[:model.SupplierQuoteMaxModels]}
		p["groups"] = groups
		p["site_url"] = "http://example.com:8080"
	})
	response := managementRequest(fixture.engine, http.MethodPost, "/api/supplier-applications", fixture.peer.session, ip, boundary)
	assert.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = managementRequest(fixture.engine, http.MethodPost, "/api/supplier-applications", fixture.peer.session, ip, supplierApplicationBody(t, func(p map[string]any) { p["site_url"] = ""; p["remark"] = "" }))
	assert.Equal(t, http.StatusOK, response.Code, response.Body.String())
}

func TestSupplierApplicationSubmissionGuardsConcurrencyAndDailyLimit(t *testing.T) {
	fixture := setupTokenKeyFixture(t, false, common.RoleCommonUser, 1)
	require.NoError(t, fixture.db.AutoMigrate(&model.SupplierApplication{}, &model.SupplierApplicationQuote{}))
	var input model.SupplierApplicationInput
	require.NoError(t, common.UnmarshalJsonStr(supplierApplicationBody(t, nil), &input))

	var wg sync.WaitGroup
	results := make(chan error, 6)
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := model.CreateSupplierApplication(fixture.owner.user.Id, input)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
			continue
		}
		assert.ErrorIs(t, err, model.ErrSupplierApplicationDuplicate)
	}
	assert.Equal(t, 1, succeeded, "并发提交相同内容只能成功一次")
	var count int64
	require.NoError(t, fixture.db.Model(&model.SupplierApplication{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
	require.NoError(t, fixture.db.Model(&model.SupplierApplicationQuote{}).Count(&count).Error)
	assert.Equal(t, int64(len(input.Groups)), count, "重复提交不得留下孤立报价")

	// 24 小时内最多 20 次不同内容的提交。
	for index := 1; index < model.SupplierApplicationDailyLimit; index++ {
		input.SiteName = fmt.Sprintf("站点%d", index)
		_, err := model.CreateSupplierApplication(fixture.owner.user.Id, input)
		require.NoError(t, err)
	}
	input.SiteName = "第二十一个站点"
	_, err := model.CreateSupplierApplication(fixture.owner.user.Id, input)
	assert.ErrorIs(t, err, model.ErrSupplierApplicationLimit)
	_, err = model.CreateSupplierApplication(fixture.peer.user.Id, input)
	assert.NoError(t, err, "限额按用户独立计算")
}
