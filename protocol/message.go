package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/derekAHua/irpc/util"
	"github.com/valyala/bytebufferpool"
)

// Compressors are compressors supported by irpc. You can add customized compressor in Compressors.
var Compressors = map[CompressType]Compressor{
	None: &RawDataCompressor{},
	Gzip: &GzipCompressor{},
}

// MaxMessageLength is the max length of a message.
// Default is 0 that means does not limit length of messages.
// It is used to validate when read messages from io.Reader.
var MaxMessageLength = 0

var (
	// ErrMetaKVMissing some keys or values are missing.
	ErrMetaKVMissing = errors.New("wrong metadata lines. some keys or values are missing")
	// ErrMessageTooLong message is too long
	ErrMessageTooLong = errors.New("message is too long")

	ErrUnsupportedCompressor = errors.New("unsupported compressor")
)

const (
	// ServiceError contains error info of service invocation
	ServiceError = "__irpc_error__"
)

// Message is the generic type of Request and Response.
type Message struct {
	*Header
	ServicePath   string
	ServiceMethod string
	Metadata      map[string]string
	Payload       []byte
	data          []byte
}

// Clone clones from a message.
func (m Message) Clone() *Message {
	header := *m.Header
	c := GetPooledMsg()
	header.SetCompressType(None)
	c.Header = &header
	c.ServicePath = m.ServicePath
	c.ServiceMethod = m.ServiceMethod
	return c
}

// Encode encodes messages.
func (m Message) Encode() []byte {
	data := m.EncodeSlicePointer()
	return *data
}

// EncodeSlicePointer encodes messages as a byte slice pointer we can use pool to improve.
func (m Message) EncodeSlicePointer() *[]byte {
	bb := bytebufferpool.Get()
	encodeMetadata(m.Metadata, bb)
	meta := bb.Bytes()

	spL := len(m.ServicePath)
	smL := len(m.ServiceMethod)

	var err error
	payload := m.Payload
	if m.CompressType() != None {
		compressor := Compressors[m.CompressType()]
		if compressor == nil {
			m.SetCompressType(None)
		} else {
			payload, err = compressor.Zip(m.Payload)
			if err != nil {
				m.SetCompressType(None)
				payload = m.Payload
			}
		}
	}

	totalL := (4 + spL) + (4 + smL) + (4 + len(meta)) + (4 + len(payload))

	metaStart := 12 + 4 + (4 + spL) + (4 + smL)
	payLoadStart := metaStart + (4 + len(meta))

	// header + dataLen + spLen + sp + smLen + sm + metaL + meta + payloadLen + payload
	l := 12 + 4 + totalL

	data := bufferPool.Get(l)

	// write header
	copy(*data, m.Header[:])

	// write totalLen
	binary.BigEndian.PutUint32((*data)[12:16], uint32(totalL))

	// write servicePath
	binary.BigEndian.PutUint32((*data)[16:20], uint32(spL))
	copy((*data)[20:20+spL], util.StringToSliceByte(m.ServicePath))

	// write serviceMethod
	binary.BigEndian.PutUint32((*data)[20+spL:24+spL], uint32(smL))
	copy((*data)[24+spL:metaStart], util.StringToSliceByte(m.ServiceMethod))

	// write meta
	binary.BigEndian.PutUint32((*data)[metaStart:metaStart+4], uint32(len(meta)))
	copy((*data)[metaStart+4:], meta)

	bytebufferpool.Put(bb)

	// write payload
	binary.BigEndian.PutUint32((*data)[payLoadStart:payLoadStart+4], uint32(len(payload)))
	copy((*data)[payLoadStart+4:], payload)

	return data
}

// WriteTo writes message to writers.
func (m Message) WriteTo(w io.Writer) (int64, error) {
	nn, err := w.Write(m.Header[:])
	n := int64(nn)
	if err != nil {
		return n, err
	}

	bb := bytebufferpool.Get()
	encodeMetadata(m.Metadata, bb)
	meta := bb.Bytes()

	spL := len(m.ServicePath)
	smL := len(m.ServiceMethod)

	payload := m.Payload
	if m.CompressType() != None {
		compressor := Compressors[m.CompressType()]
		if compressor == nil {
			return n, ErrUnsupportedCompressor
		}
		payload, err = compressor.Zip(m.Payload)
		if err != nil {
			return n, err
		}
	}

	totalL := (4 + spL) + (4 + smL) + (4 + len(meta)) + (4 + len(payload))
	err = binary.Write(w, binary.BigEndian, uint32(totalL))
	if err != nil {
		return n, err
	}

	// write servicePath and serviceMethod
	err = binary.Write(w, binary.BigEndian, uint32(len(m.ServicePath)))
	if err != nil {
		return n, err
	}
	_, err = w.Write(util.StringToSliceByte(m.ServicePath))
	if err != nil {
		return n, err
	}
	err = binary.Write(w, binary.BigEndian, uint32(len(m.ServiceMethod)))
	if err != nil {
		return n, err
	}
	_, err = w.Write(util.StringToSliceByte(m.ServiceMethod))
	if err != nil {
		return n, err
	}

	// write meta
	err = binary.Write(w, binary.BigEndian, uint32(len(meta)))
	if err != nil {
		return n, err
	}
	_, err = w.Write(meta)
	if err != nil {
		return n, err
	}

	bytebufferpool.Put(bb)

	// write payload
	err = binary.Write(w, binary.BigEndian, uint32(len(payload)))
	if err != nil {
		return n, err
	}

	nn, err = w.Write(payload)
	return int64(nn), err
}

// Decode decodes a message from a reader.
func (m *Message) Decode(r io.Reader) (err error) {
	// validate rest length for each step?

	// parse header
	_, err = io.ReadFull(r, m.Header[:1])
	if err != nil {
		return
	}
	// check the magic number of header
	if !m.Header.CheckMagicNumber() {
		return fmt.Errorf("wrong magic number: %v", m.Header[0])
	}
	_, err = io.ReadFull(r, m.Header[1:])
	if err != nil {
		return
	}

	// parse total
	lenData := poolUint32Data.Get().(*[]byte)
	_, err = io.ReadFull(r, *lenData)
	if err != nil {
		poolUint32Data.Put(lenData)
		return
	}
	l := binary.BigEndian.Uint32(*lenData)
	poolUint32Data.Put(lenData)
	// check message length
	if MaxMessageLength > 0 && int(l) > MaxMessageLength {
		return ErrMessageTooLong
	}

	// parse data
	totalL := int(l)
	if cap(m.data) >= totalL {
		m.data = m.data[:totalL]
	} else {
		m.data = make([]byte, totalL)
	}
	data := m.data
	_, err = io.ReadFull(r, data)
	if err != nil {
		return
	}

	n := 0

	// parse servicePath
	l = binary.BigEndian.Uint32(data[n:4])
	n = n + 4
	nEnd := n + int(l)
	m.ServicePath = util.SliceByteToString(data[n:nEnd])
	n = nEnd

	// parse serviceMethod
	l = binary.BigEndian.Uint32(data[n : n+4])
	n = n + 4
	nEnd = n + int(l)
	m.ServiceMethod = util.SliceByteToString(data[n:nEnd])
	n = nEnd

	// parse meta
	l = binary.BigEndian.Uint32(data[n : n+4])
	n = n + 4
	nEnd = n + int(l)
	if l > 0 {
		m.Metadata, err = decodeMetadata(l, data[n:nEnd])
		if err != nil {
			return
		}
	}
	n = nEnd

	// parse payload
	l = binary.BigEndian.Uint32(data[n : n+4])
	_ = l
	n = n + 4
	m.Payload = data[n:]
	if m.CompressType() != None {
		compressor := Compressors[m.CompressType()]
		if compressor == nil {
			return ErrUnsupportedCompressor
		}
		m.Payload, err = compressor.Unzip(m.Payload)
		if err != nil {
			return
		}
	}

	return err
}

// Reset clean data of this message but keep allocated data.
func (m *Message) Reset() {
	resetHeader(m.Header)
	m.Metadata = nil
	m.Payload = []byte{}
	m.data = m.data[:0]
	m.ServicePath = ""
	m.ServiceMethod = ""
}

func (m *Message) HandleError(err error) {
	if err == nil {
		return
	}

	m.SetMessageStatusType(Error)
	if m.Metadata == nil {
		m.Metadata = make(map[string]string)
	}
	m.Metadata[ServiceError] = err.Error()
	return
}

// NewMessage creates an empty message.
func NewMessage() *Message {
	header := Header([12]byte{})
	header[0] = magicNumber

	return &Message{
		Header: &header,
	}
}

// len,string,len,string,......
func encodeMetadata(m map[string]string, bb *bytebufferpool.ByteBuffer) {
	if len(m) == 0 {
		return
	}
	d := poolUint32Data.Get().(*[]byte)
	for k, v := range m {
		binary.BigEndian.PutUint32(*d, uint32(len(k)))
		_, _ = bb.Write(*d)
		_, _ = bb.Write(util.StringToSliceByte(k))
		binary.BigEndian.PutUint32(*d, uint32(len(v)))
		_, _ = bb.Write(*d)
		_, _ = bb.Write(util.StringToSliceByte(v))
	}
}

func decodeMetadata(l uint32, data []byte) (m map[string]string, err error) {
	m = make(map[string]string, 10)
	n := uint32(0)
	for n < l {
		// parse one key and value

		// key
		sl := binary.BigEndian.Uint32(data[n : n+4])
		n = n + 4
		if n+sl > l-4 {
			err = ErrMetaKVMissing
			return
		}
		k := string(data[n : n+sl])
		n = n + sl

		// value
		sl = binary.BigEndian.Uint32(data[n : n+4])
		n = n + 4
		if n+sl > l {
			err = ErrMetaKVMissing
			return
		}
		v := string(data[n : n+sl])
		n = n + sl
		m[k] = v
	}

	return
}

var (
	zeroHeaderArray Header
	zeroHeader      = zeroHeaderArray[1:]
)

func resetHeader(h *Header) {
	copy(h[1:], zeroHeader)
}
