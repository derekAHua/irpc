package client

import (
	"context"
	"github.com/edwingeng/doublejump"
	"sort"
)

// consistentHashSelector selects based on JumpConsistentHash.
type consistentHashSelector struct {
	h       *doublejump.Hash
	servers []string
}

func (s consistentHashSelector) Select(_ context.Context, servicePath, serviceMethod string, args interface{}) string {
	ss := s.servers
	if len(ss) == 0 {
		return ""
	}

	key := genKey(servicePath, serviceMethod, args)
	selected, _ := s.h.Get(key).(string)
	return selected
}

func (s *consistentHashSelector) UpdateServer(servers map[string]string) {
	ss := make([]string, 0, len(servers))
	for k := range servers {
		s.h.Add(k)
		ss = append(ss, k)
	}

	sort.Slice(ss, func(i, j int) bool { return ss[i] < ss[j] })

	for _, k := range s.servers {
		if servers[k] == "" { // remove
			s.h.Remove(k)
		}
	}
	s.servers = ss
}

func newConsistentHashSelector(servers map[string]string) Selector {
	h := doublejump.NewHash()
	ss := make([]string, 0, len(servers))
	for k := range servers {
		ss = append(ss, k)
		h.Add(k)
	}

	sort.Slice(ss, func(i, j int) bool { return ss[i] < ss[j] })
	return &consistentHashSelector{servers: ss, h: h}
}
