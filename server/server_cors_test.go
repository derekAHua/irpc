package server

import (
	"reflect"
	"testing"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 10:07
// @Version 1.0

func TestAllowAllCORSOptions(t *testing.T) {
	tests := []struct {
		name string
		want *CORSOptions
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AllowAllCORSOptions(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AllowAllCORSOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
