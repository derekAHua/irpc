package client

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/akutz/memconn"
	"github.com/derekAHua/irpc/log"
	"github.com/derekAHua/irpc/share"
	"golang.org/x/net/websocket"
	"io"
	"net"
	"net/http"
	"time"
)

// ReaderBuffsize is used for bufio reader.
const ReaderBuffsize = 16 * 1024

var connected = "200 Connected to irpc"

type ConnFactoryFn func(c *Client, network, address string) (net.Conn, error)

var ConnFactories = map[string]ConnFactoryFn{
	"http": newDirectHTTPConn,
	"kcp":  newDirectKCPConn,
	"quic": newDirectQuicConn,
	"unix": newDirectConn,
	"memu": newMemuConn,
}

// Connect connects the server via specified network.
func (client *Client) Connect(network, address string) (err error) {
	var conn net.Conn

	switch network {
	case "http":
		conn, err = newDirectHTTPConn(client, network, address)
	case "ws", "wss":
		conn, err = newDirectWSConn(client, network, address)
	default:
		if fn, ok := ConnFactories[network]; ok {
			conn, err = fn(client, network, address)
			break
		}

		conn, err = newDirectConn(client, network, address)
	}

	if err == nil && conn != nil {
		if tc, ok := conn.(*net.TCPConn); ok && client.option.TCPKeepAlivePeriod > 0 {
			_ = tc.SetKeepAlive(true)
			_ = tc.SetKeepAlivePeriod(client.option.TCPKeepAlivePeriod)
		}

		if client.option.IdleTimeout != 0 {
			_ = conn.SetDeadline(time.Now().Add(client.option.IdleTimeout))
		}

		if client.Plugins != nil {
			conn, err = client.Plugins.DoConnCreated(conn)
			if err != nil {
				return err
			}
		}

		client.Conn = conn
		client.r = bufio.NewReaderSize(conn, ReaderBuffsize)
		// c.w = bufio.NewWriterSize(conn, WriterBuffsize)

		// start reading and writing since connected
		go client.input()

		if client.option.Heartbeat && client.option.HeartbeatInterval > 0 {
			go client.heartbeat()
		}

	}

	if err != nil && client.Plugins != nil {
		client.Plugins.DoConnCreateFailed(network, address)
	}

	return err
}

func newDirectConn(c *Client, network, address string) (conn net.Conn, err error) {
	var tlsConn *tls.Conn

	if c == nil {
		err = fmt.Errorf("nil client")
		return
	}

	if c.option.TLSConfig != nil {
		dialer := &net.Dialer{
			Timeout: c.option.ConnectTimeout,
		}
		tlsConn, err = tls.DialWithDialer(dialer, network, address, c.option.TLSConfig)
		// or conn:= tls.Client(netConn, &config)
		conn = net.Conn(tlsConn)
	} else {
		conn, err = net.DialTimeout(network, address, c.option.ConnectTimeout)
	}

	if err != nil {
		log.Warnf("failed to dial server: %v", err)
		return
	}

	return
}

func newDirectHTTPConn(c *Client, network, address string) (conn net.Conn, err error) {
	if c == nil {
		return nil, errors.New("empty client")
	}

	path := c.option.RPCPath
	if path == "" {
		path = share.DefaultRPCPath
	}

	var tlsConn *tls.Conn

	// get conn.
	if c.option.TLSConfig != nil {
		dialer := &net.Dialer{
			Timeout: c.option.ConnectTimeout,
		}
		tlsConn, err = tls.DialWithDialer(dialer, "tcp", address, c.option.TLSConfig)
		// or conn:= tls.Client(netConn, &config)
		conn = net.Conn(tlsConn)
	} else {
		conn, err = net.DialTimeout("tcp", address, c.option.ConnectTimeout)
	}
	if err != nil {
		log.Errorf("failed to dial server: %v", err)
		return
	}

	_, err = io.WriteString(conn, "CONNECT "+path+" HTTP/1.0\n\n")
	if err != nil {
		// Dial() success but Write() failed here, close the successfully created conn before return.
		_ = conn.Close()

		log.Errorf("failed to make CONNECT: %v", err)
		return
	}

	// Require successful HTTP response
	// before switching to RPC protocol
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == connected {
		return
	}

	if err == nil {
		log.Errorf("unexpected HTTP response: %v", err)
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}

	_ = conn.Close()

	return nil, &net.OpError{
		Op:   "dial-http",
		Net:  network + " " + address,
		Addr: nil,
		Err:  err,
	}
}

func newDirectWSConn(c *Client, network, address string) (conn net.Conn, err error) {
	if c == nil {
		return nil, errors.New("empty client")
	}
	path := c.option.RPCPath
	if path == "" {
		path = share.DefaultRPCPath
	}

	// url := "ws://localhost:12345/ws"
	var url, origin string
	switch network {
	case "ws":
		url = fmt.Sprintf("ws://%s%s", address, path)
		origin = fmt.Sprintf("http://%s", address)
	default:
		url = fmt.Sprintf("wss://%s%s", address, path)
		origin = fmt.Sprintf("https://%s", address)
	}

	if c.option.TLSConfig != nil {
		config, errN := websocket.NewConfig(url, origin)
		if errN != nil {
			return nil, errN
		}

		config.TlsConfig = c.option.TLSConfig
		conn, err = websocket.DialConfig(config)
	} else {
		conn, err = websocket.Dial(url, "", origin)
	}

	return
}

func newDirectKCPConn(_ *Client, _, _ string) (net.Conn, error) {
	return nil, errors.New("kcp unsupported")
}

func newDirectQuicConn(_ *Client, _, _ string) (net.Conn, error) {
	return nil, errors.New("quic unsupported")
}

func newMemuConn(_ *Client, network, address string) (net.Conn, error) {
	return memconn.Dial(network, address)
}
