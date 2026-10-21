package service

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/pkg/activerequests"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackActiveRequestLifecycle(t *testing.T) {
	r := activerequests.NewRegistry(nil)
	previous := userActiveRegistry
	userActiveRegistry = func() *activerequests.Registry { return r }
	t.Cleanup(func() { userActiveRegistry = previous })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil).WithContext(ctx)
	c.Set("token_name", "同名令牌")
	info := &relaycommon.RelayInfo{UserId: 7, TokenId: 12, TokenKey: "private-key", OriginModelName: "public-model", RelayFormat: types.RelayFormatOpenAI, IsStream: true, StartTime: time.Now().Add(-time.Second)}
	h, finish := TrackActiveRequest(c, info, types.RelayFormatOpenAI)
	defer finish()
	h.Attempt(true, 0)
	info.SetFirstResponseTime()
	page, err := ListUserActiveRequests(ctx, 7, activerequests.Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	id := page.Items[0].ID
	assert.Equal(t, "receiving", page.Items[0].Status)
	assert.Equal(t, "public-model", page.Items[0].ModelName)
	assert.Equal(t, "同名令牌", page.Items[0].TokenName)
	h.Attempt(true, 1)
	info.SetFirstResponseTime()
	page, err = ListUserActiveRequests(ctx, 7, activerequests.Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, id, page.Items[0].ID)
	assert.Equal(t, "receiving", page.Items[0].Status)
	assert.Equal(t, 1, page.Items[0].RetryCount)
	require.NotNil(t, page.Items[0].FirstResponseMS)
	cancel()
	require.Eventually(t, func() bool {
		page, err := r.List(context.Background(), 7, activerequests.Query{})
		return err == nil && len(page.Items) == 1 && page.Items[0].Status == "completed"
	}, time.Second, time.Millisecond)
	finish()
	info.SetFirstResponseTime()
	page, err = r.List(context.Background(), 7, activerequests.Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "completed", page.Items[0].Status)
	require.NotNil(t, page.Items[0].CompletedAt)
	assert.Zero(t, page.Summary.Total)
	assert.Empty(t, page.Summary.ByToken)
}

func TestTrackTaskUsesInboundFormatAndPanicCleanup(t *testing.T) {
	r := activerequests.NewRegistry(nil)
	previous := userActiveRegistry
	userActiveRegistry = func() *activerequests.Registry { return r }
	t.Cleanup(func() { userActiveRegistry = previous })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/videos", nil)
	info, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
	require.NoError(t, err)
	info.UserId = 7
	assert.Panics(t, func() {
		_, finish := TrackActiveRequest(c, info, types.RelayFormatTask)
		defer finish()
		page, err := ListUserActiveRequests(context.Background(), 7, activerequests.Query{})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		assert.Equal(t, "task", page.Items[0].RequestType)
		panic("请求处理异常")
	})
	page, err := ListUserActiveRequests(context.Background(), 7, activerequests.Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "completed", page.Items[0].Status)
	assert.Zero(t, page.Summary.Total)
}
