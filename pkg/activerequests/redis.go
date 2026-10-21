package activerequests

import (
	"context"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) keys(userID int) (string, string) {
	prefix := fmt.Sprintf("active_requests:v1:{%d}:", userID)
	return prefix + "data", prefix + "leases"
}

const writeScript = `
redis.call('HSET', KEYS[1], ARGV[1], ARGV[2])
redis.call('ZADD', KEYS[2], ARGV[3], ARGV[1])
redis.call('EXPIRE', KEYS[1], 60)
redis.call('EXPIRE', KEYS[2], 60)
return 1
`

const removeScript = `
redis.call('HDEL', KEYS[1], ARGV[1])
redis.call('ZREM', KEYS[2], ARGV[1])
return 1
`

var readScript = redis.NewScript(`
local expired = redis.call('ZRANGEBYSCORE', KEYS[2], '-inf', ARGV[1])
for _, id in ipairs(expired) do
  redis.call('HDEL', KEYS[1], id)
  redis.call('ZREM', KEYS[2], id)
end
return redis.call('HVALS', KEYS[1])
`)

func (s *RedisStore) Write(ctx context.Context, entries []Record, removals []Removal, now time.Time) error {
	if len(entries) == 0 && len(removals) == 0 {
		return nil
	}
	pipe := s.client.Pipeline()
	defer pipe.Close()
	for _, record := range entries {
		payload, err := common.Marshal(record)
		if err != nil {
			return err
		}
		data, leases := s.keys(record.UserID)
		// Eval 在流水线中无需依赖预加载脚本，Redis 重启后也能正确续租。
		pipe.Eval(ctx, writeScript, []string{data, leases}, record.Entry.ID, string(payload), now.Add(15*time.Second).UnixMilli())
	}
	for _, removal := range removals {
		data, leases := s.keys(removal.UserID)
		pipe.Eval(ctx, removeScript, []string{data, leases}, removal.ID)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisStore) Read(ctx context.Context, userID int, now time.Time) ([]Entry, error) {
	data, leases := s.keys(userID)
	values, err := readScript.Run(ctx, s.client, []string{data, leases}, now.UnixMilli()).StringSlice()
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(values))
	for _, value := range values {
		var record Record
		if err := common.Unmarshal([]byte(value), &record); err != nil {
			return nil, fmt.Errorf("invalid active request record")
		}
		if record.UserID == userID {
			entries = append(entries, record.Entry)
		}
	}
	return entries, nil
}
