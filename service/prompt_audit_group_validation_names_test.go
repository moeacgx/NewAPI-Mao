package service

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptAuditGroupValidationNames(t *testing.T) {
	for _, whitelist := range []bool{false, true} {
		t.Run(fmt.Sprintf("whitelist_%t", whitelist), func(t *testing.T) {
			db := setupPromptAuditServiceTest(t, false, false, nil)
			require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Group{}, &model.GroupAlias{}))
			isolateSecurityAuditBuiltinOptionState(t)
			group := model.Group{Code: "vip", Name: "当前贵宾组", Status: model.GroupStatusDisabled}
			require.NoError(t, db.Create(&group).Error)
			require.NoError(t, db.Model(&group).Update("status", model.GroupStatusDisabled).Error)
			const alias = "legacy-premium"
			require.NoError(t, db.Create(&model.GroupAlias{Alias: alias, GroupId: group.Id}).Error)
			policy, err := GetSecurityAuditBuiltinPolicy()
			require.NoError(t, err)
			for _, name := range []string{"当前贵宾组", "改名后的贵宾组", ""} {
				require.NoError(t, db.Model(&model.Group{}).Where("id = ?", group.Id).Update("name", name).Error)
				for _, identifier := range []string{group.Code, alias, "unknown-group"} {
					t.Run(name+"/"+identifier, func(t *testing.T) {
						codes := []string{identifier}
						target := PromptAuditUpstreamPolicyTargetGroups
						req := SecurityAuditBuiltinPolicyUpdateRequest{ExpectedConfigVersion: policy.ConfigVersion}
						usage := "官方风控作用范围"
						if whitelist {
							req.CyberPolicyAutoBanExemptGroupCodes = &codes
							usage = "cyber_policy 自动封禁分组白名单"
						} else {
							req.UpstreamPolicyTargetType = &target
							req.UpstreamPolicyGroupCodes = &codes
						}
						wantName := name
						if wantName == "" || identifier == "unknown-group" {
							wantName = identifier
						}
						_, err := SaveSecurityAuditBuiltinPolicy(req, 23)
						require.EqualError(t, err, fmt.Sprintf("%s 引用的分组不存在或已停用：%s", usage, wantName))
						stored, err := GetSecurityAuditBuiltinPolicy()
						require.NoError(t, err)
						assert.Equal(t, policy.ConfigVersion, stored.ConfigVersion)
						assert.Equal(t, policy.UpstreamPolicyGroupCodes, stored.UpstreamPolicyGroupCodes)
						assert.Equal(t, policy.CyberPolicyAutoBanExemptGroupCodes, stored.CyberPolicyAutoBanExemptGroupCodes)
					})
				}
			}
		})
	}
}
