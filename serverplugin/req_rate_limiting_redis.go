package serverplugin

import (
	"context"
	"time"

	"github.com/derekAHua/irpc/protocol"
	"github.com/derekAHua/irpc/server"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
)

var _ server.PostReadRequestPlugin = (*RedisRateLimitingPlugin)(nil)

// RedisRateLimitingPlugin can limit requests per unit time
type RedisRateLimitingPlugin struct {
	addresses []string
	limiter   redis_rate.Limiter
	limit     redis_rate.Limit
}

// NewRedisRateLimitingPlugin creates a new RateLimitingPlugin
func NewRedisRateLimitingPlugin(addresses []string, rate int, burst int, period time.Duration) *RedisRateLimitingPlugin {
	limit := redis_rate.Limit{
		Rate:   rate,
		Burst:  burst,
		Period: period,
	}
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: addresses,
	})

	limiter := redis_rate.NewLimiter(rdb)

	return &RedisRateLimitingPlugin{
		addresses: addresses,
		limiter:   *limiter,
		limit:     limit,
	}
}

// PostReadRequest can limit request processing.
func (plugin *RedisRateLimitingPlugin) PostReadRequest(ctx context.Context, r *protocol.Message, _ error) error {
	res, err := plugin.limiter.Allow(ctx, r.ServicePath+"/"+r.ServiceMethod, plugin.limit)
	if err != nil {
		return err
	}

	if res.Allowed > 0 {
		return nil
	}
	return server.ErrReqReachLimit
}
