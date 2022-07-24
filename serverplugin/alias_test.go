package serverplugin

import (
	"reflect"
	"testing"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 10:02
// @Version 1.0

func TestNewAliasPlugin(t *testing.T) {
	tests := []struct {
		name string
		want *AliasPlugin
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAliasPlugin(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAliasPlugin() = %v, want %v", got, tt.want)
			}
		})
	}
}
