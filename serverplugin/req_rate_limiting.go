package serverplugin

import (
	"context"
	"time"

	"github.com/derekAHua/irpc/protocol"
	"github.com/derekAHua/irpc/server"
	"github.com/juju/ratelimit"
)

// ReqRateLimitingPlugin can limit requests per unit time
type ReqRateLimitingPlugin struct {
	FillInterval time.Duration
	Capacity     int64
	bucket       *ratelimit.Bucket
	block        bool // blocks or return error if reach the limit
}

// NewReqRateLimitingPlugin creates a new RateLimitingPlugin
func NewReqRateLimitingPlugin(fillInterval time.Duration, capacity int64, block bool) *ReqRateLimitingPlugin {
	tb := ratelimit.NewBucket(fillInterval, capacity)

	return &ReqRateLimitingPlugin{
		FillInterval: fillInterval,
		Capacity:     capacity,
		bucket:       tb,
		block:        block,
	}
}

// PostReadRequest can limit request processing.
func (plugin *ReqRateLimitingPlugin) PostReadRequest(_ context.Context, _ *protocol.Message, _ error) error {
	if plugin.block {
		plugin.bucket.Wait(1)
		return nil
	}

	count := plugin.bucket.TakeAvailable(1)
	if count == 1 {
		return nil
	}
	return server.ErrReqReachLimit
}
