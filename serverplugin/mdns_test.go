package serverplugin

import (
	"github.com/rcrowley/go-metrics"
	"reflect"
	"testing"
	"time"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 10:02
// @Version 1.0

func TestNewMDNSRegisterPlugin(t *testing.T) {
	type args struct {
		serviceAddress string
		port           int
		m              metrics.Registry
		updateInterval time.Duration
		domain         string
	}
	tests := []struct {
		name string
		args args
		want *MDNSRegisterPlugin
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMDNSRegisterPlugin(tt.args.serviceAddress, tt.args.port, tt.args.m, tt.args.updateInterval, tt.args.domain); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMDNSRegisterPlugin() = %v, want %v", got, tt.want)
			}
		})
	}
}
