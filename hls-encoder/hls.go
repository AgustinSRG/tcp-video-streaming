// HLS utils

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
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

const (
	M3U8_DEFAULT_VERSION     = 3
	HARD_HLS_FRAGMENTS_LIMIT = 86400 // Max number of fragments allowed in a playlist due to format limitations
)

// Stores a HLS playlist
type HLS_PlayList struct {
	Version        int  // M3U8 version
	TargetDuration int  // Fragment duration
	MediaSequence  int  // First fragment index
	IsVOD          bool // True if the playlist is a VOD playlist
	IsEnded        bool // True if the playlist is an ended playlist

	fragments []HLS_Fragment // Video TS fragments
}

// Stores HLS fragment metadata
type HLS_Fragment struct {
	Index        int     // Fragment index
	Duration     float64 // Fragment duration
	FragmentName string  // Fragment file name
}

// Encodes playlist to M3U8
func (playlist *HLS_PlayList) Encode() string {
	result := "#EXTM3U" + "\n"

	if playlist.IsVOD {
		result += "#EXT-X-PLAYLIST-TYPE:VOD" + "\n"
	}

	result += "#EXT-X-VERSION:" + fmt.Sprint(playlist.Version) + "\n"
	result += "#EXT-X-TARGETDURATION:" + fmt.Sprint(playlist.TargetDuration) + "\n"
	result += "#EXT-X-MEDIA-SEQUENCE:" + fmt.Sprint(playlist.MediaSequence) + "\n"

	for i := 0; i < len(playlist.fragments); i++ {
		result += "#EXTINF:" + fmt.Sprintf("%0.6f", playlist.fragments[i].Duration) + "," + "\n"
		result += playlist.fragments[i].FragmentName + "\n"
	}

	if playlist.IsEnded {
		result += "#EXT-X-ENDLIST" + "\n"
	}

	return result
}

// Decodes HLS playlist
// m3u8 - Content of the .m3u8 file
// Returns the playlist data
func DecodeHLSPlayList(m3u8 string) *HLS_PlayList {
	result := &HLS_PlayList{
		Version:        M3U8_DEFAULT_VERSION,
		TargetDuration: HLS_DEFAULT_SEGMENT_TIME,
		MediaSequence:  0,
		IsVOD:          false,
		IsEnded:        false,
		fragments:      make([]HLS_Fragment, 0),
	}

	lines := strings.Split(m3u8, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "#") {
			continue
		}

		if line == "#EXT-X-ENDLIST" {
			result.IsEnded = true
			continue
		}

		parts := strings.Split(line, ":")

		if len(parts) != 2 {
			continue
		}

		switch strings.ToUpper(parts[0]) {
		case "#EXT-X-PLAYLIST-TYPE":
			if strings.ToUpper(parts[1]) == "VOD" {
				result.IsVOD = true
			}
		case "#EXT-X-VERSION":
			v, err := strconv.Atoi(parts[1])
			if err == nil && v >= 0 {
				result.Version = v
			}
		case "#EXT-X-TARGETDURATION":
			td, err := strconv.Atoi(parts[1])
			if err == nil && td > 0 {
				result.TargetDuration = td
			}
		case "#EXT-X-MEDIA-SEQUENCE":
			ms, err := strconv.Atoi(parts[1])
			if err == nil && ms >= 0 {
				result.MediaSequence = ms
			}
		case "#EXTINF":
			d, err := strconv.ParseFloat(strings.TrimSuffix(parts[1], ","), 64)

			if err != nil && d > 0 && i < (len(lines)-1) {
				frag := HLS_Fragment{
					Index:        len(result.fragments) + result.MediaSequence,
					Duration:     d,
					FragmentName: strings.TrimSpace(lines[i+1]),
				}

				result.fragments = append(result.fragments, frag)
			}
		}
	}

	return result
}
