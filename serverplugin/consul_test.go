package serverplugin

import (
	"testing"
	"time"

	"github.com/derekAHua/irpc/server"
	metrics "github.com/rcrowley/go-metrics"
)

func TestConsulRegistry(t *testing.T) {
	s := server.New()

	r := &ConsulRegisterPlugin{
		ServiceAddress: "tcp@127.0.0.1:8972",
		ConsulServers:  []string{"127.0.0.1:8500"},
		BasePath:       "/irpc_test",
		Metrics:        metrics.NewRegistry(),
		UpdateInterval: time.Minute,
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
		t.Fatal("failed to register services in consul")
	}

	if err := r.Stop(); err != nil {
		t.Fatal(err)
	}
}
