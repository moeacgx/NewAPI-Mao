package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogSkillFilterEndpointsRejectInvalidValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, endpoint := range []struct {
		path   string
		handle gin.HandlerFunc
	}{
		{"/api/log/", GetAllLogs}, {"/api/log/self/", GetUserLogs},
		{"/api/log/stat", GetLogsStat}, {"/api/log/self/stat", GetLogsSelfStat},
	} {
		for _, value := range []string{"unused", "all", "USED", "used ", "' OR 1=1"} {
			t.Run(endpoint.path+value, func(t *testing.T) {
				recorder := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(recorder)
				ctx.Request = httptest.NewRequest(http.MethodGet, endpoint.path+"?skill="+url.QueryEscape(value), nil)
				endpoint.handle(ctx)
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "Invalid skill filter")
			})
		}
	}
}
