package server

import (
	"context"
	"net"

	"github.com/derekAHua/irpc/protocol"
	"github.com/soheilhy/cmux"
)

type (
	// Plugin is the server plugin interface.
	Plugin interface{}

	// RegisterPlugin is .
	RegisterPlugin interface {
		Register(name string, rcvr interface{}, metadata string) error
		Unregister(name string) error
	}

	// RegisterFunctionPlugin is .
	RegisterFunctionPlugin interface {
		RegisterFunction(serviceName, fname string, fn interface{}, metadata string) error
	}

	// PostConnAcceptPlugin represents connection accept plugin.
	// if returns false, it means subsequent IPostConnAcceptPlugins should not continue to handle this connection
	// and this connection has been closed.
	PostConnAcceptPlugin interface {
		HandleConnAccept(net.Conn) (net.Conn, bool)
	}

	// PostConnClosePlugin represents client connection close plugin.
	PostConnClosePlugin interface {
		HandleConnClose(net.Conn) bool
	}

	// PreReadRequestPlugin represents .
	PreReadRequestPlugin interface {
		PreReadRequest(ctx context.Context) error
	}

	// PostReadRequestPlugin represents .
	PostReadRequestPlugin interface {
		PostReadRequest(ctx context.Context, r *protocol.Message, e error) error
	}

	// PreHandleRequestPlugin represents .
	PreHandleRequestPlugin interface {
		PreHandleRequest(ctx context.Context, r *protocol.Message) error
	}

	PreCallPlugin interface {
		PreCall(ctx context.Context, serviceName, methodName string, args interface{}) (interface{}, error)
	}

	PostCallPlugin interface {
		PostCall(ctx context.Context, serviceName, methodName string, args, reply interface{}) (interface{}, error)
	}

	// PreWriteResponsePlugin represents .
	PreWriteResponsePlugin interface {
		PreWriteResponse(context.Context, *protocol.Message, *protocol.Message, error) error
	}

	// PostWriteResponsePlugin represents .
	PostWriteResponsePlugin interface {
		PostWriteResponse(context.Context, *protocol.Message, *protocol.Message, error) error
	}

	// PreWriteRequestPlugin represents .
	PreWriteRequestPlugin interface {
		PreWriteRequest(ctx context.Context) error
	}

	// PostWriteRequestPlugin represents .
	PostWriteRequestPlugin interface {
		PostWriteRequest(ctx context.Context, r *protocol.Message, e error) error
	}

	// HeartbeatPlugin is .
	HeartbeatPlugin interface {
		HeartbeatRequest(ctx context.Context, req *protocol.Message) error
	}

	CMuxPlugin interface {
		MuxMatch(m cmux.CMux)
	}
)
