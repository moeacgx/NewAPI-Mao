package router

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupplierApplicationMessagesAreIsolatedRecords(t *testing.T) {
	fixture := setupTokenKeyFixture(t, false, common.RoleAdminUser, 1)
	common.GlobalApiRateLimitEnable = false
	common.CriticalRateLimitEnable = false
	wasMaster := common.IsMasterNode
	common.IsMasterNode = true
	t.Cleanup(func() { common.IsMasterNode = wasMaster })
	require.NoError(t, fixture.db.AutoMigrate(
		&model.SupplierApplication{}, &model.SupplierApplicationQuote{}, &model.SupplierApplicationMessage{},
		&model.Log{}, &model.CasbinRule{}, &model.AuthzRole{},
	))
	require.NoError(t, authz.Init(fixture.db))
	oldLogDB := model.LOG_DB
	model.LOG_DB = fixture.db
	t.Cleanup(func() { model.LOG_DB = oldLogDB })
	fixture.engine = newManagementRouter(t)
	ip := "192.0.2.81"
	internalNote := "内部评估：报价偏高，暂缓"
	secret := "sk-" + strings.Repeat("a", 20)

	require.NoError(t, fixture.db.Model(&model.User{}).Where("id = ?", fixture.peer.user.Id).Updates(map[string]any{
		"quota": 88000, "used_quota": 120, "aff_quota": 30,
	}).Error)
	require.NoError(t, fixture.db.Model(&model.User{}).Where("id = ?", fixture.owner.user.Id).Updates(map[string]any{
		"quota": 66000, "used_quota": 80, "aff_quota": 15,
	}).Error)

	created := managementRequest(fixture.engine, http.MethodPost, "/api/supplier-applications", fixture.peer.session, ip, supplierApplicationBody(t, nil))
	require.Equal(t, http.StatusOK, created.Code, created.Body.String())
	var createdPayload struct {
		Data model.SupplierApplicationUserView `json:"data"`
	}
	require.NoError(t, common.Unmarshal(created.Body.Bytes(), &createdPayload))
	applicationID := createdPayload.Data.Id
	require.NotNil(t, createdPayload.Data.Messages)
	assert.Empty(t, createdPayload.Data.Messages)

	own := managementRequest(fixture.engine, http.MethodPost, "/api/supplier-applications", fixture.owner.session, ip, supplierApplicationBody(t, func(payload map[string]any) {
		payload["site_name"] = "管理员自己的站点"
		payload["contact"] = "wx-owner-009"
		payload["groups"] = []any{map[string]any{"group_name": "自有分组", "platform": "gemini", "ratio": 2}}
	}))
	require.Equal(t, http.StatusOK, own.Code, own.Body.String())
	var ownPayload struct {
		Data model.SupplierApplicationUserView `json:"data"`
	}
	require.NoError(t, common.Unmarshal(own.Body.Bytes(), &ownPayload))

	personalPath := fmt.Sprintf("/api/supplier-applications/%d/messages", applicationID)
	adminPath := fmt.Sprintf("/api/supplier-application-admin/%d/messages", applicationID)
	reviewPath := fmt.Sprintf("/api/supplier-application-admin/%d", applicationID)
	review := managementRequest(fixture.engine, http.MethodPut, reviewPath, fixture.owner.session, ip, fmt.Sprintf(`{"status":"contacted","admin_note":%q,"version":1}`, internalNote))
	require.Equal(t, http.StatusOK, review.Code, review.Body.String())

	var before model.SupplierApplication
	require.NoError(t, fixture.db.First(&before, applicationID).Error)
	var groupBefore model.Group
	require.NoError(t, fixture.db.First(&groupBefore).Error)

	// 未登录、其他申请人和普通用户不能写这条申请的对话。
	assert.Equal(t, http.StatusUnauthorized, managementRequest(fixture.engine, http.MethodPost, personalPath, "", ip, `{"type":"message","body":"你好"}`).Code)
	assert.Equal(t, http.StatusNotFound, managementRequest(fixture.engine, http.MethodPost, personalPath, fixture.owner.session, ip, `{"type":"message","body":"管理员不能走个人接口写别人的申请"}`).Code)
	assert.Equal(t, http.StatusForbidden, managementRequest(fixture.engine, http.MethodPost, adminPath, fixture.peer.session, ip, `{"type":"offer","body":"申请人不能走管理接口"}`).Code)

	messages := []struct {
		path    string
		session string
		body    string
	}{
		{personalPath, fixture.peer.session, `{"type":"message","body":"可以按月结算，先小流量试跑"}`},
		{personalPath, fixture.peer.session, `{"type":"negotiation","group_name":"官方直连议价","ratio":1.25,"body":"希望降到这个倍率"}`},
		{adminPath, fixture.owner.session, `{"type":"offer","body":"首月九五折，仅作记录"}`},
		{adminPath, fixture.owner.session, `{"type":"test_quota","quantity":100,"unit":"USD","quota_status":"pending","body":"人工登记，尚未发放"}`},
		{adminPath, fixture.owner.session, `{"type":"test_quota","quantity":0.5,"unit":"tokens","quota_status":"issued","body":"人工标记已发放，不入账"}`},
	}
	for _, message := range messages {
		response := managementRequest(fixture.engine, http.MethodPost, message.path, message.session, ip, message.body)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	}
	ownMessage := managementRequest(fixture.engine, http.MethodPost, fmt.Sprintf("/api/supplier-applications/%d/messages", ownPayload.Data.Id), fixture.owner.session, ip, `{"type":"message","body":"只属于管理员自己的申请对话"}`)
	require.Equal(t, http.StatusOK, ownMessage.Code, ownMessage.Body.String())

	// 追加消息不改申请本身，也不能覆盖已有消息。
	var after model.SupplierApplication
	require.NoError(t, fixture.db.First(&after, applicationID).Error)
	assert.Equal(t, before.Status, after.Status)
	assert.Equal(t, before.AdminNote, after.AdminNote)
	assert.Equal(t, before.Version, after.Version)
	assert.Equal(t, before.HandledBy, after.HandledBy)
	assert.Equal(t, before.UserId, after.UserId)
	var stored []model.SupplierApplicationMessage
	require.NoError(t, fixture.db.Where("application_id = ?", applicationID).Order("id ASC").Find(&stored).Error)
	require.Len(t, stored, 5)
	firstBody := stored[0].Body
	assert.Equal(t, http.StatusNotFound, managementRequest(fixture.engine, http.MethodPut, fmt.Sprintf("%s/%d", personalPath, stored[0].Id), fixture.peer.session, ip, `{"type":"message","body":"改掉第一条"}`).Code)
	assert.Equal(t, http.StatusNotFound, managementRequest(fixture.engine, http.MethodPut, fmt.Sprintf("%s/%d", adminPath, stored[0].Id), fixture.owner.session, ip, `{"type":"message","body":"改掉第一条"}`).Code)
	var unchanged model.SupplierApplicationMessage
	require.NoError(t, fixture.db.First(&unchanged, stored[0].Id).Error)
	assert.Equal(t, firstBody, unchanged.Body)

	personal := managementRequest(fixture.engine, http.MethodGet, fmt.Sprintf("/api/supplier-applications/%d", applicationID), fixture.peer.session, ip, "")
	require.Equal(t, http.StatusOK, personal.Code, personal.Body.String())
	var personalPayload struct {
		Data model.SupplierApplicationUserView `json:"data"`
	}
	require.NoError(t, common.Unmarshal(personal.Body.Bytes(), &personalPayload))
	require.Len(t, personalPayload.Data.Messages, 5)
	assert.Equal(t, model.SupplierMessageTypeMessage, personalPayload.Data.Messages[0].Type)
	assert.Equal(t, "可以按月结算，先小流量试跑", personalPayload.Data.Messages[0].Body)
	assert.Equal(t, model.SupplierMessageAuthorApplicant, personalPayload.Data.Messages[0].AuthorRole)
	assert.Equal(t, fixture.peer.user.Username, personalPayload.Data.Messages[0].AuthorName)
	assert.Equal(t, "官方直连议价", personalPayload.Data.Messages[1].GroupName)
	require.NotNil(t, personalPayload.Data.Messages[1].Ratio)
	assert.Equal(t, 1.25, *personalPayload.Data.Messages[1].Ratio)
	assert.Equal(t, "希望降到这个倍率", personalPayload.Data.Messages[1].Body)
	assert.Equal(t, model.SupplierMessageTypeOffer, personalPayload.Data.Messages[2].Type)
	assert.Equal(t, model.SupplierMessageAuthorAdmin, personalPayload.Data.Messages[2].AuthorRole)
	assert.Equal(t, "首月九五折，仅作记录", personalPayload.Data.Messages[2].Body)
	require.NotNil(t, personalPayload.Data.Messages[3].Quantity)
	assert.Equal(t, 100.0, *personalPayload.Data.Messages[3].Quantity)
	assert.Equal(t, "USD", personalPayload.Data.Messages[3].Unit)
	assert.Equal(t, model.SupplierTestQuotaStatusPending, personalPayload.Data.Messages[3].QuotaStatus)
	require.NotNil(t, personalPayload.Data.Messages[4].Quantity)
	assert.Equal(t, 0.5, *personalPayload.Data.Messages[4].Quantity)
	assert.Equal(t, model.SupplierTestQuotaStatusIssued, personalPayload.Data.Messages[4].QuotaStatus)
	for _, forbidden := range []string{internalNote, "admin_note", "handled_by", secret} {
		assert.NotContains(t, personal.Body.String(), forbidden)
	}

	adminDetail := managementRequest(fixture.engine, http.MethodGet, reviewPath, fixture.owner.session, ip, "")
	require.Equal(t, http.StatusOK, adminDetail.Code, adminDetail.Body.String())
	var adminPayload struct {
		Data model.SupplierApplicationAdminView `json:"data"`
	}
	require.NoError(t, common.Unmarshal(adminDetail.Body.Bytes(), &adminPayload))
	require.Len(t, adminPayload.Data.Messages, 5)
	assert.Equal(t, personalPayload.Data.Messages[4].Id, adminPayload.Data.Messages[4].Id)
	assert.Equal(t, internalNote, adminPayload.Data.AdminNote)

	// 申请人只能看到自己的申请，管理员自己的对话不会串过来。
	peerList := managementRequest(fixture.engine, http.MethodGet, "/api/supplier-applications", fixture.peer.session, ip, "")
	require.Equal(t, http.StatusOK, peerList.Code, peerList.Body.String())
	assert.Contains(t, peerList.Body.String(), "可以按月结算，先小流量试跑")
	assert.NotContains(t, peerList.Body.String(), "只属于管理员自己的申请对话")
	assert.NotContains(t, peerList.Body.String(), internalNote)
	assert.Equal(t, http.StatusNotFound, managementRequest(fixture.engine, http.MethodGet, fmt.Sprintf("/api/supplier-applications/%d", ownPayload.Data.Id), fixture.peer.session, ip, "").Code)
	ownerList := managementRequest(fixture.engine, http.MethodGet, "/api/supplier-applications", fixture.owner.session, ip, "")
	require.Equal(t, http.StatusOK, ownerList.Code, ownerList.Body.String())
	assert.Contains(t, ownerList.Body.String(), "只属于管理员自己的申请对话")
	assert.NotContains(t, ownerList.Body.String(), "可以按月结算，先小流量试跑")
	assert.NotContains(t, ownerList.Body.String(), internalNote)

	// 凭据和非法结构被拒绝，且不落库。
	rejected := []string{
		fmt.Sprintf(`{"type":"message","body":%q}`, secret),
		`{"type":"offer","body":"password=example-value"}`,
		`{"type":"message","body":"普通消息不能带倍率","ratio":1}`,
		`{"type":"negotiation","group_name":"缺少倍率","body":"没有建议倍率"}`,
		`{"type":"test_quota","quantity":1000000000000,"unit":"USD","quota_status":"pending","body":"数量太大"}`,
		`{"type":"test_quota","quantity":10,"unit":"USD","body":"缺少发放状态"}`,
		`{"type":"unknown","body":"不是四种类型"}`,
	}
	for _, body := range rejected {
		response := managementRequest(fixture.engine, http.MethodPost, personalPath, fixture.peer.session, ip, body)
		assert.Equal(t, http.StatusBadRequest, response.Code, body)
		assert.NotContains(t, response.Body.String(), secret)
		assert.NotContains(t, response.Body.String(), "example-value")
	}
	var count int64
	require.NoError(t, fixture.db.Model(&model.SupplierApplicationMessage{}).Where("application_id = ?", applicationID).Count(&count).Error)
	assert.Equal(t, int64(5), count)

	// 只读管理员能看对话，不能发；没有读权限时管理接口不可用。个人接口仍只返回自己的申请。
	require.NoError(t, authz.SetUserPermissions(fixture.owner.user.Id, authz.PermissionsMap{authz.ResourceSupplierApplications: {"read": true, "write": false}}))
	assert.Equal(t, http.StatusOK, managementRequest(fixture.engine, http.MethodGet, reviewPath, fixture.owner.session, ip, "").Code)
	assert.Equal(t, http.StatusForbidden, managementRequest(fixture.engine, http.MethodPost, adminPath, fixture.owner.session, ip, `{"type":"message","body":"只读不能发"}`).Code)
	require.NoError(t, authz.SetUserPermissions(fixture.owner.user.Id, authz.PermissionsMap{authz.ResourceSupplierApplications: {"read": false, "write": false}}))
	assert.Equal(t, http.StatusForbidden, managementRequest(fixture.engine, http.MethodGet, reviewPath, fixture.owner.session, ip, "").Code)
	assert.Equal(t, http.StatusForbidden, managementRequest(fixture.engine, http.MethodPost, adminPath, fixture.owner.session, ip, `{"type":"message","body":"无权限不能发"}`).Code)
	assert.Equal(t, http.StatusOK, managementRequest(fixture.engine, http.MethodGet, fmt.Sprintf("/api/supplier-applications/%d", ownPayload.Data.Id), fixture.owner.session, ip, "").Code)
	require.NoError(t, fixture.db.Model(&model.SupplierApplicationMessage{}).Where("application_id = ?", applicationID).Count(&count).Error)
	assert.Equal(t, int64(5), count)

	var peer model.User
	var owner model.User
	require.NoError(t, fixture.db.First(&peer, fixture.peer.user.Id).Error)
	require.NoError(t, fixture.db.First(&owner, fixture.owner.user.Id).Error)
	assert.Equal(t, int64(88000), peer.Quota)
	assert.Equal(t, int64(120), peer.UsedQuota)
	assert.Equal(t, int64(30), peer.AffQuota)
	assert.Equal(t, int64(66000), owner.Quota)
	assert.Equal(t, int64(80), owner.UsedQuota)
	assert.Equal(t, int64(15), owner.AffQuota)
	var groupAfter model.Group
	require.NoError(t, fixture.db.First(&groupAfter).Error)
	assert.Equal(t, groupBefore.Code, groupAfter.Code)
	assert.Equal(t, groupBefore.Name, groupAfter.Name)
	assert.False(t, fixture.db.Migrator().HasTable("channels"))
	assert.False(t, fixture.db.Migrator().HasTable("redemptions"))

	var logs []model.Log
	require.NoError(t, fixture.db.Find(&logs).Error)
	require.NotEmpty(t, logs)
	sawMessageAudit := false
	for _, entry := range logs {
		blob := entry.Content + "\n" + entry.Other
		for _, forbidden := range []string{secret, "example-value", internalNote, "wx-supplier-001", "可以按月结算，先小流量试跑", "首月九五折，仅作记录", "希望降到这个倍率", "人工登记，尚未发放"} {
			assert.NotContains(t, blob, forbidden)
		}
		if strings.Contains(entry.Content, "supplier_application.message") || strings.Contains(entry.Content, "Recorded ") {
			sawMessageAudit = true
		}
	}
	assert.True(t, sawMessageAudit)
}
