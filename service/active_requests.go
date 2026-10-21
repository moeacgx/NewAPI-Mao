package service

import (
	"context"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/activerequests"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

var userActiveRegistry = sync.OnceValue(func() *activerequests.Registry {
	var store activerequests.Store
	if common.RedisEnabled && common.RDB != nil {
		store = activerequests.NewRedisStore(common.RDB)
	}
	r := activerequests.NewRegistry(store)
	go r.Run(context.Background())
	return r
})

// TrackActiveRequest 只提取允许展示的字段；发布器不读取可变的 RelayInfo。
func TrackActiveRequest(c *gin.Context, info *relaycommon.RelayInfo, format types.RelayFormat) (*activerequests.Handle, func()) {
	h := userActiveRegistry().Start(info.UserId, activerequests.Entry{
		RequestID: info.RequestId, TokenID: info.TokenId, TokenName: c.GetString("token_name"),
		ModelName: info.OriginModelName, RequestType: string(format),
		IsStream:  info.IsStream || format == types.RelayFormatOpenAIRealtime,
		StartedAt: info.StartTime.UnixMilli(),
	})
	info.OnResponseData = h.FirstResponse
	// 即使上游尚未响应取消，断开的客户端也不再占用活动请求列表。
	stop := context.AfterFunc(c.Request.Context(), h.Finish)
	return h, func() {
		stop()
		h.Finish()
	}
}

func ListUserActiveRequests(ctx context.Context, userID int, query activerequests.Query) (activerequests.Page, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	return userActiveRegistry().List(ctx, userID, query)
}
