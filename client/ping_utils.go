package client

import (
	"github.com/go-ping/ping"
)

// Ping gets network traffic by ICMP.
func Ping(host string) (rtt int, err error) {
	rtt = 1000 //default and timeout is 1000 ms

	pinger, err := ping.NewPinger(host)
	if err != nil {
		return rtt, err
	}
	pinger.Count = 3
	stats := pinger.Statistics()
	rtt = int(stats.AvgRtt)

	return rtt, err
}

// CalculateWeight converts the rtt to weighted by:
//  1. weight=191 if t <= 10
//  2. weight=201 -t if 10 < t <=200
//  3. weight=1 if 200 < t < 1000
//  4. weight = 0 if t >= 1000
//
// It means servers that ping time t < 10 will be preferred
// and servers won't be selected if t > 1000.
// It is hard coded based on Ops experience.
func CalculateWeight(rtt int) int {
	switch {
	case rtt >= 0 && rtt <= 10:
		return 191
	case rtt > 10 && rtt <= 200:
		return 201 - rtt
	case rtt > 100 && rtt < 1000:
		return 1
	default:
		return 0
	}
}
