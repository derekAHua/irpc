package client

import (
	"reflect"
	"testing"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 10:03
// @Version 1.0

func TestNewPluginContainer(t *testing.T) {
	tests := []struct {
		name string
		want PluginContainer
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewPluginContainer(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPluginContainer() = %v, want %v", got, tt.want)
			}
		})
	}
}
