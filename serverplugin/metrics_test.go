package serverplugin

import (
	"github.com/rcrowley/go-metrics"
	"reflect"
	"testing"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 09:58
// @Version 1.0

func TestNewMetricsPlugin(t *testing.T) {
	type args struct {
		registry metrics.Registry
	}
	tests := []struct {
		name string
		args args
		want *MetricsPlugin
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMetricsPlugin(tt.args.registry); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMetricsPlugin() = %v, want %v", got, tt.want)
			}
		})
	}
}
