// Stream source utils

package main

import (
	"errors"
	"strings"
	"time"

	"gopkg.in/vansante/go-ffprobe.v2"
)

const (
	SOURCE_TYPE_WS   = "WS"
	SOURCE_TYPE_RTMP = "RTMP"
)

const SOURCE_PROBE_TIMEOUT = 30 * time.Second

// Probes video stream source
// sourceType - Source type (WS, RTMP)
// sourceURI - Source URI
// Returns
//  probeData - Video stream probe data
//  err - Error
func ProbeStreamSource(sourceType string, sourceURI string) (probeData *ffprobe.ProbeData, err error) {
	switch strings.ToUpper(sourceType) {
	case SOURCE_TYPE_WS:
		return ProbeStreamSource_WS(sourceURI)
	case SOURCE_TYPE_RTMP:
		return ProbeStreamSource_RTMP(sourceURI)
	default:
		return nil, errors.New("Invalid source type")
	}
}
