package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSetApiRouterRegistersTaskRootWithAndWithoutTrailingSlash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)

	routes := make(map[string]bool)
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	for _, route := range []string{
		"GET /api/task",
		"GET /api/task/",
		"GET /api/task/self",
	} {
		require.True(t, routes[route], route)
	}

	for _, path := range []string{
		"/api/task",
		"/api/task/",
		"/api/task/self",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		require.Equal(t, http.StatusUnauthorized, response.Code, path)
	}
}
