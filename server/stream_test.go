package server

import (
	"reflect"
	"testing"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 10:03
// @Version 1.0

func TestNewStreamService(t *testing.T) {
	type args struct {
		addr          string
		streamHandler StreamHandler
		in2           StreamAcceptor
		waitNum       int
	}
	tests := []struct {
		name string
		args args
		want *StreamService
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewStreamService(tt.args.addr, tt.args.streamHandler, tt.args.in2, tt.args.waitNum); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewStreamService() = %v, want %v", got, tt.want)
			}
		})
	}
}
