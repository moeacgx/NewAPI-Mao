package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetStatusReturnsSystemNameAndDescriptionFromOptionSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousOptions := common.OptionMap
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptions
		common.OptionMapRWMutex.Unlock()
	})

	for _, values := range []map[string]string{
		{"SystemName": "MaoLao API", "SystemDescription": "中文站点描述"},
		{"SystemName": "MaoLao API", "SystemDescription": ""},
	} {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = values
		common.OptionMapRWMutex.Unlock()

		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(http.MethodGet, "/api/status", nil)
		GetStatus(context)

		var response struct {
			Data map[string]any `json:"data"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, values["SystemName"], response.Data["system_name"])
		require.Equal(t, values["SystemDescription"], response.Data["system_description"])
	}
}
