package client

import "testing"

// @Author: Derek
// @Description:
// @Date: 2022/7/24 10:03
// @Version 1.0

func TestJumpConsistentHash(t *testing.T) {
	type args struct {
		len     int
		options []interface{}
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JumpConsistentHash(tt.args.len, tt.args.options...); got != tt.want {
				t.Errorf("JumpConsistentHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
