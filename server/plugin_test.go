package server

import (
	"context"
	"github.com/derekAHua/irpc/client"
	"github.com/derekAHua/irpc/protocol"
	"net"
	"sync"
	"testing"
	"time"
)

type HeartbeatHandler struct{}

func (h *HeartbeatHandler) HeartbeatRequest(ctx context.Context, _ *protocol.Message) error {
	conn := ctx.Value(RemoteConnContextKey).(net.Conn)
	println("OnHeartbeat:", conn.RemoteAddr().String())
	return nil
}

// TestPluginHeartbeat: go test -v -test.run TestPluginHeartbeat
func TestPluginHeartbeat(t *testing.T) {
	h := &HeartbeatHandler{}
	s := New(
		WithReadTimeout(time.Duration(5)*time.Second),
		WithWriteTimeout(time.Duration(5)*time.Second),
	)
	s.Plugins.Add(h)
	_ = s.RegisterName("Arith", new(Arith), "")

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		// server
		defer wg.Done()
		err := s.Serve("tcp", "127.0.0.1:9001")
		if err != nil {
			t.Log(err.Error())
		}
	}()
	go func() {
		// wait for server start complete
		time.Sleep(time.Second)
		defer wg.Done()
		// client
		opts := client.DefaultOption
		opts.Heartbeat = true
		opts.HeartbeatInterval = time.Second / 5
		opts.IdleTimeout = time.Duration(5) * time.Second
		opts.ConnectTimeout = time.Duration(5) * time.Second
		// PeerDiscovery
		d, err := client.NewPeer2PeerDiscovery("tcp@127.0.0.1:9001", "")
		if err != nil {
			t.Errorf("failed to NewPeer2PeerDiscovery: %v", err)
			return
		}

		c := client.NewXClient("Arith", client.Failtry, client.RoundRobin, d, opts)
		i := 0
		for {
			i++
			resp := &Reply{}
			_ = c.Call(context.Background(), "Mul", &Args{A: 1, B: 5}, resp)
			t.Log(i, " call Mul resp:", resp.C)
			time.Sleep(time.Second)
			if i == 10 {
				break
			}
		}
		_ = c.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	}()

	wg.Wait()
}
