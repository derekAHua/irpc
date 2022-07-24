package server

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/derekAHua/irpc/protocol"
	"github.com/derekAHua/irpc/share"
)

const (
	XVersion           = "X-IRPC-Version"
	XMessageType       = "X-IRPC-MessageType"
	XHeartbeat         = "X-IRPC-Heartbeat"
	XOneway            = "X-IRPC-Oneway"
	XMessageStatusType = "X-IRPC-MessageStatusType"
	XSerializeType     = "X-IRPC-SerializeType"
	XMessageID         = "X-IRPC-MessageID"
	XServicePath       = "X-IRPC-ServicePath"
	XServiceMethod     = "X-IRPC-ServiceMethod"
	XMeta              = "X-IRPC-Meta"
	XErrorMessage      = "X-IRPC-ErrorMessage"
)

// HTTPRequest2IrpcRequest converts a http request to a irpc request.
func HTTPRequest2IrpcRequest(r *http.Request) (*protocol.Message, error) {
	req := protocol.GetPooledMsg()
	req.SetMessageType(protocol.Request)

	h := r.Header
	seq := h.Get(XMessageID)
	if seq != "" {
		id, err := strconv.ParseUint(seq, 10, 64)
		if err != nil {
			return nil, err
		}
		req.SetSeq(id)
	}

	heartbeat := h.Get(XHeartbeat)
	if heartbeat != "" {
		req.SetHeartbeat(true)
	}

	oneway := h.Get(XOneway)
	if oneway != "" {
		req.SetOneway(true)
	}

	st := h.Get(XSerializeType)
	if st != "" {
		rst, err := strconv.Atoi(st)
		if err != nil {
			return nil, err
		}
		req.SetSerializeType(protocol.SerializeType(rst))
	}

	meta := h.Get(XMeta)
	if meta != "" {
		metadata, err := url.ParseQuery(meta)
		if err != nil {
			return nil, err
		}
		mm := make(map[string]string)
		for k, v := range metadata {
			if len(v) > 0 {
				mm[k] = v[0]
			}
		}
		req.Metadata = mm
	}

	auth := h.Get("Authorization")
	if auth != "" {
		if req.Metadata == nil {
			req.Metadata = make(map[string]string)
		}
		req.Metadata[share.AuthKey] = auth
	}

	req.ServicePath = h.Get(XServicePath)

	req.ServiceMethod = h.Get(XServiceMethod)

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	req.Payload = payload

	return req, nil
}
