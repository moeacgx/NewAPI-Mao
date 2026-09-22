package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseLogUserID(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "omitted", raw: "", want: 0},
		{name: "positive", raw: "146", want: 146},
		{name: "zero", raw: "0", want: -1},
		{name: "negative", raw: "-1", want: -1},
		{name: "malformed", raw: "abc", want: -1},
		{name: "overflow", raw: "999999999999999999999", want: -1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, parseLogUserID(test.raw))
		})
	}
}

func TestLogUserIDEndpointsRejectInvalidIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{"/api/log/?user_id=abc", "/api/log/stat?user_id=0", "/api/log/?user_id=999999999999999999999"} {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodGet, path, nil)
		if strings.Contains(path, "/stat") {
			GetLogsStat(context)
		} else {
			GetAllLogs(context)
		}
		require.Equal(t, http.StatusBadRequest, recorder.Code, path)
	}
}
