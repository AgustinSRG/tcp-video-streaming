// Server utils

package main

import (
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Validates stream ID
// str - Received stream ID
// Returns true only if valid
func validateStreamIDString(str string) bool {
	var ID_MAX_LENGTH = 128
	var idCustomMaxLength = os.Getenv("ID_MAX_LENGTH")

	if idCustomMaxLength != "" {
		var e error
		ID_MAX_LENGTH, e = strconv.Atoi(idCustomMaxLength)
		if e != nil {
			ID_MAX_LENGTH = 128
		}
	}

	if len(str) > ID_MAX_LENGTH {
		return false
	}

	m, e := regexp.MatchString("^[A-Za-z0-9\\_\\-]+$", str)

	if e != nil {
		return false
	}

	return m
}

// Checks if a session is allowed to play streams
// ipStr - The client IP address
// Returns true only if allowed
func checkSessionCanPlay(ipStr string) bool {
	playWhiteList := os.Getenv("PLAY_WHITELIST")

	if playWhiteList == "" || playWhiteList == "*" {
		return true
	}

	ip := net.ParseIP(ipStr)

	parts := strings.Split(playWhiteList, ",")

	for i := 0; i < len(parts); i++ {
		_, rang, e := net.ParseCIDR(parts[i])

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
