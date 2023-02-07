// Stream source utils (For RTMP sources)

package main

import (
	"context"

	"gopkg.in/vansante/go-ffprobe.v2"
)

// Probes RTMP source
// sourceURI - RTMP URI
// Returns
//  probeData - Video stream probe data
//  err - Error
func ProbeStreamSource_RTMP(sourceURI string) (probeData *ffprobe.ProbeData, err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), SOURCE_PROBE_TIMEOUT)
	defer cancelFn()

	data, err := ffprobe.ProbeURL(ctx, sourceURI)
	if err != nil {
		return nil, err
	}

	return data, nil
}
