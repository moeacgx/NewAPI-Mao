package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/activerequests"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAdminActiveRequestsSubjectAndIsolation(t *testing.T) {
	db := setupManageUserTestDB(t)
	target := model.User{Id: 87501, Username: "active-target", Password: "private-password", Email: "private@example.test", AffCode: "active-target-code"}
	other := model.User{Id: 87502, Username: "active-other", Password: "private-other-password", AffCode: "active-other-code"}
	require.NoError(t, db.Create(&target).Error)
	require.NoError(t, db.Create(&other).Error)
	for _, user := range []model.User{target, other} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
		c.Set("token_name", user.Username+" token")
		_, finish := service.TrackActiveRequest(c, &relaycommon.RelayInfo{UserId: user.Id, TokenId: user.Id, OriginModelName: "public-model", StartTime: time.Now()}, "openai")
		defer finish()
	}
	for _, selector := range []string{"user_id=" + strconv.Itoa(target.Id), "username=" + target.Username} {
		t.Run(selector, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/log/active?"+selector+"&page_size=1&token_id=87501&status=processing&search=public", nil)
			c.Set("id", other.Id)
			c.Set("role", common.RoleAdminUser)
			GetAdminActiveRequests(c)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			var response struct {
				Success bool
				Data    struct {
					activerequests.Page
					Subject struct {
						ID       int    `json:"id"`
						Username string `json:"username"`
					}
				}
			}
			require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
			require.True(t, response.Success)
			assert.Equal(t, target.Id, response.Data.Subject.ID)
			assert.Equal(t, target.Username, response.Data.Subject.Username)
			require.Len(t, response.Data.Items, 1)
			assert.Equal(t, target.Id, response.Data.Items[0].TokenID)
			assert.Equal(t, 1, response.Data.Total)
			assert.Equal(t, 1, response.Data.Summary.Total)
			assert.Equal(t, 1, response.Data.PageSize)
			for _, secret := range []string{target.Password, target.Email, "\"password\"", "\"access_token\"", other.Username} {
				assert.NotContains(t, w.Body.String(), secret)
			}
		})
	}
}

func TestGetAdminActiveRequestsValidationAndFailures(t *testing.T) {
	db := setupManageUserTestDB(t)
	target := model.User{Id: 87503, Username: "active-lookup", Password: "password", AffCode: "active-lookup-code"}
	require.NoError(t, db.Create(&target).Error)
	deleted := model.User{Id: 87505, Username: "deleted-lookup", Password: "password", AffCode: "deleted-lookup-code"}
	require.NoError(t, db.Create(&deleted).Error)
	require.NoError(t, db.Delete(&deleted).Error)
	for _, tc := range []struct {
		name, query      string
		id, role, status int
	}{
		{"unauthenticated", "?user_id=87503", 0, 0, 401},
		{"ordinary user", "?user_id=87503", 1, common.RoleCommonUser, 403},
		{"missing subject", "", 1, common.RoleAdminUser, 400},
		{"empty subject", "?username=", 1, common.RoleAdminUser, 400},
		{"blank subject", "?username=%20", 1, common.RoleAdminUser, 400},
		{"conflicting subject", "?user_id=87503&username=active-lookup", 1, common.RoleAdminUser, 400},
		{"empty conflict", "?user_id=87503&username=", 1, common.RoleAdminUser, 400},
		{"duplicate subject", "?user_id=87503&user_id=87504", 1, common.RoleAdminUser, 400},
		{"duplicate username", "?username=active-lookup&username=active-other", 1, common.RoleAdminUser, 400},
		{"zero id", "?user_id=0", 1, common.RoleAdminUser, 400},
		{"negative id", "?user_id=-1", 1, common.RoleAdminUser, 400},
		{"overflow id", "?user_id=99999999999999999999999", 1, common.RoleAdminUser, 400},
		{"long username", "?username=" + strings.Repeat("a", model.UserNameMaxLength+1), 1, common.RoleAdminUser, 400},
		{"invalid page", "?user_id=87503&page=0", 1, common.RoleAdminUser, 400},
		{"invalid status", "?user_id=87503&status=invalid", 1, common.RoleAdminUser, 400},
		{"long search", "?user_id=87503&search=" + url.QueryEscape(strings.Repeat("用", 257)), 1, common.RoleAdminUser, 400},
		{"unknown id", "?user_id=87504", 1, common.RoleAdminUser, 404},
		{"partial username", "?username=active", 1, common.RoleAdminUser, 404},
		{"wildcard username", "?username=active%25", 1, common.RoleAdminUser, 404},
		{"SQL syntax username", "?username=" + url.QueryEscape("' OR 1=1 --"), 1, common.RoleAdminUser, 404},
		{"unknown username", "?username=absent", 1, common.RoleAdminUser, 404},
		{"case mismatch", "?username=ACTIVE-LOOKUP", 1, common.RoleAdminUser, 404},
		{"deleted user id", "?user_id=87505", 1, common.RoleAdminUser, 404},
		{"deleted username", "?username=deleted-lookup", 1, common.RoleAdminUser, 404},
		{"empty connections", "?user_id=87503&status=completed", 1, common.RoleRootUser, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/log/active"+tc.query, nil)
			c.Set("id", tc.id)
			c.Set("role", tc.role)
			GetAdminActiveRequests(c)
			assert.Equal(t, tc.status, w.Code, w.Body.String())
			assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			if tc.status == http.StatusOK {
				assert.Contains(t, w.Body.String(), `"items":[]`)
			}
		})
	}
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/log/active?user_id=87503", nil)
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	GetAdminActiveRequests(c)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.NotContains(t, w.Body.String(), "database")
}

func TestGetSelfActiveRequestsScopeAndValidation(t *testing.T) {
	for _, owner := range []int{87401, 87402} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		_, finish := service.TrackActiveRequest(c, &relaycommon.RelayInfo{UserId: owner, TokenId: owner, OriginModelName: "public-model", StartTime: time.Now()}, "openai")
		defer finish()
	}
	for _, tc := range []struct {
		query         string
		owner, status int
	}{
		{"?user_id=87402&page_size=1", 87401, 200},
		{"", 0, 401},
		{"?page=0", 87401, 400},
		{"?page=9999999999999999999999999", 87401, 400},
		{"?page_size=51", 87401, 400},
		{"?token_id=-1", 87401, 400},
		{"?status=unknown", 87401, 400},
	} {
		t.Run(tc.query, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/log/self/active"+tc.query, nil)
			c.Set("id", tc.owner)
			c.Set("role", common.RoleAdminUser)
			GetSelfActiveRequests(c)
			assert.Equal(t, tc.status, w.Code)
			assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			if tc.status == 200 {
				var response struct {
					Success bool
					Data    activerequests.Page
				}
				require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
				require.True(t, response.Success)
				require.Len(t, response.Data.Items, 1)
				assert.Equal(t, tc.owner, response.Data.Items[0].TokenID)
				assert.Equal(t, 1, response.Data.Total)
			}
		})
	}
}

func TestRelayTracksInFlightRequestAndRetainsAfterSettlement(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		close(entered)
		<-release
		w.Header().Set("Content-Type", gin.MIMEJSON)
		_, _ = w.Write([]byte(`{"created":1,"data":[{"url":"https://images.example/active.png"}]}`))
	}))
	defer upstream.Close()
	defer releaseOnce.Do(func() { close(release) })
	setupCanvasRouteBillingTest(t, upstream.URL)
	user, cookie := createCanvasRouteUserSession(t, "active-request", "wallet_only", 1000)
	engine := newCanvasRouteBillingEngine(nil)
	completed := make(chan *httptest.ResponseRecorder, 1)
	go func() { completed <- performCanvasRouteImageRequest(engine, cookie) }()
	select {
	case <-entered:
	case response := <-completed:
		t.Fatalf("请求在抵达上游前结束: %s", response.Body.String())
	case <-time.After(5 * time.Second):
		t.Fatal("请求未抵达本地上游")
	}
	page, err := service.ListUserActiveRequests(context.Background(), user.Id, activerequests.Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, canvasRouteBillingModel, page.Items[0].ModelName)
	assert.Equal(t, "openai_image", page.Items[0].RequestType)
	assert.Equal(t, "processing", page.Items[0].Status)
	assert.Zero(t, page.Summary.ActiveTokens, "会话请求不虚构 API 令牌")
	releaseOnce.Do(func() { close(release) })
	select {
	case response := <-completed:
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	case <-time.After(5 * time.Second):
		t.Fatal("上游结束后未完成结算")
	}
	page, err = service.ListUserActiveRequests(context.Background(), user.Id, activerequests.Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "completed", page.Items[0].Status)
	require.NotNil(t, page.Items[0].CompletedAt)
	assert.Zero(t, page.Summary.Total)
	var stored model.User
	require.NoError(t, model.DB.Select("quota").First(&stored, user.Id).Error)
	assert.EqualValues(t, 900, stored.Quota, "活动监控不得改变原计费结果")
}

func TestGetSelfActiveRequestsAllowsEndedStatusWithoutCountingItAsActive(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	_, finish := service.TrackActiveRequest(c, &relaycommon.RelayInfo{UserId: 87406, TokenId: 12, StartTime: time.Now()}, "openai")
	finish()
	w := httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/log/self/active?status=completed", nil)
	c.Set("id", 87406)
	GetSelfActiveRequests(c)
	require.Equal(t, http.StatusOK, w.Code)
	var response struct {
		Success bool
		Data    activerequests.Page
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data.Items, 1)
	assert.Equal(t, "completed", response.Data.Items[0].Status)
	assert.NotNil(t, response.Data.Items[0].CompletedAt)
	assert.Zero(t, response.Data.Summary.Total)
	assert.Empty(t, response.Data.Summary.ByToken)
}

func TestSearchTokensUsesOwnersLiveIDsBeforePagination(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	first := seedToken(t, db, 87403, "working first", "live-first-private-key")
	second := seedToken(t, db, 87403, "working second", "live-second-private-key")
	ended := seedToken(t, db, 87403, "working inactive", "inactive-private-key")
	other := seedToken(t, db, 87404, "working other", "other-private-key")
	for _, token := range []*model.Token{first, second, other} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		_, finish := service.TrackActiveRequest(c, &relaycommon.RelayInfo{UserId: token.UserId, TokenId: token.Id, StartTime: time.Now()}, "openai")
		defer finish()
	}
	cEnded, _ := gin.CreateTestContext(httptest.NewRecorder())
	cEnded.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	_, finishEnded := service.TrackActiveRequest(cEnded, &relaycommon.RelayInfo{UserId: ended.UserId, TokenId: ended.Id, StartTime: time.Now()}, "openai")
	finishEnded()
	c, w := newAuthenticatedContext(t, http.MethodGet, "/api/token/search?active_only=true&keyword=working&p=2&page_size=1&user_id=87404", nil, 87403)
	SearchTokens(c)
	var response struct {
		Success bool
		Data    struct {
			Total int
			Items []model.Token
		}
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success, w.Body.String())
	assert.Equal(t, 2, response.Data.Total)
	require.Len(t, response.Data.Items, 1)
	assert.Equal(t, first.Id, response.Data.Items[0].Id)
	assert.NotContains(t, w.Body.String(), first.Key)
	c, w = newAuthenticatedContext(t, http.MethodGet, "/api/token/search?active_only=true", nil, 87405)
	SearchTokens(c)
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Zero(t, response.Data.Total)
	assert.Empty(t, response.Data.Items)
}
