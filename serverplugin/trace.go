package serverplugin

import (
	"context"
	"net"
	"reflect"
	"runtime"

	"github.com/derekAHua/irpc/protocol"
	"golang.org/x/net/trace"
)

type TracePlugin struct {
}

func (p *TracePlugin) Register(name string, receiver interface{}, _ string) error {
	tr := trace.New("irpc.Server", "Register")
	defer tr.Finish()
	tr.LazyPrintf("register %s: %T", name, receiver)
	return nil
}

func (p *TracePlugin) RegisterFunction(serviceName, fName string, fn interface{}, _ string) error {
	tr := trace.New("irpc.Server", "RegisterFunction")
	defer tr.Finish()
	tr.LazyPrintf("register %s.%s: %T", serviceName, fName, GetFunctionName(fn))
	return nil
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (p *TracePlugin) PostConnAccept(conn net.Conn) (net.Conn, bool) {
	tr := trace.New("irpc.Server", "Accept")
	defer tr.Finish()
	tr.LazyPrintf("accept conn %s", conn.RemoteAddr().String())
	return conn, true
}

func (p *TracePlugin) PostReadRequest(_ context.Context, r *protocol.Message, _ error) error {
	tr := trace.New("irpc.Server", "ReadRequest")
	defer tr.Finish()
	tr.LazyPrintf("read request %s.%s, seq: %d", r.ServicePath, r.ServiceMethod, r.Seq())
	return nil
}

func (p *TracePlugin) PostWriteResponse(_ context.Context, req *protocol.Message, _ *protocol.Message, err error) error {
	tr := trace.New("irpc.Server", "WriteResponse")
	defer tr.Finish()
	if err == nil {
		tr.LazyPrintf("succeed to call %s.%s, seq: %d", req.ServicePath, req.ServiceMethod, req.Seq())
	} else {
		tr.LazyPrintf("failed to call %s.%s, seq: %d : %v", req.Seq, req.ServicePath, req.ServiceMethod, req.Seq(), err)
		tr.SetError()
	}

	return nil
}
