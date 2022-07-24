package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/derekAHua/irpc/log"
	"github.com/derekAHua/irpc/protocol"
	"github.com/derekAHua/irpc/share"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"github.com/soheilhy/cmux"
)

func (s *Server) startGateway(network string, ln net.Listener) (irpcLn net.Listener) {
	switch network {
	case "tcp", "tcp4", "tcp6", "reuseport":
	default:
		log.Debugf("network is not tcp/tcp4/tcp6 so can not start gateway")
		return ln
	}

	m := cmux.New(ln)

	irpcLn = m.Match(irpcPrefixByteMatcher())

	if s.Plugins != nil {
		s.Plugins.MuxMatch(m)
	}

	if !s.DisableJSONRPC {
		jsonrpc2Ln := m.Match(cmux.HTTP1HeaderField("X-JSONRPC-2.0", "true"))
		go s.startJSONRPC2(jsonrpc2Ln)
	}

	if !s.DisableHTTPGateway {
		httpLn := m.Match(cmux.HTTP1Fast())
		go s.startHTTP1APIGateway(httpLn)
	}

	go func() {
		_ = m.Serve()
	}()

	return
}

func (s *Server) startHTTP1APIGateway(ln net.Listener) {
	router := httprouter.New()
	router.POST("/*servicePath", s.handleGatewayRequest)
	router.GET("/*servicePath", s.handleGatewayRequest)
	router.PUT("/*servicePath", s.handleGatewayRequest)

	if s.corsOptions != nil {
		opt := cors.Options(*s.corsOptions)
		c := cors.New(opt)
		mux := c.Handler(router)
		s.mu.Lock()
		s.gatewayHTTPServer = &http.Server{Handler: mux}
		s.mu.Unlock()
	} else {
		s.mu.Lock()
		s.gatewayHTTPServer = &http.Server{Handler: router}
		s.mu.Unlock()
	}

	if err := s.gatewayHTTPServer.Serve(ln); err != nil {
		if err == ErrServerClosed || errors.Is(err, cmux.ErrListenerClosed) {
			log.Info("gateway server closed")
		} else {
			log.Errorf("error in gateway Serve: %T %s", err, err)
		}
	}
}

func (s *Server) handleGatewayRequest(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	ctx := share.WithValue(r.Context(), RemoteConnContextKey, r.RemoteAddr) // notice: It is a string, different with TCP (net.Conn)
	err := s.Plugins.DoPreReadRequest(ctx)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if r.Header.Get(XServicePath) == "" {
		servicePath := params.ByName("servicePath")
		servicePath = strings.TrimPrefix(servicePath, "/")
		r.Header.Set(XServicePath, servicePath)
	}

	servicePath := r.Header.Get(XServicePath)
	wh := w.Header()
	req, err := HTTPRequest2IrpcRequest(r)
	defer protocol.FreeMsg(req)

	// set headers
	wh.Set(XVersion, r.Header.Get(XVersion))
	wh.Set(XMessageID, r.Header.Get(XMessageID))

	if err == nil && servicePath == "" {
		err = errors.New("empty servicePath")
	} else {
		wh.Set(XServicePath, servicePath)
	}

	if err == nil && r.Header.Get(XServiceMethod) == "" {
		err = errors.New("empty serviceMethod")
	} else {
		wh.Set(XServiceMethod, r.Header.Get(XServiceMethod))
	}

	if err == nil && r.Header.Get(XSerializeType) == "" {
		err = errors.New("empty serialized type")
	} else {
		wh.Set(XSerializeType, r.Header.Get(XSerializeType))
	}

	if err != nil {
		rh := r.Header
		for k, v := range rh {
			if strings.HasPrefix(k, "X-IRPC-") && len(v) > 0 {
				wh.Set(k, v[0])
			}
		}

		wh.Set(XMessageStatusType, "Error")
		wh.Set(XErrorMessage, err.Error())
		return
	}

	err = s.Plugins.DoPostReadRequest(ctx, req, nil)
	if err != nil {
		_ = s.Plugins.DoPreWriteResponse(ctx, req, nil, err)
		http.Error(w, err.Error(), 500)
		_ = s.Plugins.DoPostWriteResponse(ctx, req, req.Clone(), err)
		return
	}

	ctx.SetValue(StartRequestContextKey, time.Now().UnixNano())
	err = s.auth(ctx, req)
	if err != nil {
		_ = s.Plugins.DoPreWriteResponse(ctx, req, nil, err)
		wh.Set(XMessageStatusType, "Error")
		wh.Set(XErrorMessage, err.Error())
		w.WriteHeader(401)
		_ = s.Plugins.DoPostWriteResponse(ctx, req, req.Clone(), err)
		return
	}

	resMetadata := make(map[string]string)
	ctx.SetValue(share.ReqMetaDataKey, req.Metadata)
	ctx.SetValue(share.ResMetaDataKey, resMetadata)

	res, err := s.handleRequest(ctx, req)
	defer protocol.FreeMsg(res)

	if err != nil {
		_ = s.Plugins.DoPreWriteResponse(ctx, req, nil, err)
		if s.HandleServiceError != nil {
			s.HandleServiceError(err)
		} else {
			log.Warnf("irpc:  gateway request: %v", err)
		}
		wh.Set(XMessageStatusType, "Error")
		wh.Set(XErrorMessage, err.Error())
		w.WriteHeader(500)
		_ = s.Plugins.DoPostWriteResponse(ctx, req, req.Clone(), err)
		return
	}

	// will set res to call
	_ = s.Plugins.DoPreWriteResponse(ctx, req, res, nil)
	if len(resMetadata) > 0 { // copy meta in context to request
		meta := res.Metadata
		if meta == nil {
			res.Metadata = resMetadata
		} else {
			for k, v := range resMetadata {
				meta[k] = v
			}
		}
	}

	meta := url.Values{}
	for k, v := range res.Metadata {
		meta.Add(k, v)
	}
	wh.Set(XMeta, meta.Encode())
	_, _ = w.Write(res.Payload)
	_ = s.Plugins.DoPostWriteResponse(ctx, req, res, err)
}

func (s *Server) closeHTTP1APIGateway(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gatewayHTTPServer != nil {
		return s.gatewayHTTPServer.Shutdown(ctx)
	}

	return nil
}

func irpcPrefixByteMatcher() cmux.Matcher {
	magic := protocol.MagicNumber()
	return func(r io.Reader) bool {
		buf := make([]byte, 1)
		n, _ := r.Read(buf)
		return n == 1 && buf[0] == magic
	}
}
