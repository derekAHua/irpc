package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/derekAHua/irpc/codec"
	"github.com/derekAHua/irpc/log"
	"github.com/derekAHua/irpc/protocol"
	"github.com/derekAHua/irpc/share"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var connected = "200 Connected to irpc"

var (
	// ErrServerClosed is returned by the Server.ServeListener after a call Server.Shutdown or Server.Close.
	ErrServerClosed  = errors.New("ServeListener: Server closed")
	ErrReqReachLimit = errors.New("request reached rate limit")
)

const (
	// ReaderBuffSize is used for bufio.Reader.
	ReaderBuffSize = 1024

	//// WriteChanSize is used for response.
	//WriteChanSize = 1024 * 1024
)

type Handler func(ctx *Context) error

// Server is the irpc server that use TCP or UDP.
type Server struct {
	ln           net.Listener
	readTimeout  time.Duration
	writeTimeout time.Duration

	gatewayHTTPServer  *http.Server
	DisableHTTPGateway bool // should disable http invoke or not.
	DisableJSONRPC     bool // should disable json rpc or not.
	AsyncWrite         bool // set true if your server only serves few clients

	serviceMapMu sync.RWMutex
	serviceMap   map[string]*service

	router map[string]Handler

	mu         sync.RWMutex
	activeConn map[net.Conn]struct{}
	doneChan   chan struct{}
	seq        uint64

	inShutdown int32
	onShutdown []func(s *Server)
	onRestart  []func(s *Server)

	// tlsConfig for creating tls tcp connection
	tlsConfig *tls.Config
	// BlockCrypt for kcp.BlockCrypt
	options map[string]interface{}

	// CORSOptions
	corsOptions *CORSOptions

	Plugins PluginContainer

	// AuthFunc can be used to auth
	AuthFunc func(ctx context.Context, req *protocol.Message, token string) error

	handlerMsgNum int32

	HandleServiceError func(error)
}

// New returns a server.
func New(options ...Option) *Server {
	s := &Server{
		Plugins:    &pluginContainer{},
		options:    make(map[string]interface{}),
		activeConn: make(map[net.Conn]struct{}),
		doneChan:   make(chan struct{}),
		serviceMap: make(map[string]*service),
		router:     make(map[string]Handler),
		AsyncWrite: false, // 除非你想做进一步优化测试，否则建议你设置为false
	}

	for _, op := range options {
		op(s)
	}

	if s.options["TCPKeepAlivePeriod"] == nil {
		s.options["TCPKeepAlivePeriod"] = 3 * time.Minute
	}
	return s
}

func (s *Server) AddHandler(servicePath, serviceMethod string, handler func(*Context) error) {
	s.router[servicePath+"."+serviceMethod] = handler
}

func (s *Server) readRequest(ctx context.Context, r io.Reader) (req *protocol.Message, err error) {
	err = s.Plugins.DoPreReadRequest(ctx)
	if err != nil {
		return
	}

	req = protocol.GetPooledMsg()
	err = req.Decode(r)
	if err == io.EOF {
		return
	}

	pErr := s.Plugins.DoPostReadRequest(ctx, req, err)
	if err == nil {
		err = pErr
	}

	return
}

func (s *Server) auth(ctx context.Context, req *protocol.Message) (err error) {
	if s.AuthFunc != nil {
		token := req.Metadata[share.AuthKey]
		return s.AuthFunc(ctx, req, token)
	}

	return
}

func (s *Server) handleRequest(ctx context.Context, req *protocol.Message) (res *protocol.Message, err error) {
	defer res.HandleError(err)

	serviceName := req.ServicePath
	methodName := req.ServiceMethod

	res = req.Clone()
	res.SetMessageType(protocol.Response)

	service := s.getService(serviceName)
	if share.Trace {
		log.Debugf("server get service %+v for an request %+v", service, req)
	}
	if service == nil {
		res.HandleError(errors.New("irpc: can't find service " + serviceName))
		return
	}

	coder := share.Codecs[req.SerializeType()]
	if coder == nil {
		err = fmt.Errorf("can't find codec for %d", req.SerializeType())
		res.HandleError(err)
		return
	}

	mType := service.method[methodName]
	if mType == nil {
		if service.function[methodName] != nil {
			err = s.handleRequestForFunction(ctx, service, coder, req, res)
			return
		}
		res.HandleError(errors.New("irpc: can't find method " + methodName))
		return
	}

	// get a argv object from object pool.
	argv := reflectTypePools.Get(mType.ArgType)
	defer reflectTypePools.Put(mType.ArgType, argv)
	err = coder.Decode(req.Payload, argv)
	if err != nil {
		return
	}

	// get a reply object from object pool.
	reply := reflectTypePools.Get(mType.ReplyType)
	defer reflectTypePools.Put(mType.ReplyType, reply)

	argv, err = s.Plugins.DoPreCall(ctx, serviceName, methodName, argv)
	if err != nil {
		return
	}

	if mType.ArgType.Kind() != reflect.Ptr {
		err = service.call(ctx, mType, reflect.ValueOf(argv).Elem(), reflect.ValueOf(reply))
	} else {
		err = service.call(ctx, mType, reflect.ValueOf(argv), reflect.ValueOf(reply))
	}

	if err == nil {
		reply, err = s.Plugins.DoPostCall(ctx, serviceName, methodName, argv, reply)
	}

	if err != nil {
		if reply != nil {
			res.Payload, err = coder.Encode(reply)
		}
		return
	}

	if !req.IsOneway() {
		res.Payload, err = coder.Encode(reply)
		if err != nil {
			return
		}
	}

	if share.Trace {
		log.Debugf("server called service %+v for an request %+v", service, req)
	}

	return
}

func (s *Server) handleRequestForFunction(ctx context.Context, service *service, coder codec.Codec, req *protocol.Message, res *protocol.Message) (err error) {
	mType := service.function[req.ServiceMethod]
	if mType == nil {
		res.HandleError(errors.New("irpc: can't find method " + req.ServiceMethod))
		return
	}

	argv := reflectTypePools.Get(mType.ArgType)
	defer reflectTypePools.Put(mType.ArgType, argv)
	err = coder.Decode(req.Payload, argv)
	if err != nil {
		return
	}

	reply := reflectTypePools.Get(mType.ReplyType)
	defer reflectTypePools.Put(mType.ReplyType, reply)

	if mType.ArgType.Kind() != reflect.Ptr {
		err = service.callForFunction(ctx, mType, reflect.ValueOf(argv).Elem(), reflect.ValueOf(reply))
	} else {
		err = service.callForFunction(ctx, mType, reflect.ValueOf(argv), reflect.ValueOf(reply))
	}

	if err != nil {
		return
	}

	if !req.IsOneway() {
		res.Payload, err = coder.Encode(reply)
		if err != nil {
			return
		}
	}

	return
}

// ----------------------------------------------------------------------------------------------------------------

// Address returns listened address.
func (s *Server) Address() net.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ln == nil {
		return nil
	}
	return s.ln.Addr()
}

// ActiveClientConn returns active connections.
func (s *Server) ActiveClientConn() []net.Conn {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]net.Conn, 0, len(s.activeConn))
	for clientConn := range s.activeConn {
		result = append(result, clientConn)
	}
	return result
}

//SendMessage a request to the specified client.
//The client is designated by the conn.
//conn can be gotten from context in services:
//
//  ctx.Value(RemoteConnContextKey)
//
//servicePath, serviceMethod, metadata can be set to zero values.
func (s *Server) SendMessage(conn net.Conn, servicePath, serviceMethod string, metadata map[string]string, data []byte) error {
	ctx := share.WithValue(context.Background(), StartSendRequestContextKey, time.Now().UnixNano())
	_ = s.Plugins.DoPreWriteRequest(ctx)

	req := protocol.GetPooledMsg()
	req.SetMessageType(protocol.Request)

	seq := atomic.AddUint64(&s.seq, 1)
	req.SetSeq(seq)
	req.SetOneway(true)
	req.SetSerializeType(protocol.SerializeNone)
	req.ServicePath = servicePath
	req.ServiceMethod = serviceMethod
	req.Metadata = metadata
	req.Payload = data

	b := req.EncodeSlicePointer()
	_, err := conn.Write(*b)
	protocol.PutData(b)

	_ = s.Plugins.DoPostWriteRequest(ctx, req, err)
	protocol.FreeMsg(req)
	return err
}

func (s *Server) getDoneChan() <-chan struct{} {
	return s.doneChan
}

// Close immediately closes all active net.Listeners.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	if s.ln != nil {
		err = s.ln.Close()
	}
	for c := range s.activeConn {
		_ = c.Close()
		delete(s.activeConn, c)
		s.Plugins.DoPostConnClose(c)
	}
	s.closeDoneChanLocked()
	return err
}

// RegisterOnShutdown registers a function to call on Shutdown.
// This can be used to gracefully shutdown connections.
func (s *Server) RegisterOnShutdown(f func(s *Server)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onShutdown = append(s.onShutdown, f)
}

// RegisterOnRestart registers a function to call on Restart.
func (s *Server) RegisterOnRestart(f func(s *Server)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onRestart = append(s.onRestart, f)
}

var shutdownPollInterval = 1000 * time.Millisecond

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Shutdown works by first closing the
// listener, then closing all idle connections, and then waiting
// indefinitely for connections to return to idle and then shut down.
// If the provided context expires before the shutdown is complete,
// Shutdown returns the context's error, otherwise it returns any
// error returned from closing the Server's underlying Listener.
func (s *Server) Shutdown(ctx context.Context) (err error) {
	if atomic.CompareAndSwapInt32(&s.inShutdown, 0, 1) {
		log.Info("shutdown begin")

		s.mu.Lock()
		// 主动注销注册的服务
		if s.Plugins != nil {
			for name := range s.serviceMap {
				_ = s.Plugins.DoUnregister(name)
			}
		}

		_ = s.ln.Close()
		for conn := range s.activeConn {
			if tcpConn, ok := conn.(*net.TCPConn); ok {
				_ = tcpConn.CloseRead()
			}
		}
		s.mu.Unlock()

		// wait all in-processing requests finish.
		ticker := time.NewTicker(shutdownPollInterval)
		defer ticker.Stop()
	outer:
		for {
			if s.checkProcessMsg() {
				break
			}
			select {
			case <-ctx.Done():
				err = ctx.Err()
				break outer
			case <-ticker.C:
			}
		}

		if s.gatewayHTTPServer != nil {
			if err = s.closeHTTP1APIGateway(ctx); err != nil {
				log.Warnf("failed to close gateway: %v", err)
			} else {
				log.Info("closed gateway")
			}
		}

		s.mu.Lock()
		for conn := range s.activeConn {
			_ = conn.Close()
			delete(s.activeConn, conn)
			s.Plugins.DoPostConnClose(conn)
		}
		s.closeDoneChanLocked()
		s.mu.Unlock()

		log.Info("shutdown end")
	}

	return
}

// Restart restarts this server gracefully.
// It starts a new irpc server with the same port with SO_REUSEPORT socket option,
// and shutdown this irpc server gracefully.
func (s *Server) Restart(ctx context.Context) error {
	pid, err := s.startProcess()
	if err != nil {
		return err
	}
	log.Infof("restart a new irpc server: %d", pid)

	// TODO: is it necessary?
	time.Sleep(3 * time.Second)
	return s.Shutdown(ctx)
}

func (s *Server) startProcess() (int, error) {
	argv0, err := exec.LookPath(os.Args[0])
	if err != nil {
		return 0, err
	}

	// Pass on the environment and replace the old count key with the new one.
	var env []string
	env = append(env, os.Environ()...)

	originalWD, _ := os.Getwd()
	allFiles := []*os.File{os.Stdin, os.Stdout, os.Stderr}
	process, err := os.StartProcess(argv0, os.Args, &os.ProcAttr{
		Dir:   originalWD,
		Env:   env,
		Files: allFiles,
	})
	if err != nil {
		return 0, err
	}
	return process.Pid, nil
}

func (s *Server) checkProcessMsg() bool {
	size := atomic.LoadInt32(&s.handlerMsgNum)
	log.Info("need handle in-processing msg size:", size)
	return size == 0
}

func (s *Server) closeDoneChanLocked() {
	select {
	case <-s.doneChan:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by s.mu.RegisterName
		close(s.doneChan)
	}
}

var ip4Reg = regexp.MustCompile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)

func validIP4(ipAddress string) bool {
	ipAddress = strings.Trim(ipAddress, " ")
	i := strings.LastIndex(ipAddress, ":")
	ipAddress = ipAddress[:i] // remove port

	return ip4Reg.MatchString(ipAddress)
}

func validIP6(ipAddress string) bool {
	ipAddress = strings.Trim(ipAddress, " ")
	i := strings.LastIndex(ipAddress, ":")
	ipAddress = ipAddress[:i] // remove port
	ipAddress = strings.TrimPrefix(ipAddress, "[")
	ipAddress = strings.TrimSuffix(ipAddress, "]")
	ip := net.ParseIP(ipAddress)
	if ip != nil && ip.To4() == nil {
		return true
	} else {
		return false
	}
}
