package testutils

import "testing"

// @Author: Derek
// @Description:
// @Date: 2022/7/24 10:08
// @Version 1.0

func Test_encodeFixed64ArithService(t *testing.T) {
	type args struct {
		dAtA   []byte
		offset int
		v      uint64
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
			if got := encodeFixed64ArithService(tt.args.dAtA, tt.args.offset, tt.args.v); got != tt.want {
				t.Errorf("encodeFixed64ArithService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sozArithService(t *testing.T) {
	type args struct {
		x uint64
	}
	tests := []struct {
		name  string
		args  args
		wantN int
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotN := sozArithService(tt.args.x); gotN != tt.wantN {
				t.Errorf("sozArithService() = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}

func Test_encodeFixed32ArithService(t *testing.T) {
	type args struct {
		dAtA   []byte
		offset int
		v      uint32
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
			if got := encodeFixed32ArithService(tt.args.dAtA, tt.args.offset, tt.args.v); got != tt.want {
				t.Errorf("encodeFixed32ArithService() = %v, want %v", got, tt.want)
			}
		})
	}
}
