package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"github.com/derekAHua/irpc/log"
	"github.com/derekAHua/irpc/protocol"
	"github.com/derekAHua/irpc/share"
	"github.com/soheilhy/cmux"
	"golang.org/x/net/websocket"
	"io"
	"net"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// Serve starts and listens RPC requests.
// It is blocking until receiving connections from clients.
func (s *Server) Serve(network, address string) (err error) {
	var ln net.Listener
	ln, err = s.makeListener(network, address)
	if err != nil {
		return
	}

	switch network {
	case "http", "ws", "wss":
		s.ln = ln
		rpcPath := share.DefaultRPCPath
		mux := http.NewServeMux()

		if network == "http" {
			mux.Handle(rpcPath, s)
		} else {
			mux.Handle(rpcPath, websocket.Handler(s.ServeWS))
		}

		srv := &http.Server{Handler: mux}
		err = srv.Serve(ln)
	default:
		// try to start gateway
		ln = s.startGateway(network, ln)
		err = s.ServeListener(ln)
	}

	return
}

// ServeHTTP implements http.Handler that answers RPC requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodConnect {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, "405 must CONNECT\n")
		return
	}

	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Info("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	_, _ = io.WriteString(conn, "HTTP/1.0 "+connected+"\n\n")

	s.setActiveConn(conn)
	s.serveConn(conn)
}

func (s *Server) ServeWS(conn *websocket.Conn) {
	s.setActiveConn(conn)
	conn.PayloadType = websocket.BinaryFrame
	s.serveConn(conn)
}

// ServeListener accepts incoming connections on the Listener ln,
// creating a new service goroutine for each.
// The service goroutines read requests and then call services to reply to them.
func (s *Server) ServeListener(ln net.Listener) (err error) {
	s.mu.Lock()
	s.ln = ln
	s.mu.Unlock()

	var (
		tempDelay time.Duration
		conn      net.Conn
		ok        bool
		tc        *net.TCPConn
		ne        net.Error
	)

	const max = time.Second

	for {
		conn, err = ln.Accept()
		if err != nil {
			if s.isShutdown() {
				<-s.doneChan
				return ErrServerClosed
			}

			// if error is temporary, delay then continue .
			if ne, ok = err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if tempDelay > max {
					tempDelay = max
				}

				log.Errorf("irpc: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}

			if errors.Is(err, cmux.ErrListenerClosed) {
				return ErrServerClosed
			}
			return err
		}

		tempDelay = 0 // reset delay time

		if tc, ok = conn.(*net.TCPConn); ok {
			if period := s.options["TCPKeepAlivePeriod"]; period != nil {
				_ = tc.SetKeepAlive(true)
				_ = tc.SetKeepAlivePeriod(period.(time.Duration))
				_ = tc.SetLinger(10)
			}
		}

		if conn, ok = s.Plugins.DoPostConnAccept(conn); !ok {
			_ = conn.Close()
			continue
		}

		if tlsConn, ok := conn.(*tls.Conn); ok {
			if d := s.readTimeout; d != 0 {
				_ = conn.SetReadDeadline(time.Now().Add(d))
			}
			if d := s.writeTimeout; d != 0 {
				_ = conn.SetWriteDeadline(time.Now().Add(d))
			}
			if err = tlsConn.Handshake(); err != nil {
				log.Errorf("irpc: TLS handshake error from %s: %v", conn.RemoteAddr(), err)
				return
			}
		}

		s.setActiveConn(conn)

		if share.Trace {
			log.Debugf("server accepted an conn: %v", conn.RemoteAddr().String())
		}

		go s.serveConn(conn)
	}
}

func (s *Server) serveConn(conn net.Conn) {
	if s.isShutdown() {
		s.closeConn(conn)
		return
	}

	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			ss := runtime.Stack(buf, false)
			if ss > size {
				ss = size
			}
			buf = buf[:ss]

			log.Errorf("serving %s panic error: %s, stack:\n %s", conn.RemoteAddr(), err, buf)
		}

		if share.Trace {
			log.Debugf("server closed conn: %v", conn.RemoteAddr().String())
		}

		// make sure all inflight requests are handled and all drained
		if s.isShutdown() {
			<-s.doneChan
		}

		s.closeConn(conn)
	}()

	r := bufio.NewReaderSize(conn, ReaderBuffSize)

	var writeCh chan *[]byte
	if s.AsyncWrite {
		writeCh = make(chan *[]byte, 1)
		defer close(writeCh)
		go s.serveAsyncWrite(conn, writeCh)
	}

	for {
		if s.isShutdown() {
			return
		}

		t0 := time.Now()
		if s.readTimeout != 0 {
			_ = conn.SetReadDeadline(t0.Add(s.readTimeout))
		}

		ctx := share.WithValue(context.Background(), RemoteConnContextKey, conn)

		req, err := s.readRequest(ctx, r)
		if err != nil {
			switch err {
			case io.EOF:
				log.Infof("client has closed this connection: %s", conn.RemoteAddr().String())
			case net.ErrClosed:
				log.Infof("irpc: connection %s is closed", conn.RemoteAddr().String())
			case ErrReqReachLimit:
				s.handleError(ctx, conn, writeCh, req, err)
				continue
			default:
				log.Warnf("irpc: failed to read request: %v", err)
			}
			protocol.FreeMsg(req)
			return
		}

		if share.Trace {
			log.Debugf("server received an request %+v from conn: %v", req, conn.RemoteAddr().String())
		}

		ctx.SetValue(StartRequestContextKey, time.Now().UnixNano())
		authFail := false
		if !req.IsHeartbeat() {
			err = s.auth(ctx, req)
			authFail = err != nil
		}

		if err != nil {
			s.handleError(ctx, conn, writeCh, req, err)
			if authFail {
				log.Infof("auth failed for conn %s: %v", conn.RemoteAddr().String(), err)
				return
			}
			continue
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					// maybe panic because the writeCh is closed.
					log.Errorf("[panic] failed to handle request: %v", r)
				}
			}()

			atomic.AddInt32(&s.handlerMsgNum, 1)
			defer atomic.AddInt32(&s.handlerMsgNum, -1)

			if req.IsHeartbeat() {
				// reuse request as response
				_ = s.Plugins.DoHeartbeatRequest(ctx, req)
				req.SetMessageType(protocol.Response)
				s.writeResponse(conn, writeCh, req)
				protocol.FreeMsg(req)
				return
			}

			resMetadata := make(map[string]string)
			ctx.SetValue(share.ReqMetaDataKey, req.Metadata)
			ctx.SetValue(share.ResMetaDataKey, resMetadata)

			cancelFunc := parseServerTimeout(ctx, req)
			if cancelFunc != nil {
				defer cancelFunc()
			}

			_ = s.Plugins.DoPreHandleRequest(ctx, req)

			if share.Trace {
				log.Debugf("server handle request %+v from conn: %v", req, conn.RemoteAddr().String())
			}

			// first use handler
			if handler, ok := s.router[req.ServicePath+"."+req.ServiceMethod]; ok {
				sCtx := NewContext(ctx, conn, req, writeCh)
				err := handler(sCtx)
				if err != nil {
					log.Errorf("[handler internal error]: servicePath: %s, serviceMethod, err: %v", req.ServicePath, req.ServiceMethod, err)
				}

				protocol.FreeMsg(req)
				return
			}

			var res *protocol.Message
			res, err = s.handleRequest(ctx, req)
			if err != nil {
				if s.HandleServiceError != nil {
					s.HandleServiceError(err)
				} else {
					log.Warnf("irpc: failed to handle request: %v", err)
				}
			}

			if !req.IsOneway() {
				if len(resMetadata) > 0 { // copy meta in context to request
					meta := res.Metadata
					if meta == nil {
						res.Metadata = resMetadata
					} else {
						for k, v := range resMetadata {
							if meta[k] == "" {
								meta[k] = v
							}
						}
					}
				}

				s.sendResponse(ctx, conn, writeCh, err, req, res)
			}

			if share.Trace {
				log.Debugf("server write response %+v for an request %+v from conn: %v", res, req, conn.RemoteAddr().String())
			}

			protocol.FreeMsg(req)
			protocol.FreeMsg(res)
		}()
	}
}

func (s *Server) handleError(ctx *share.Context, conn net.Conn, writeCh chan *[]byte, req *protocol.Message, err error) {
	if !req.IsOneway() {
		res := req.Clone()
		res.SetMessageType(protocol.Response)

		res.HandleError(err)
		s.sendResponse(ctx, conn, writeCh, err, req, res)
		protocol.FreeMsg(res)
	} else {
		_ = s.Plugins.DoPreWriteResponse(ctx, req, nil, err)
	}
	protocol.FreeMsg(req)
}

func (s *Server) sendResponse(ctx *share.Context, conn net.Conn, writeCh chan *[]byte, err error, req, res *protocol.Message) {
	if len(res.Payload) > 1024 && req.CompressType() != protocol.None {
		res.SetCompressType(req.CompressType())
	}

	_ = s.Plugins.DoPreWriteResponse(ctx, req, res, err)
	s.writeResponse(conn, writeCh, res)
	_ = s.Plugins.DoPostWriteResponse(ctx, req, res, err)
}

func (s *Server) writeResponse(conn net.Conn, writeCh chan *[]byte, res *protocol.Message) {
	data := res.EncodeSlicePointer()
	if s.AsyncWrite {
		writeCh <- data
	} else {
		if s.writeTimeout != 0 {
			_ = conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
		}
		_, _ = conn.Write(*data)
		protocol.PutData(data)
	}
}

func (s *Server) serveAsyncWrite(conn net.Conn, writeCh chan *[]byte) {
	for {
		select {
		case <-s.doneChan:
			return
		case data := <-writeCh:
			if data == nil {
				return
			}
			if s.writeTimeout != 0 {
				_ = conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
			}
			_, _ = conn.Write(*data)
			protocol.PutData(data)
		}
	}
}
