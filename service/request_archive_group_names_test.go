package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupArchiveGroupNamesTest(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupRequestArchiveServiceTest(t)
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.GroupAlias{}))
	require.NoError(t, db.Create(&[]model.Group{
		{Id: 11, Code: "default", Name: "普通用户", Status: model.GroupStatusActive},
		{Id: 22, Code: "vip", Name: "专业请求", Status: model.GroupStatusActive},
		{Id: 44, Code: "reused", Name: "重建分组", Status: model.GroupStatusActive},
	}).Error)
	require.NoError(t, db.Create(&model.GroupAlias{GroupId: 22, Alias: "legacy-vip"}).Error)
	return db
}

func TestIssue403RequestArchiveCapturesRequestGroupIdentity(t *testing.T) {
	setupArchiveGroupNamesTest(t)
	tests := []struct {
		name, using, auditCode, auditName, selected string
		auditID, wantID                             int
		wantCode, wantName                          string
	}{
		{name: "用户归属不能冒充请求组", using: "vip", wantID: 22, wantCode: "vip", wantName: "专业请求"},
		{name: "历史别名解析实体", using: "legacy-vip", wantID: 22, wantCode: "vip", wantName: "专业请求"},
		{name: "审计稳定ID优先", using: "default", auditID: 22, auditCode: "legacy-vip", auditName: "pre_allocation:legacy-vip", wantID: 22, wantCode: "vip", wantName: "专业请求"},
		{name: "稳定ID删除不能认作重建组", using: "reused", auditID: 33, auditCode: "reused", auditName: "已删除旧组", wantID: 33, wantCode: "reused", wantName: "已删除旧组"},
		{name: "相同选组标识保留已知稳定ID", using: "reused", auditID: 33, auditCode: "reused", auditName: "已删除旧组", selected: "reused", wantID: 33, wantCode: "reused", wantName: "已删除旧组"},
		{name: "仅稳定ID删除后不拼接另一使用组", using: "default", auditID: 33, wantID: 33, wantName: "33"},
		{name: "未知标识回退", using: "unknown", wantCode: "unknown", wantName: "unknown"},
		{name: "未分配自动组不冒充实体", using: "auto", wantCode: "auto", wantName: "auto"},
		{name: "未分配多组不冒充单一实体", using: "default,vip", wantCode: "default,vip", wantName: "default,vip"},
		{name: "缺少请求组不套用用户组"},
		{name: "已分配Realtime帧使用选中组", using: "vip", auditID: 11, auditCode: "default", auditName: "pre_allocation:default", selected: "vip", wantID: 22, wantCode: "vip", wantName: "专业请求"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses?private=hidden", nil)
			common.SetContextKey(c, constant.ContextKeyUserGroupId, 11)
			common.SetContextKey(c, constant.ContextKeyUsingGroup, tt.using)
			common.SetContextKey(c, constant.ContextKeyPromptAuditGroupId, tt.auditID)
			common.SetContextKey(c, constant.ContextKeyPromptAuditGroupCode, tt.auditCode)
			common.SetContextKey(c, constant.ContextKeyPromptAuditGroupName, tt.auditName)
			common.SetContextKey(c, constant.ContextKeySelectedChannelGroup, tt.selected)
			request := BuildRequestArchiveRequest(c, []byte("原始正文"), "application/json", http.MethodPost)
			assert.Equal(t, tt.wantID, request.GroupId)
			assert.Equal(t, tt.wantCode, request.GroupCode)
			assert.Equal(t, tt.wantName, request.GroupName)
			assert.Equal(t, "/v1/responses", request.Path)
			assert.Equal(t, []byte("原始正文"), request.Body)
		})
	}
}

func TestIssue403RequestArchivePendingSnapshotAndAADSurviveRename(t *testing.T) {
	db := setupArchiveGroupNamesTest(t)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, "legacy-vip")
	request := BuildRequestArchiveRequest(c, []byte("原始归档正文"), "application/json", http.MethodPost)
	SetPendingRequestArchive(c, request)
	job := model.RequestArchiveJob{
		ArchiveId: "issue403-aad", TargetId: "local", ConfigVersion: 1,
		GroupId: request.GroupId, GroupName: request.GroupName, ByteSize: int64(len(request.Body)),
		ContentType: request.ContentType, Method: request.Method, Path: request.Path,
		CreatedAt: 1700000000, ExpiresAt: 1800000000,
	}
	var err error
	job.SHA256, err = requestArchivePlaintextDigest(&job, requestArchiveCipherVersion, request.Body)
	require.NoError(t, err)
	ciphertext, err := EncryptRequestArchiveJobPayload(request.Body, &job)
	require.NoError(t, err)
	job.RequestCiphertext = model.RequestArchiveLargeText(ciphertext)
	require.NoError(t, db.Create(&job).Error)
	require.NoError(t, db.Model(&model.Group{}).Where("id = ?", 22).Update("name", "改名后的专业组").Error)

	fresh := BuildRequestArchiveRequest(c, request.Body, request.ContentType, request.Method)
	assert.Equal(t, "改名后的专业组", fresh.GroupName)
	assert.Equal(t, "专业请求", pendingRequestArchive(c).GroupName)
	var stored model.RequestArchiveJob
	require.NoError(t, db.First(&stored, job.Id).Error)
	assert.Equal(t, "专业请求", stored.GroupName)
	assert.Equal(t, model.RequestArchiveLargeText(ciphertext), stored.RequestCiphertext)
	plaintext, err := DecryptRequestArchivePayload(&stored)
	require.NoError(t, err)
	assert.Equal(t, request.Body, plaintext)
	tampered := stored
	tampered.GroupName = fresh.GroupName
	_, err = DecryptRequestArchivePayload(&tampered)
	assert.Error(t, err, "改写历史分组名称必须仍导致AAD校验失败")
}

func TestIssue403RequestArchiveMissingDatabaseKeepsSnapshot(t *testing.T) {
	oldDB := model.DB
	model.DB = nil
	t.Cleanup(func() { model.DB = oldDB })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(c, constant.ContextKeyUserGroupId, 11)
	common.SetContextKey(c, constant.ContextKeyPromptAuditGroupId, 22)
	common.SetContextKey(c, constant.ContextKeyPromptAuditGroupCode, "vip")
	common.SetContextKey(c, constant.ContextKeyPromptAuditGroupName, "已捕获名称")
	request := BuildRequestArchiveRequest(c, nil, "", "")
	assert.Equal(t, 22, request.GroupId)
	assert.Equal(t, "已捕获名称", request.GroupName)
	assert.Zero(t, BuildRequestArchiveRequest(nil, nil, "", "").GroupId)
}
