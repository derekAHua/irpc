package client

import (
	"crypto/tls"
	"github.com/derekAHua/irpc/protocol"
	"github.com/derekAHua/irpc/share"
	"time"
)

// Option contains all options for creating clients.
type Option struct {
	// Group is used to select the services in the same group. Services set group info in their meta.
	// If it is empty, clients will ignore group.
	Group string

	// Retries retries to send
	Retries int

	// TLSConfig for tcp and quic
	TLSConfig *tls.Config
	// kcp.BlockCrypt
	Block interface{}
	// RPCPath for http connection
	RPCPath string
	// ConnectTimeout sets timeout for dialing
	ConnectTimeout time.Duration
	// IdleTimeout sets max idle time for underlying connections.
	IdleTimeout time.Duration

	// BackupLatency is used for Failbackup mode.
	// irpc will send another request if the first response doesn't return in BackupLatency time.
	BackupLatency time.Duration

	// Breaker is used to config CircuitBreaker
	GenBreaker func() Breaker

	SerializeType protocol.SerializeType
	CompressType  protocol.CompressType

	// send heartbeat message to service and check responses
	Heartbeat bool
	// interval for heartbeat
	HeartbeatInterval   time.Duration
	MaxWaitForHeartbeat time.Duration // default 30 * time.Second

	// TCPKeepAlivePeriod
	// If it is zero we don't set keepalive.
	TCPKeepAlivePeriod time.Duration
	// bidirectional mode, if true serverMessageChan will block to wait message for consume. default false.
	BidirectionalBlock bool
}

// DefaultOption is a common option configuration for client.
var DefaultOption = Option{
	Retries:             3,
	RPCPath:             share.DefaultRPCPath,
	ConnectTimeout:      time.Second,
	SerializeType:       protocol.MsgPack,
	CompressType:        protocol.None,
	BackupLatency:       10 * time.Millisecond,
	MaxWaitForHeartbeat: 30 * time.Second,
	TCPKeepAlivePeriod:  time.Minute,
	BidirectionalBlock:  false,
}
