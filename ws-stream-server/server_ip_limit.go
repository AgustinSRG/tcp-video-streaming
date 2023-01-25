// Connections limits handler

package main

import (
	"net"
	"os"
	"strings"

	"github.com/netdata/go.d.plugin/pkg/iprange"
)

// Adds IP address to the list
// ip - IP address
// Returns false if the limit has been reached
func (server *WS_Streaming_Server) AddIP(ip string) bool {
	server.mutexIpCount.Lock()
	defer server.mutexIpCount.Unlock()

	c := server.ipCount[ip]

	if c >= server.ipLimit {
		return false
	}

	server.ipCount[ip] = c + 1

	return true
}

// Checks if an IP is exempted from the limit
// ipStr - IP address
// Returns true only if exempted
func (server *WS_Streaming_Server) isIPExempted(ipStr string) bool {
	r := os.Getenv("CONCURRENT_LIMIT_WHITELIST")

	if r == "" {
		return false
	}

	if r == "*" {
		return true
	}

	ip := net.ParseIP(ipStr)

	parts := strings.Split(r, ",")

	for i := 0; i < len(parts); i++ {
		rang, e := iprange.ParseRange(parts[i])

		if e != nil {
			LogError(e)
			continue
		}

		if rang.Contains(ip) {
			return true
		}
	}

	return false
}

// Removes an IP from the list
// ip - IP address
func (server *WS_Streaming_Server) RemoveIP(ip string) {
	server.mutexIpCount.Lock()
	defer server.mutexIpCount.Unlock()

	c := server.ipCount[ip]

	if c <= 1 {
		delete(server.ipCount, ip)
	} else {
		server.ipCount[ip] = c - 1
	}
}
