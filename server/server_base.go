package server

import (
	"net"
	"sync/atomic"
)

func (s *Server) getService(serviceName string) *service {
	s.serviceMapMu.RLock()
	defer s.serviceMapMu.RUnlock()

	return s.serviceMap[serviceName]
}

func (s *Server) closeConn(conn net.Conn) {
	s.deleteActiveConn(conn)
	_ = conn.Close()

	s.Plugins.DoPostConnClose(conn)
}

// setActiveConn sets active connection in Server.
func (s *Server) setActiveConn(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.activeConn[conn] = struct{}{}
}

// deleteActiveConn deletes active connection in Server.
func (s *Server) deleteActiveConn(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.activeConn, conn)
}

// isShutdown returns true if Server is shutdown.
func (s *Server) isShutdown() bool {
	return atomic.LoadInt32(&s.inShutdown) == 1
}
