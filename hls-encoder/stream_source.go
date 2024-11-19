// Stream source utils

package main

import (
	"errors"
	"os/exec"
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
//
//	probeData - Video stream probe data
//	err - Error
func ProbeStreamSource(sourceType string, sourceURI string) (probeData *ffprobe.ProbeData, err error) {
	switch strings.ToUpper(sourceType) {
	case SOURCE_TYPE_WS:
		return ProbeStreamSource_WS(sourceURI)
	case SOURCE_TYPE_RTMP:
		return ProbeStreamSource_RTMP(sourceURI)
	default:
		return nil, errors.New("invalid source type")
	}
}

// Manager for input source
type InputSourceManager interface {
	Start() error // Starts sending the source to the process
	Close()       // Closes the source, clearing all resources
}

// Prepares an encoding process to receive a video stream source
// cmd - A reference to the command (will be configured)
// sourceType - Source type (WS, RTMP)
// sourceURI - Source URI
// Returns
//
//	manager - A manager to control the source (optional, can be nil)
//	err - Error
func PrepareEncodingProcessToReceiveSource(cmd *exec.Cmd, sourceType string, sourceURI string) (manager InputSourceManager, err error) {
	switch strings.ToUpper(sourceType) {
	case SOURCE_TYPE_WS:
		return PrepareEncodingProcessToReceiveSource_WS(cmd, sourceURI)
	case SOURCE_TYPE_RTMP:
		return PrepareEncodingProcessToReceiveSource_RTMP(cmd, sourceURI)
	default:
		return nil, errors.New("invalid source type")
	}
}
