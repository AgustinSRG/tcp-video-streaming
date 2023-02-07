// Stream source utils (For RTMP sources)

package main

import (
	"context"
	"os/exec"

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

// Prepares an encoding process to receive a RTMP source
// cmd - A reference to the command (will be configured)
// sourceURI - RTMP URI
// Returns
//  manager - A manager to control the source. In this case, no manager is needed, so it will return nil
//  err - Error
func PrepareEncodingProcessToReceiveSource_RTMP(cmd *exec.Cmd, sourceURI string) (manager InputSourceManager, err error) {
	cmd.Args = append(cmd.Args, "-i", sourceURI)
	return nil, nil
}
