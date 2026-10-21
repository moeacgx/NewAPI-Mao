package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestRelayTracksInFlightRequestAndClearsAfterSettlement(t *testing.T) {
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
	assert.Zero(t, page.Total)
	var stored model.User
	require.NoError(t, model.DB.Select("quota").First(&stored, user.Id).Error)
	assert.EqualValues(t, 900, stored.Quota, "活动监控不得改变原计费结果")
}

func TestSearchTokensUsesOwnersLiveIDsBeforePagination(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	first := seedToken(t, db, 87403, "working first", "live-first-private-key")
	second := seedToken(t, db, 87403, "working second", "live-second-private-key")
	seedToken(t, db, 87403, "working inactive", "inactive-private-key")
	other := seedToken(t, db, 87404, "working other", "other-private-key")
	for _, token := range []*model.Token{first, second, other} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		_, finish := service.TrackActiveRequest(c, &relaycommon.RelayInfo{UserId: token.UserId, TokenId: token.Id, StartTime: time.Now()}, "openai")
		defer finish()
	}
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
