package client

import (
	"reflect"
	"testing"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 10:23
// @Version 1.0

func TestNewClient(t *testing.T) {
	type args struct {
		option Option
	}
	tests := []struct {
		name string
		args args
		want *Client
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClient(tt.args.option); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClient() = %v, want %v", got, tt.want)
			}
		})
	}
}
