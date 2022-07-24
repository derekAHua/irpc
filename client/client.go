package client

import (
	"bufio"
	"bytes"
	"context"
	"github.com/derekAHua/irpc/log"
	"github.com/derekAHua/irpc/protocol"
	"github.com/derekAHua/irpc/share"
	"io"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// RPCClient is interface that defines one client to call one server.
type RPCClient interface {
	Connect(network, address string) error
	Go(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}, done chan *Call) *Call
	Call(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error
	SendRaw(ctx context.Context, r *protocol.Message) (map[string]string, []byte, error)
	Close() error
	RemoteAddr() string

	RegisterServerMessageChan(ch chan<- *protocol.Message)
	UnregisterServerMessageChan()

	IsClosing() bool
	IsShutdown() bool

	GetConn() net.Conn
}

// Client represents a RPCClient.
type (
	Client struct {
		option Option

		Conn net.Conn
		r    *bufio.Reader

		mutex        sync.Mutex // protects following
		seq          uint64
		pending      map[uint64]*Call
		closing      bool // user has called Close
		shutdown     bool // server has told us to stop
		pluginClosed bool // the plugin has been called

		Plugins PluginContainer

		ServerMessageChan chan<- *protocol.Message
	}

	// Call represents an active RPC.
	Call struct {
		ServicePath   string            // The name of the service and method to call
		ServiceMethod string            // The name of the service and method to call
		Metadata      map[string]string // metadata
		ResMetadata   map[string]string // metadata of response
		Args          interface{}       // The argument to the function (*struct)
		Reply         interface{}       // The reply from the function (*struct)
		Error         error             // After completion, the error status
		Done          chan *Call        // Strobes when call is complete
		Raw           bool              // raw message or not
	}
)

// input wait for server's messages.
func (client *Client) input() {
	var err error

	for err == nil {
		res := protocol.NewMessage()
		if client.option.IdleTimeout != 0 {
			_ = client.Conn.SetDeadline(time.Now().Add(client.option.IdleTimeout))
		}

		err = res.Decode(client.r)
		if err != nil {
			break
		}
		if client.Plugins != nil {
			_ = client.Plugins.DoClientAfterDecode(res)
		}

		seq := res.Seq()
		var call *Call
		isServerMessage := res.MessageType() == protocol.Request && !res.IsHeartbeat() && res.IsOneway()
		if !isServerMessage {
			client.mutex.Lock()
			call = client.pending[seq]
			delete(client.pending, seq)
			client.mutex.Unlock()
		}

		if share.Trace {
			log.Debugf("client.input received %v", res)
		}

		switch {
		case call == nil:
			if isServerMessage {
				if client.ServerMessageChan != nil {
					client.handleServerRequest(res)
				}
				continue
			}
		case res.MessageStatusType() == protocol.Error:
			// We've got an error response. Give this to the request.
			if len(res.Metadata) > 0 {
				call.ResMetadata = res.Metadata
				call.Error = ServiceError(res.Metadata[protocol.ServiceError])
			}

			if call.Raw {
				call.Metadata, call.Reply, _ = convertRes2Raw(res)
				call.Metadata[XErrorMessage] = call.Error.Error()
			} else if len(res.Payload) > 0 {
				data := res.Payload
				codec := share.Codecs[res.SerializeType()]
				if codec != nil {
					_ = codec.Decode(data, call.Reply)
				}
			}
			call.done()
		default:
			if call.Raw {
				call.Metadata, call.Reply, _ = convertRes2Raw(res)
			} else {
				data := res.Payload
				if len(data) > 0 {
					codec := share.Codecs[res.SerializeType()]
					if codec == nil {
						call.Error = ServiceError(ErrUnsupportedCodec.Error())
					} else {
						err = codec.Decode(data, call.Reply)
						if err != nil {
							call.Error = ServiceError(err.Error())
						}
					}
				}
				if len(res.Metadata) > 0 {
					call.ResMetadata = res.Metadata
				}

			}

			call.done()
		}
	}

	if client.ServerMessageChan != nil {
		req := protocol.NewMessage()
		req.SetMessageType(protocol.Request)
		req.SetMessageStatusType(protocol.Error)
		if req.Metadata == nil {
			req.Metadata = make(map[string]string)
			if err != nil {
				req.Metadata[protocol.ServiceError] = err.Error()
			}
		}
		req.Metadata["server"] = client.Conn.RemoteAddr().String()
		client.handleServerRequest(req)
	}

	client.mutex.Lock()

	if !client.pluginClosed {
		if client.Plugins != nil {
			_ = client.Plugins.DoClientConnectionClose(client.Conn)
		}
		client.pluginClosed = true
	}
	_ = client.Conn.Close()

	client.shutdown = true
	closing := client.closing
	if err == io.EOF {
		if closing {
			err = ErrShutdown
		} else {
			err = io.ErrUnexpectedEOF
		}
	}
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}

	client.mutex.Unlock()

	if err != nil && !closing {
		log.Errorf("irpc: client protocol error: %v", err)
	}
}

func (client *Client) handleServerRequest(msg *protocol.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("ServerMessageChan may be closed so client remove it. Please add it again if you want to handle server requests. error is %v", r)
			client.ServerMessageChan = nil
		}
	}()

	serverMessageChan := client.ServerMessageChan
	if serverMessageChan != nil {
		if client.option.BidirectionalBlock {
			serverMessageChan <- msg
		} else {
			select {
			case serverMessageChan <- msg:
			default:
				log.Warnf("ServerMessageChan may be full so the server request %d has been dropped", msg.Seq())
			}
		}
	}
}

func (client *Client) heartbeat() {
	t := time.NewTicker(client.option.HeartbeatInterval)

	if client.option.MaxWaitForHeartbeat == 0 {
		client.option.MaxWaitForHeartbeat = 30 * time.Second
	}

	for range t.C {
		if client.IsShutdown() || client.IsClosing() {
			t.Stop()
			return
		}

		request := time.Now().UnixNano()
		reply := int64(0)
		ctx, cancel := context.WithTimeout(context.Background(), client.option.MaxWaitForHeartbeat)
		err := client.Call(ctx, "", "", &request, &reply)
		abnormal := false
		if ctx.Err() != nil {
			log.Warnf("failed to heartbeat to %s, context err: %v", client.Conn.RemoteAddr().String(), ctx.Err())
			abnormal = true
		}
		cancel()
		if err != nil {
			log.Warnf("failed to heartbeat to %s: %v", client.Conn.RemoteAddr().String(), err)
			abnormal = true
		}

		if reply != request {
			log.Warnf("reply %d in heartbeat to %s is different from request %d", reply, client.Conn.RemoteAddr().String(), request)
		}

		if abnormal {
			_ = client.Close()
		}
	}
}

// IsShutdown client is shutdown or not.
func (client *Client) IsShutdown() bool {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	return client.shutdown
}

// IsClosing client is closing or not.
func (client *Client) IsClosing() bool {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	return client.closing
}

// Close calls the underlying connection's Close method. If the connection is already
// shutting down, ErrShutdown is returned.
func (client *Client) Close() (err error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	for seq, call := range client.pending {
		delete(client.pending, seq)
		if call != nil {
			call.Error = ErrShutdown
			call.done()
		}
	}

	if !client.pluginClosed {
		if client.Plugins != nil {
			_ = client.Plugins.DoClientConnectionClose(client.Conn)
		}

		client.pluginClosed = true
		err = client.Conn.Close()
	}

	if client.closing || client.shutdown {
		client.mutex.Unlock()
		return ErrShutdown
	}

	client.closing = true

	return
}

// RemoteAddr returns the remote address.
func (client *Client) RemoteAddr() string {
	return client.Conn.RemoteAddr().String()
}

// GetConn returns the underlying conn.
func (client *Client) GetConn() net.Conn {
	return client.Conn
}

// RegisterServerMessageChan registers the channel that receives server requests.
func (client *Client) RegisterServerMessageChan(ch chan<- *protocol.Message) {
	client.ServerMessageChan = ch
}

// UnregisterServerMessageChan removes ServerMessageChan.
func (client *Client) UnregisterServerMessageChan() {
	client.ServerMessageChan = nil
}

// NewClient returns a new Client with the option.
func NewClient(option Option) *Client {
	return &Client{
		option: option,
	}
}

func convertRes2Raw(res *protocol.Message) (map[string]string, []byte, error) {
	m := make(map[string]string)
	m[XVersion] = strconv.Itoa(int(res.Version()))
	if res.IsHeartbeat() {
		m[XHeartbeat] = "true"
	}
	if res.IsOneway() {
		m[XOneway] = "true"
	}
	if res.MessageStatusType() == protocol.Error {
		m[XMessageStatusType] = "Error"
	} else {
		m[XMessageStatusType] = "Normal"
	}

	m[XMeta] = urlEncode(res.Metadata)
	m[XSerializeType] = strconv.Itoa(int(res.SerializeType()))
	m[XMessageID] = strconv.FormatUint(res.Seq(), 10)
	m[XServicePath] = res.ServicePath
	m[XServiceMethod] = res.ServiceMethod

	return m, res.Payload, nil
}

func urlEncode(data map[string]string) string {
	if len(data) == 0 {
		return ""
	}
	var buf bytes.Buffer
	for k, v := range data {
		buf.WriteString(url.QueryEscape(k))
		buf.WriteByte('=')
		buf.WriteString(url.QueryEscape(v))
		buf.WriteByte('&')
	}
	s := buf.String()
	return s[0 : len(s)-1]
}

func (call *Call) done() {
	select {
	case call.Done <- call:
		// ok
	default:
		log.Debug("irpc: discarding Call reply due to insufficient Done chan capacity")
	}
}
