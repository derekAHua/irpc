package server

import (
	"context"
	"encoding/json"
	"github.com/rs/cors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/derekAHua/irpc/protocol"
	"github.com/derekAHua/irpc/share"
)

func (s *Server) startJSONRPC2(ln net.Listener) {
	newServer := http.NewServeMux()
	newServer.HandleFunc("/", s.jsonrpcHandler)

	srv := http.Server{ConnContext: func(ctx context.Context, c net.Conn) context.Context {
		return context.WithValue(ctx, HttpConnContextKey, c)
	}}

	if s.corsOptions != nil {
		opt := cors.Options(*s.corsOptions)
		c := cors.New(opt)
		mux := c.Handler(newServer)
		srv.Handler = mux

		go func() { _ = srv.Serve(ln) }()

		return
	}

	srv.Handler = newServer
	go func() { _ = srv.Serve(ln) }()
}

func (s *Server) jsonrpcHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req = &jsonrpcRequest{}
	err = json.Unmarshal(data, req)
	if err != nil {
		var res = &jsonrpcResponse{}
		res.Error = &JSONRPCError{
			Code:    CodeParseJSONRPCError,
			Message: err.Error(),
		}

		writeResponse(w, res)
		return
	}

	conn := r.Context().Value(HttpConnContextKey).(net.Conn)

	ctx := share.WithValue(r.Context(), RemoteConnContextKey, conn)
	if req.ID != nil {
		res := s.handleJSONRPCRequest(ctx, req, r.Header)
		writeResponse(w, res)
		return
	}

	// notification
	go s.handleJSONRPCRequest(ctx, req, r.Header)
}

func writeResponse(w http.ResponseWriter, res *jsonrpcResponse) {
	data, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Context-Type", "application/json")
	_, _ = w.Write(data)
}

func (s *Server) handleJSONRPCRequest(ctx context.Context, r *jsonrpcRequest, header http.Header) (res *jsonrpcResponse) {
	_ = s.Plugins.DoPreReadRequest(ctx)

	res.ID = r.ID

	req := protocol.GetPooledMsg()
	if req.Metadata == nil {
		req.Metadata = make(map[string]string)
	}

	if r.ID == nil {
		req.SetOneway(true)
	}
	req.SetMessageType(protocol.Request)
	req.SetSerializeType(protocol.JSON)

	lastDot := strings.LastIndex(r.Method, ".")
	if lastDot <= 0 {
		res.Error = &JSONRPCError{
			Code:    CodeMethodNotFound,
			Message: "must contains servicePath and method",
		}
		return res
	}
	req.ServicePath = r.Method[:lastDot]
	req.ServiceMethod = r.Method[lastDot+1:]
	req.Payload = *r.Params

	// meta
	meta := header.Get(XMeta)
	if meta != "" {
		metadata, _ := url.ParseQuery(meta)
		for k, v := range metadata {
			if len(v) > 0 {
				req.Metadata[k] = v[0]
			}
		}
	}

	auth := header.Get("Authorization")
	if auth != "" {
		req.Metadata[share.AuthKey] = auth
	}

	err := s.Plugins.DoPostReadRequest(ctx, req, nil)
	if err != nil {
		res.Error = &JSONRPCError{
			Code:    CodeInternalJSONRPCError,
			Message: err.Error(),
		}
		return res
	}

	err = s.auth(ctx, req)
	if err != nil {
		_ = s.Plugins.DoPreWriteResponse(ctx, req, nil, err)
		res.Error = &JSONRPCError{
			Code:    CodeInternalJSONRPCError,
			Message: err.Error(),
		}
		_ = s.Plugins.DoPostWriteResponse(ctx, req, req.Clone(), err)
		return res
	}

	resp, err := s.handleRequest(ctx, req)
	if r.ID == nil {
		return nil
	}

	_ = s.Plugins.DoPreWriteResponse(ctx, req, nil, err)
	if err != nil {
		res.Error = &JSONRPCError{
			Code:    CodeInternalJSONRPCError,
			Message: err.Error(),
		}
		_ = s.Plugins.DoPostWriteResponse(ctx, req, req.Clone(), err)
		return res
	}

	result := json.RawMessage(resp.Payload)
	res.Result = &result
	_ = s.Plugins.DoPostWriteResponse(ctx, req, req.Clone(), err)
	return res
}
