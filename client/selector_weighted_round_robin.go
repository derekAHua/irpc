package client

import (
	"context"
	"net/url"
	"strconv"
)

type (
	// weightedRoundRobinSelector selects servers with weighted.
	weightedRoundRobinSelector struct {
		servers []*Weighted
	}

	// Weighted is a wrapped server with  weight
	Weighted struct {
		Server          string
		Weight          int
		CurrentWeight   int
		EffectiveWeight int
	}
)

func (s *weightedRoundRobinSelector) Select(_ context.Context, _, _ string, _ interface{}) string {
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

func (s *weightedRoundRobinSelector) UpdateServer(servers map[string]string) {
	ss := createWeighted(servers)
	s.servers = ss
}

func newWeightedRoundRobinSelector(servers map[string]string) Selector {
	ss := createWeighted(servers)
	return &weightedRoundRobinSelector{servers: ss}
}

func createWeighted(servers map[string]string) []*Weighted {
	ss := make([]*Weighted, 0, len(servers))
	for k, metadata := range servers {
		w := &Weighted{Server: k, Weight: 1, EffectiveWeight: 1}

		if v, err := url.ParseQuery(metadata); err == nil {
			ww := v.Get("weight")
			if ww != "" {
				if weight, err := strconv.Atoi(ww); err == nil {
					w.Weight = weight
					w.EffectiveWeight = weight
				}
			}
		}

		ss = append(ss, w)
	}

	return ss
}

// func (w *Weighted) fail() {
// 	w.EffectiveWeight -= w.Weight
// 	if w.EffectiveWeight < 0 {
// 		w.EffectiveWeight = 0
// 	}
// }

// https://github.com/phusion/nginx/commit/27e94984486058d73157038f7950a0a36ecc6e35
func nextWeighted(servers []*Weighted) (best *Weighted) {
	total := 0

	for i := 0; i < len(servers); i++ {
		w := servers[i]

		if w == nil {
			continue
		}
		//if w is down, continue

		w.CurrentWeight += w.EffectiveWeight
		total += w.EffectiveWeight
		if w.EffectiveWeight < w.Weight {
			w.EffectiveWeight++
		}

		if best == nil || w.CurrentWeight > best.CurrentWeight {
			best = w
		}

	}

	if best == nil {
		return nil
	}

	best.CurrentWeight -= total
	return best
}
