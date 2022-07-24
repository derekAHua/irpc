package main

import (
	"context"
	"github.com/derekAHua/irpc/client"
	"github.com/derekAHua/irpc/server"
	"log"
	"testing"
)

// @Author: Derek
// @Description:
// @Date: 2022/5/14 08:39
// @Version 1.0

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

func (t *Arith) Mul2(_ context.Context, args *Args, reply *Reply) error {
	reply.C = args.A*args.B + 100
	return nil
}

const addr = "localhost:8972"

func TestServer(t *testing.T) {
	s := server.New()
	_ = s.RegisterName("Arith", new(Arith), "")
	_ = s.Serve("tcp", ":8972")
}

func TestClient(t *testing.T) {
	cli := client.NewClient(client.DefaultOption)

	err := cli.Connect("tcp", addr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer func() { _ = cli.Close() }()

	args := &Args{
		A: 10,
		B: 20,
	}

	reply := &Reply{}
	err = cli.Call(context.Background(), "Arith", "Mul", args, reply)
	if err != nil {
		t.Fatalf("failed to call: %v", err)
	}

	if reply.C != 200 {
		t.Fatalf("expect 200 but got %d", reply.C)
	}

	t.Logf("%d * %d = %d", args.A, args.B, reply.C)
}

func TestXClient(t *testing.T) {
	// #1
	d, err := client.NewPeer2PeerDiscovery("tcp@"+addr, "")
	if err != nil {
		t.Error(err)
		return
	}

	// #2
	option := client.DefaultOption
	option.Retries = 0
	xClient := client.NewXClient("Arith", client.Failtry, client.RandomSelect, d, option)
	defer func() { _ = xClient.Close() }()

	// #3
	args := &Args{
		A: 10,
		B: 20,
	}

	// #4
	reply := &Reply{}

	// #5
	err = xClient.Call(context.Background(), "Mul", args, reply)
	if err != nil {
		t.Error(err)
		return
	}

	t.Logf("%d * %d = %d", args.A, args.B, reply.C)
}

func TestClientASync(t *testing.T) {
	d, err := client.NewPeer2PeerDiscovery("tcp@"+addr, "")
	if err != nil {
		t.Error(err)
		return
	}
	xClient := client.NewXClient("Arith", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer func() { _ = xClient.Close() }()

	args := &Args{
		A: 10,
		B: 20,
	}

	reply := &Reply{}
	call, err := xClient.Go(context.Background(), "Mul", args, reply, nil)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	replyCall := <-call.Done
	if replyCall.Error != nil {
		log.Fatalf("failed to call: %v", replyCall.Error)
	} else {
		log.Printf("%d * %d = %d", args.A, args.B, reply.C)
	}
}

func TestMultiServers(t *testing.T) {
	d, err := client.NewMultipleServersDiscovery([]*client.KVPair{{Key: addr}, {Key: "localhost:8973"}})
	if err != nil {
		t.Error(err)
		return
	}
	xClient := client.NewXClient("Arith", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer xClient.Close()
}

func TestConsul(t *testing.T) {
	d, err := client.NewConsulDiscovery("base", "Arith", []string{""}, nil)
	if err != nil {
		t.Error(err)
		return
	}
	xClient := client.NewXClient("Arith", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer xClient.Close()

	args := &Args{
		A: 10,
		B: 20,
	}

	reply := &Reply{}
	call, err := xClient.Go(context.Background(), "Mul", args, reply, nil)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	replyCall := <-call.Done
	if replyCall.Error != nil {
		log.Fatalf("failed to call: %v", replyCall.Error)
	} else {
		log.Printf("%d * %d = %d", args.A, args.B, reply.C)
	}
}
