package router

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPUserSearchRealRoute(t *testing.T) {
	f := setupMCPAccountRoutes(t)
	model.InitDBColumns()
	key, secret, err := service.CreateMCPKey(f.user.ToBaseUser(), "搜索专项", []string{"newapi_users_search", "newapi_users_list"}, time.Now().Unix()+3600)
	require.NoError(t, err)
	f.key, f.secret = key, secret
	group := model.Group{Code: "search-other", Name: "另一展示分组", Status: model.GroupStatusActive}
	require.NoError(t, model.DB.Create(&group).Error)
	users := []model.User{
		{Username: "search-alpha", DisplayName: "独有显示名", Email: "unique-search@example.test", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 2147483648},
		{Username: "search-beta", DisplayName: "另一个用户", Role: common.RoleAdminUser, Status: common.UserStatusDisabled, Group: group.Code, Quota: -2147483649},
		{Username: "search-deleted", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default"},
	}
	for i := range users {
		users[i].AffCode = "srch" + strconv.Itoa(i)
		require.NoError(t, model.DB.Create(&users[i]).Error)
	}
	require.NoError(t, model.DB.Delete(&users[2]).Error)
	for _, tc := range []struct {
		name  string
		query map[string]string
		ids   []int
		total int
	}{
		{"用户名", map[string]string{"keyword": "search-alpha"}, []int{users[0].Id}, 1},
		{"显示名", map[string]string{"keyword": "独有显示名"}, []int{users[0].Id}, 1},
		{"邮箱定位但不返回邮箱", map[string]string{"keyword": "unique-search@example.test"}, []int{users[0].Id}, 1},
		{"精确ID", map[string]string{"search_type": "id", "keyword": strconv.Itoa(users[1].Id)}, []int{users[1].Id}, 1},
		{"仅用户名不匹配显示名", map[string]string{"search_type": "username", "keyword": "独有显示名"}, []int{}, 0},
		{"非法ID无结果", map[string]string{"search_type": "id", "keyword": "abc"}, []int{}, 0},
		{"组合筛选", map[string]string{"keyword": "search-", "group": group.Code, "role": strconv.Itoa(common.RoleAdminUser), "status": strconv.Itoa(common.UserStatusDisabled)}, []int{users[1].Id}, 1},
		{"分组不符排除", map[string]string{"keyword": "search-beta", "group": "default"}, []int{}, 0},
		{"角色不符排除", map[string]string{"keyword": "search-beta", "role": strconv.Itoa(common.RoleCommonUser)}, []int{}, 0},
		{"状态不符排除", map[string]string{"keyword": "search-beta", "status": strconv.Itoa(common.UserStatusEnabled)}, []int{}, 0},
		{"升序首条", map[string]string{"keyword": "search-", "sort_by": "id", "sort_order": "asc", "p": "1", "page_size": "1"}, []int{users[0].Id}, 3},
		{"降序首条", map[string]string{"keyword": "search-", "sort_by": "id", "sort_order": "desc", "p": "1", "page_size": "1"}, []int{users[2].Id}, 3},
		{"分页排序", map[string]string{"keyword": "search-", "sort_by": "id", "sort_order": "asc", "p": "2", "page_size": "1"}, []int{users[1].Id}, 3},
		{"软删除筛选", map[string]string{"keyword": "search-", "status": "-1"}, []int{users[2].Id}, 1},
		{"普通状态排除软删除", map[string]string{"keyword": "search-", "status": strconv.Itoa(common.UserStatusEnabled)}, []int{users[0].Id}, 1},
		{"无匹配", map[string]string{"keyword": "no-such-search-user"}, []int{}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := callMCPAccountRoute(t, f, "newapi_users_search", map[string]any{"query": tc.query})
			var result struct {
				Data struct {
					Items []struct {
						ID    int             `json:"id"`
						Quota json.RawMessage `json:"quota"`
					} `json:"items"`
					Total int `json:"total"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(out, &result))
			actual := make([]int, 0, len(result.Data.Items))
			for _, item := range result.Data.Items {
				actual = append(actual, item.ID)
				if item.ID == users[0].Id {
					assert.Equal(t, "2147483648", string(item.Quota))
				}
				if item.ID == users[1].Id {
					assert.Equal(t, "-2147483649", string(item.Quota))
				}
			}
			assert.Equal(t, tc.ids, actual)
			assert.Equal(t, tc.total, result.Data.Total)
			assert.NotContains(t, string(out), "unique-search@example.test")
			assert.NotContains(t, string(out), `"access_token"`)
		})
	}
	// 查询凭据没有写权限，且搜索与列表仍是两个独立授权工具。
	limited, limitedSecret, err := service.CreateMCPKey(f.user.ToBaseUser(), "只列表", []string{"newapi_users_list"}, time.Now().Unix()+3600)
	require.NoError(t, err)
	f.key, f.secret = limited, limitedSecret
	response := f.rpc(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"newapi_users_search","arguments":{"query":{"keyword":"search-alpha"}}}}`, nil)
	assert.Contains(t, response.Body.String(), `"isError":true`)
	assert.Contains(t, response.Body.String(), "Tool is unavailable or unauthorized")
	assert.NotContains(t, response.Body.String(), "search-alpha")
}
