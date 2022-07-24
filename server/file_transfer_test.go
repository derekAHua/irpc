package server

import (
	"reflect"
	"testing"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 10:05
// @Version 1.0

func TestNewFileTransfer(t *testing.T) {
	type args struct {
		addr                string
		handler             FileTransferHandler
		downloadFileHandler DownloadFileHandler
		waitNum             int
	}
	tests := []struct {
		name string
		args args
		want *FileTransfer
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewFileTransfer(tt.args.addr, tt.args.handler, tt.args.downloadFileHandler, tt.args.waitNum); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFileTransfer() = %v, want %v", got, tt.want)
			}
		})
	}
}
