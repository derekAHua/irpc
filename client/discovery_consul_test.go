package client

import (
	"github.com/rpcxio/libkv/store"
	"reflect"
	"testing"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 10:06
// @Version 1.0

func TestNewConsulDiscoveryTemplate(t *testing.T) {
	type args struct {
		basePath   string
		consulAddr []string
		options    *store.Config
	}
	tests := []struct {
		name    string
		args    args
		want    *ConsulDiscovery
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConsulDiscoveryTemplate(tt.args.basePath, tt.args.consulAddr, tt.args.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConsulDiscoveryTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConsulDiscoveryTemplate() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewConsulDiscovery(t *testing.T) {
	type args struct {
		basePath    string
		servicePath string
		consulAddr  []string
		options     *store.Config
	}
	tests := []struct {
		name    string
		args    args
		want    *ConsulDiscovery
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConsulDiscovery(tt.args.basePath, tt.args.servicePath, tt.args.consulAddr, tt.args.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConsulDiscovery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConsulDiscovery() got = %v, want %v", got, tt.want)
			}
		})
	}
}
