package client

import (
	"context"
	"strings"
)

func splitNetworkAndAddress(server string) (string, string) {
	ss := strings.SplitN(server, "@", 2)
	if len(ss) == 1 {
		return "tcp", server
	}

	return ss[0], ss[1]
}

func contextCanceled(err error) bool {
	if err == context.DeadlineExceeded {
		return true
	}

	if err == context.Canceled {
		return true
	}

	return false
}
