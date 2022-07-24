package server

import (
	"crypto/tls"
	"time"
)

// Option configures options of server.
type Option func(*Server)

// WithTLSConfig sets tls.Config.
func WithTLSConfig(cfg *tls.Config) Option {
	return func(s *Server) {
		s.tlsConfig = cfg
	}
}

// WithReadTimeout sets readTimeout.
func WithReadTimeout(readTimeout time.Duration) Option {
	return func(s *Server) {
		s.readTimeout = readTimeout
	}
}

// WithWriteTimeout sets writeTimeout.
func WithWriteTimeout(writeTimeout time.Duration) Option {
	return func(s *Server) {
		s.writeTimeout = writeTimeout
	}
}

//// WithTCPKeepAlivePeriod sets tcp keepalive period.
//func WithTCPKeepAlivePeriod(period time.Duration) Option {
//	return func(s *Server) {
//		s.options["TCPKeepAlivePeriod"] = period
//	}
//}
