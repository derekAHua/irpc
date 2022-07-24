package serverplugin

import (
	"reflect"
	"testing"
	"time"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 09:53
// @Version 1.0

func TestNewReqRateLimitingPlugin(t *testing.T) {
	type args struct {
		fillInterval time.Duration
		capacity     int64
		block        bool
	}
	tests := []struct {
		name string
		args args
		want *ReqRateLimitingPlugin
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewReqRateLimitingPlugin(tt.args.fillInterval, tt.args.capacity, tt.args.block); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewReqRateLimitingPlugin() = %v, want %v", got, tt.want)
			}
		})
	}
}
