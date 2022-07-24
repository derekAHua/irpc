package client

import (
	"context"
	"github.com/derekAHua/irpc/protocol"
	"net"
)

type (
	// Plugin is the client plugin interface.
	Plugin interface{}

	// PreCallPlugin is invoked before the client calls a server.
	PreCallPlugin interface {
		PreCall(ctx context.Context, servicePath, serviceMethod string, args interface{}) error
	}

	// PostCallPlugin is invoked after the client calls a server.
	PostCallPlugin interface {
		PostCall(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}, err error) error
	}

	// ConnCreatedPlugin is invoked when the client connection has created.
	ConnCreatedPlugin interface {
		ConnCreated(net.Conn) (net.Conn, error)
	}

	ConnCreateFailedPlugin interface {
		ConnCreateFailed(network, address string)
	}

	// ConnectedPlugin is invoked when the client has connected the server.
	ConnectedPlugin interface {
		ClientConnected(net.Conn) (net.Conn, error)
	}

	// ConnectionClosePlugin is invoked when the connection is closing.
	ConnectionClosePlugin interface {
		ClientConnectionClose(net.Conn) error
	}

	// BeforeEncodePlugin is invoked when the message is encoded and sent.
	BeforeEncodePlugin interface {
		ClientBeforeEncode(*protocol.Message) error
	}

	// AfterDecodePlugin is invoked when the message is decoded.
	AfterDecodePlugin interface {
		ClientAfterDecode(*protocol.Message) error
	}

	// SelectNodePlugin can interrupt selecting of xclient and add customized logics such as skipping some nodes.
	SelectNodePlugin interface {
		WrapSelect(SelectFunc) SelectFunc
	}
)
