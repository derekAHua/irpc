package serverplugin

import (
	"context"
	"testing"
	"time"

	"github.com/derekAHua/irpc/server"
	metrics "github.com/rcrowley/go-metrics"
)

type Args struct {
	A int
	B int
}

type Reply struct {
	C int
}

type Arith int

func (t *Arith) Mul(_ context.Context, args *Args, reply *Reply) error {
	reply.C = args.A * args.B
	return nil
}

func TestZookeeperRegistry(t *testing.T) {
	s := server.New()

	r := &ZooKeeperRegisterPlugin{
		ServiceAddress:   "tcp@127.0.0.1:8972",
		ZooKeeperServers: []string{"127.0.0.1:2181"},
		BasePath:         "/irpc_test",
		Metrics:          metrics.NewRegistry(),
		UpdateInterval:   time.Minute,
	}
	err := r.Start()
	if err != nil {
		return
	}
	s.Plugins.Add(r)

	_ = s.RegisterName("Arith", new(Arith), "")
	go func() { _ = s.Serve("tcp", "127.0.0.1:8972") }()
	defer func() { _ = s.Close() }()

	if len(r.Services) != 1 {
		t.Fatal("failed to register services in zookeeper")
	}

	if err := r.Stop(); err != nil {
		t.Fatal(err)
	}
}
