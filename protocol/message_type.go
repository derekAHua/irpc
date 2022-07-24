package protocol

type (
	// MessageType is message type of requests and responses.
	MessageType byte

	// MessageStatusType is status of messages.
	MessageStatusType byte

	// CompressType defines decompression type.
	CompressType byte

	// SerializeType defines serialization type of payload.
	SerializeType byte
)

// MessageType's constant.
const (
	// Request is message type of request
	Request MessageType = iota
	// Response is message type of response
	Response
)

// MessageStatusType's constant.
const (
	// Normal is normal requests and responses.
	Normal MessageStatusType = iota
	// Error indicates some errors occur.
	Error
)

// CompressType's constant.
const (
	// None does not compress.
	None CompressType = iota
	// Gzip uses gzip compression.
	Gzip
)

// SerializeType's constant.
const (
	// SerializeNone uses raw []byte and don't serialize/deserialize
	SerializeNone SerializeType = iota

	// JSON for payload.
	JSON

	// ProtoBuffer for payload.
	ProtoBuffer

	// MsgPack for payload and heartbeat.
	MsgPack

	// Thrift for payload.
	Thrift
)
