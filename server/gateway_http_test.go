package server

import (
	"bytes"
	"github.com/derekAHua/irpc/codec"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
)

func TestGatewayHttp_test(t *testing.T) {
	cc := &codec.MsgpackCodec{}

	// request
	args := &Args{
		A: 10,
		B: 20,
	}
	data, _ := cc.Encode(args)

	req, err := http.NewRequest("POST", "http://127.0.0.1:8973/", bytes.NewReader(data))
	if err != nil {
		log.Fatal("failed to create request: ", err)
		return
	}

	// set extra headers
	h := req.Header
	h.Set(XMessageID, "10000")
	h.Set(XMessageType, "0")
	h.Set(XSerializeType, "3")
	h.Set(XServicePath, "Arith")
	h.Set(XServiceMethod, "Mul")

	// send to gateway
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("failed to call: ", err)
	}
	defer func() { _ = res.Body.Close() }()

	// handle http response
	replyData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("failed to read response: ", err)
	}

	// parse reply
	reply := &Reply{}
	err = cc.Decode(replyData, reply)
	if err != nil {
		log.Fatal("failed to decode reply: ", err)
	}

	log.Printf("%d * %d = %d", args.A, args.B, reply.C)
}
