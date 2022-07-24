package serverplugin

import (
	"reflect"
	"testing"
	"time"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 09:54
// @Version 1.0

func TestNewRedisRateLimitingPlugin(t *testing.T) {
	type args struct {
		addresses []string
		rate      int
		burst     int
		period    time.Duration
	}
	tests := []struct {
		name string
		args args
		want *RedisRateLimitingPlugin
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRedisRateLimitingPlugin(tt.args.addresses, tt.args.rate, tt.args.burst, tt.args.period); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRedisRateLimitingPlugin() = %v, want %v", got, tt.want)
			}
		})
	}
}
