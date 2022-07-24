//go:build !windows
// +build !windows

package server

import (
	"net"

	reuseport "github.com/kavu/go_reuseport"
)

func init() {
	makeListeners["reuseport"] = reuseportMakeListener
	makeListeners["unix"] = unixMakeListener
}

func reuseportMakeListener(_ *Server, address string) (ln net.Listener, err error) {
	var network string
	if validIP6(address) {
		network = "tcp6"
	} else {
		network = "tcp4"
	}

	return reuseport.NewReusablePortListener(network, address)
}

func unixMakeListener(_ *Server, address string) (ln net.Listener, err error) {
	addr, err := net.ResolveUnixAddr("unix", address)
	if err != nil {
		return nil, err
	}
	return net.ListenUnix("unix", addr)
}
