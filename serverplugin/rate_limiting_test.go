package serverplugin

import (
	"testing"
	"time"
)

// @Author: Derek
// @Description:
// @Date: 2022/7/24 09:48
// @Version 1.0

func TestNewRateLimitingPlugin(t *testing.T) {
	NewRateLimitingPlugin(time.Second, 100)
}
