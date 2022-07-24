package client

import (
	"context"
	"net"
	"strings"
)

// weightedICMPSelector selects servers with ping result.
type weightedICMPSelector struct {
	servers []*Weighted
}

func (s weightedICMPSelector) Select(_ context.Context, _, _ string, _ interface{}) string {
	ss := s.servers
	if len(ss) == 0 {
		return ""
	}
	w := nextWeighted(ss)
	if w == nil {
		return ""
	}
	return w.Server
}

func (s *weightedICMPSelector) UpdateServer(servers map[string]string) {
	ss := createICMPWeighted(servers)
	s.servers = ss
}

func newWeightedICMPSelector(servers map[string]string) Selector {
	ss := createICMPWeighted(servers)
	return &weightedICMPSelector{servers: ss}
}

func createICMPWeighted(servers map[string]string) []*Weighted {
	var ss = make([]*Weighted, 0, len(servers))
	for k := range servers {
		w := &Weighted{Server: k, Weight: 1, EffectiveWeight: 1}
		server := strings.Split(k, "@")
		host, _, _ := net.SplitHostPort(server[1])
		rtt, _ := Ping(host)
		rtt = CalculateWeight(rtt)
		w.Weight = rtt
		w.EffectiveWeight = rtt
		ss = append(ss, w)
	}
	return ss
}
