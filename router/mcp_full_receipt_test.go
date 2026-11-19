package router

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPFullRedemptionSecretReturnedOnceWithoutPersistence(t *testing.T) {
	f := setupMCPCore(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Redemption{}))
	payment := operation_setting.GetPaymentSetting()
	previous := *payment
	payment.ComplianceConfirmed = true
	payment.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	t.Cleanup(func() { *payment = previous })
	key, secret, err := service.CreateMCPKey(f.user.ToBaseUser(), "生成兑换码", []string{"newapi_redemption_create"}, time.Now().Add(time.Hour).Unix())
	require.NoError(t, err)
	f.key, f.secret = key, secret
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"newapi_redemption_create","arguments":{"request_id":"one-secret","body":{"name":"测试码","count":1,"quota":100,"expired_time":0,"max_redeem_count":1}}}}`
	first := f.rpc(request, nil)
	require.Contains(t, first.Body.String(), `"isError":false`, first.Body.String())
	var redemption model.Redemption
	require.NoError(t, model.DB.First(&redemption).Error)
	require.NotEmpty(t, redemption.Key)
	assert.Contains(t, first.Body.String(), redemption.Key)
	second := f.rpc(request, nil)
	require.Contains(t, second.Body.String(), `"isError":false`, second.Body.String())
	assert.NotContains(t, second.Body.String(), redemption.Key)
	assert.Contains(t, second.Body.String(), "result_not_retained")
	var count int64
	require.NoError(t, model.DB.Model(&model.Redemption{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
	var writes []model.MCPWriteRequest
	require.NoError(t, model.DB.Find(&writes).Error)
	require.Len(t, writes, 1)
	assert.Equal(t, "done", writes[0].State)
	assert.NotContains(t, writes[0].Result, redemption.Key)
	var logs []model.MCPCallLog
	require.NoError(t, model.DB.Find(&logs).Error)
	encoded, err := common.Marshal(logs)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), redemption.Key)
}
