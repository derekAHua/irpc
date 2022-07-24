package log

import "testing"

// @Author: Derek
// @Description:
// @Date: 2022/7/24 09:56
// @Version 1.0

func TestSetLogger(t *testing.T) {
	SetLogger(nil)
	SetDummyLogger()
}
