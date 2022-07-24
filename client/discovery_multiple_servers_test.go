package client

import (
	"reflect"
	"testing"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 10:15
// @Version 1.0

func TestNewMultipleServersDiscovery(t *testing.T) {
	type args struct {
		pairs []*KVPair
	}
	tests := []struct {
		name    string
		args    args
		want    *MultipleServersDiscovery
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMultipleServersDiscovery(tt.args.pairs)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMultipleServersDiscovery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMultipleServersDiscovery() got = %v, want %v", got, tt.want)
			}
		})
	}
}
