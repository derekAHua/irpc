package server

import "testing"

// @Author: Derek
// @Description:
// @Date: 2022/7/24 09:53
// @Version 1.0

func TestRegisterMakeListener(t *testing.T) {
	type args struct {
		network string
		ml      MakeListener
	}
	tests := []struct {
		name string
		args args
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func TestRegisterMakeListener1(t *testing.T) {
	RegisterMakeListener("", nil)
}
