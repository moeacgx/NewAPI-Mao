package router

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/activerequests"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminActiveRequestsRouteEnforcesAuthenticationAndSelfIsolation(t *testing.T) {
	fixture := setupTokenKeyFixture(t, false, common.RoleAdminUser, 1)
	common.GlobalApiRateLimitEnable = false
	fixture.engine = newManagementRouter(t)
	for _, account := range []tokenKeyAccount{fixture.owner, fixture.peer} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
		_, finish := service.TrackActiveRequest(c, &relaycommon.RelayInfo{UserId: account.user.Id, TokenId: account.token.Id, StartTime: time.Now()}, "openai")
		defer finish()
	}
	targetQuery := "?user_id=" + strconv.Itoa(fixture.peer.user.Id)
	for _, tc := range []struct {
		name, credential string
		status           int
	}{
		{"anonymous", "", http.StatusUnauthorized},
		{"ordinary PAT", fixture.peer.pat, http.StatusForbidden},
		{"ordinary session", fixture.peer.session, http.StatusForbidden},
		{"admin PAT", fixture.owner.pat, http.StatusOK},
		{"admin session", fixture.owner.session, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := managementRequest(fixture.engine, http.MethodGet, "/api/log/active"+targetQuery, tc.credential, "192.0.2.23", "")
			require.Equal(t, tc.status, response.Code, response.Body.String())
			if tc.status != http.StatusOK {
				return
			}
			assert.Contains(t, response.Header().Get("Cache-Control"), "no-store")
			var payload struct {
				Data struct {
					activerequests.Page
					Subject struct {
						ID int `json:"id"`
					}
				}
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
			assert.Equal(t, fixture.peer.user.Id, payload.Data.Subject.ID)
			require.Len(t, payload.Data.Items, 1)
			assert.Equal(t, fixture.peer.token.Id, payload.Data.Items[0].TokenID)
		})
	}
	// 管理员的本人接口保持会话隔离，跨用户查询只能走独立的管理员接口。
	response := managementRequest(fixture.engine, http.MethodGet, "/api/log/self/active"+targetQuery, fixture.owner.session, "192.0.2.23", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var payload struct{ Data activerequests.Page }
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Items, 1)
	assert.Equal(t, fixture.owner.token.Id, payload.Data.Items[0].TokenID)
}
