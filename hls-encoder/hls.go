// HLS utils

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

const (
	HLS_INTERNAL_PLAYLIST_SIZE = 5
	HLS_DEFAULT_PLAYLIST_SIZE  = 10
	HLS_DEFAULT_SEGMENT_TIME   = 3
)

// Returns the configured HLS segment time
func GetConfiguredHLSTime() int {
	configuredTime := os.Getenv("HLS_TIME_SECONDS")
	if configuredTime != "" {
		t, err := strconv.ParseInt(configuredTime, 10, 32)

		if err != nil || t <= 0 {
			return HLS_DEFAULT_SEGMENT_TIME
		}

		return int(t)
	} else {
		return HLS_DEFAULT_SEGMENT_TIME
	}
}

// Returns the configured HLS playlist size
func GetConfiguredHLSPlaylistSize() int {
	configuredTime := os.Getenv("HLS_LIVE_PLAYLIST_SIZE")
	if configuredTime != "" {
		t, err := strconv.ParseInt(configuredTime, 10, 32)

		if err != nil || t <= 0 {
			return HLS_DEFAULT_PLAYLIST_SIZE
		}

		return int(t)
	} else {
		return HLS_DEFAULT_PLAYLIST_SIZE
	}
}

// Appends HLS arguments to the encoder command
// cmd - The command
// resolution - The resolution
// task - The task
func AppendGenericHLSArguments(cmd *exec.Cmd, resolution Resolution, task *EncodingTask) {
	// Set format
	cmd.Args = append(cmd.Args, "-f", "hls")

	// Force key frames so we can cut each second
	cmd.Args = append(cmd.Args, "-force_key_frames", "expr:gte(t,n_forced*1)")

	// Set HLS options
	cmd.Args = append(cmd.Args, "-hls_list_size", fmt.Sprint(HLS_INTERNAL_PLAYLIST_SIZE))
	cmd.Args = append(cmd.Args, "-hls_time", fmt.Sprint(GetConfiguredHLSPlaylistSize()))

	// Method and URL
	cmd.Args = append(cmd.Args, "-method", "PUT")
	cmd.Args = append(cmd.Args, "-hls_segment_filename", "http://127.0.0.1:"+fmt.Sprint(task.server.loopBackPort)+"/"+task.channel+"/"+task.streamId+"/"+resolution.Encode()+"/%d.ts")
	cmd.Args = append(cmd.Args, "http://127.0.0.1:"+fmt.Sprint(task.server.loopBackPort)+"/"+task.channel+"/"+task.streamId+"/"+resolution.Encode()+"/index.m3u8")
}
