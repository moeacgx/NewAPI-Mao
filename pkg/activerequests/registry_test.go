package activerequests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryIsolatesUsersAndCountsTokensAcrossPages(t *testing.T) {
	r := NewRegistry(nil)
	now := time.Now()
	for i := 0; i < 24; i++ {
		r.Start(7, Entry{TokenID: 10 + i%2, TokenName: "开发令牌", ModelName: "claude", IsStream: i%3 != 0, StartedAt: now.Add(time.Duration(i) * time.Millisecond).UnixMilli()})
	}
	r.Start(8, Entry{TokenID: 30, ModelName: "secret"})
	page, err := r.List(context.Background(), 7, Query{Page: 2, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, 24, page.Total)
	assert.Len(t, page.Items, 10)
	assert.Equal(t, 2, page.Summary.ActiveTokens)
	assert.Equal(t, 12, page.Summary.ByToken["10"])
	assert.Equal(t, 16, page.Summary.Streaming)
	assert.Equal(t, 8, page.Summary.NonStreaming)
	assert.Equal(t, 16, page.Summary.Waiting)
	for _, item := range page.Items {
		assert.Equal(t, "claude", item.ModelName)
	}
	page, err = r.List(context.Background(), 7, Query{Page: 1, PageSize: 3, TokenID: 10, Search: "开发", Status: "waiting_response"})
	require.NoError(t, err)
	assert.Equal(t, 8, page.Total)
	assert.Equal(t, 24, page.Summary.Total)
	assert.Equal(t, "instance", page.Scope)
}

func TestRequestFirstResponseRetryAndFinish(t *testing.T) {
	r := NewRegistry(nil)
	started := time.Now().Add(-time.Second)
	h := r.Start(7, Entry{RequestID: "client-id", TokenID: 12, IsStream: true, StartedAt: started.UnixMilli()})
	h.FirstResponse(started.Add(250 * time.Millisecond))
	page, err := r.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "receiving", page.Items[0].Status)
	require.NotNil(t, page.Items[0].FirstResponseMS)
	assert.EqualValues(t, 250, *page.Items[0].FirstResponseMS)
	h.Attempt(true, 1)
	page, err = r.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	assert.Equal(t, "retrying", page.Items[0].Status)
	assert.Nil(t, page.Items[0].FirstResponseMS)
	h.Finish()
	h.FirstResponse(time.Now())
	h.Finish()
	page, err = r.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "completed", page.Items[0].Status)
	require.NotNil(t, page.Items[0].CompletedAt)
	assert.Zero(t, page.Summary.Total)
	assert.Zero(t, page.Summary.Streaming)
	assert.Zero(t, page.Summary.NonStreaming)
	assert.Zero(t, page.Summary.Waiting)
	assert.Empty(t, page.Summary.ByToken)
}

func TestFinishedRequestExpiresAtFiveSecondsWithoutExtendingOnLateCallbacks(t *testing.T) {
	r := NewRegistry(nil)
	now := time.Unix(1700000000, 0)
	r.now = func() time.Time { return now }
	h := r.Start(7, Entry{TokenID: 12, IsStream: true})
	h.Finish()
	finishedAt := now.UnixMilli()
	now = now.Add(4999 * time.Millisecond)
	h.Attempt(true, 2)
	h.FirstResponse(now)
	h.Finish()
	page, err := r.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "completed", page.Items[0].Status)
	require.NotNil(t, page.Items[0].CompletedAt)
	assert.Equal(t, finishedAt, *page.Items[0].CompletedAt)
	assert.Nil(t, page.Items[0].FirstResponseMS)
	assert.Zero(t, page.Items[0].RetryCount)
	assert.Zero(t, page.Summary.Total)
	now = now.Add(time.Millisecond)
	page, err = r.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	assert.Zero(t, page.Total)
	assert.Empty(t, page.Items)
	h.Finish()
	h.FirstResponse(now)
	page, err = r.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	assert.Empty(t, page.Items)
}

func TestActiveRequestsPrecedeRetainedEntriesAndSummaryExcludesEndedTokens(t *testing.T) {
	r := NewRegistry(nil)
	ended := r.Start(7, Entry{TokenID: 10, IsStream: true, StartedAt: 1000})
	r.Start(7, Entry{TokenID: 11, StartedAt: 2000})
	ended.Finish()
	page, err := r.List(context.Background(), 7, Query{PageSize: 1})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, 11, page.Items[0].TokenID)
	assert.Equal(t, 2, page.Total)
	assert.Equal(t, 1, page.Summary.Total)
	assert.Equal(t, 1, page.Summary.NonStreaming)
	assert.Zero(t, page.Summary.Streaming)
	assert.Zero(t, page.Summary.Waiting)
	assert.Equal(t, 1, page.Summary.ActiveTokens)
	assert.Equal(t, map[string]int{"11": 1}, page.Summary.ByToken)
	page, err = r.List(context.Background(), 7, Query{Status: "completed", TokenID: 10})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "completed", page.Items[0].Status)
	assert.Equal(t, 1, page.Summary.Total)
}

func TestRedisMergesNodesAndExpiresOnlyMissingLeases(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	store := NewRedisStore(client)
	nodeA, nodeB := NewRegistry(store), NewRegistry(store)
	now := time.Unix(1700000000, 0)
	nodeA.now = func() time.Time { return now }
	nodeB.now = func() time.Time { return now }
	first := nodeA.Start(7, Entry{RequestID: "same-client-id", TokenID: 10, IsStream: true})
	nodeB.Start(7, Entry{RequestID: "same-client-id", TokenID: 10})
	nodeB.Start(8, Entry{TokenID: 90, ModelName: "private-model"})
	require.NoError(t, nodeA.Publish(context.Background(), now))
	require.NoError(t, nodeB.Publish(context.Background(), now))
	page, err := nodeA.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	assert.NotEqual(t, page.Items[0].ID, page.Items[1].ID)
	assert.Equal(t, "cluster", page.Scope)
	assert.Equal(t, 1, page.Summary.ActiveTokens)
	assert.Equal(t, 2, page.Summary.ByToken["10"])
	finishedAt := now
	first.Finish()
	require.NoError(t, nodeA.Publish(context.Background(), finishedAt))
	require.NoError(t, nodeA.Publish(context.Background(), finishedAt.Add(4*time.Second)))
	items, err := store.Read(context.Background(), 7, finishedAt.Add(4999*time.Millisecond))
	require.NoError(t, err)
	assert.Len(t, items, 2, "Redis 中完成记录在 5 秒内仍应可见")
	items, err = store.Read(context.Background(), 7, finishedAt.Add(5*time.Second))
	require.NoError(t, err)
	assert.Len(t, items, 1, "Redis 中完成记录超过 5 秒后应清理")
	items, err = store.Read(context.Background(), 7, finishedAt.Add(16*time.Second))
	require.NoError(t, err)
	assert.Empty(t, items)
	keys, _ := store.keys(7)
	assert.EqualValues(t, 0, client.HLen(context.Background(), keys).Val())
}

func TestRedisFailureDoesNotInventZeroConnections(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	r := NewRegistry(NewRedisStore(client))
	mini.Close()
	_, err := r.List(context.Background(), 7, Query{})
	require.Error(t, err)
}

func TestRedisPublishesRequestThatEndedBeforeItsFirstSnapshot(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	store := NewRedisStore(client)
	writer, reader := NewRegistry(store), NewRegistry(store)
	now := time.Unix(1700000000, 0)
	writer.now = func() time.Time { return now }
	reader.now = func() time.Time { return now }
	h := writer.Start(7, Entry{TokenID: 12, IsStream: true})
	now = now.Add(time.Millisecond)
	h.Finish()
	require.NoError(t, writer.Publish(context.Background(), now))
	page, err := reader.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "completed", page.Items[0].Status)
	assert.Zero(t, page.Summary.Total)
	assert.Empty(t, page.Summary.ByToken)
	now = now.Add(5 * time.Second)
	require.NoError(t, writer.Publish(context.Background(), now))
	page, err = reader.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	assert.Empty(t, page.Items)
}

type failingWriteStore struct {
	Store
	fail bool
}

func (s *failingWriteStore) Write(ctx context.Context, records []Record, removals []Removal, now time.Time) error {
	if s.fail {
		return errors.New("write unavailable")
	}
	return s.Store.Write(ctx, records, removals, now)
}

func TestPublishFailureIsVisibleAndDeletionIsRetried(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	store := &failingWriteStore{Store: NewRedisStore(client), fail: true}
	r := NewRegistry(store)
	now := time.Unix(1700000000, 0)
	r.now = func() time.Time { return now }
	h := r.Start(7, Entry{TokenID: 10})
	require.Error(t, r.Publish(context.Background(), now))
	_, err := r.List(context.Background(), 7, Query{})
	require.Error(t, err, "写入失败时不能因为读库正常而报告零连接")
	store.fail = false
	require.NoError(t, r.Publish(context.Background(), now))
	page, err := r.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	h.Finish()
	now = now.Add(5 * time.Second)
	store.fail = true
	require.Error(t, r.Publish(context.Background(), now))
	store.fail = false
	require.NoError(t, r.Publish(context.Background(), now))
	page, err = r.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	assert.Empty(t, page.Items)
}

func TestPublicEntriesExcludeSecretsAndInternalIdentity(t *testing.T) {
	r := NewRegistry(nil)
	r.Start(7, Entry{RequestID: "req_public", TokenID: 10, TokenName: "工作", ModelName: "requested-model"})
	page, err := r.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	encoded, err := common.Marshal(page)
	require.NoError(t, err)
	for _, forbidden := range []string{"user_id", "token_key", "authorization", "channel", "prompt", "node_name"} {
		assert.NotContains(t, string(encoded), forbidden)
	}
}

type pausedWriteStore struct {
	Store
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (s *pausedWriteStore) Write(ctx context.Context, entries []Record, removals []Removal, now time.Time) error {
	s.once.Do(func() {
		close(s.entered)
		<-s.release
	})
	return s.Store.Write(ctx, entries, removals, now)
}

func TestFinishDuringPublishDoesNotResurrectRequest(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	store := &pausedWriteStore{Store: NewRedisStore(client), entered: make(chan struct{}), release: make(chan struct{})}
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(store.release) })
	r := NewRegistry(store)
	h := r.Start(7, Entry{TokenID: 10})
	done := make(chan error, 1)
	go func() { done <- r.Publish(context.Background(), time.Now()) }()
	select {
	case <-store.entered:
	case <-time.After(time.Second):
		t.Fatal("发布没有开始")
	}
	h.Finish()
	releaseOnce.Do(func() { close(store.release) })
	require.NoError(t, <-done)
	require.NoError(t, r.Publish(context.Background(), time.Now()))
	h.FirstResponse(time.Now())
	require.NoError(t, r.Publish(context.Background(), time.Now()))
	page, err := r.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	assert.Equal(t, 1, page.Total)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "completed", page.Items[0].Status)
	require.NotNil(t, page.Items[0].CompletedAt)
	expiredAt := time.UnixMilli(*page.Items[0].CompletedAt).Add(5 * time.Second)
	require.NoError(t, r.Publish(context.Background(), expiredAt))
	h.Attempt(true, 1)
	h.FirstResponse(expiredAt)
	h.Finish()
	require.NoError(t, r.Publish(context.Background(), expiredAt))
	page, err = r.List(context.Background(), 7, Query{})
	require.NoError(t, err)
	assert.Empty(t, page.Items)
}

func TestExpiredNodeDoesNotRemoveRenewedNode(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	store := NewRedisStore(client)
	stopped, running := NewRegistry(store), NewRegistry(store)
	stopped.Start(7, Entry{TokenID: 10})
	running.Start(7, Entry{TokenID: 11})
	now := time.Now()
	require.NoError(t, stopped.Publish(context.Background(), now))
	require.NoError(t, running.Publish(context.Background(), now.Add(10*time.Second)))
	entries, err := store.Read(context.Background(), 7, now.Add(16*time.Second))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, 11, entries[0].TokenID)
}
