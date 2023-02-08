// Encoding task

package main

import (
	"os"
	"sync"
)

// Encoding task data
type EncodingTask struct {
	server *HLS_Encoder_Server // Reference to the server

	channel  string // ID of the channel
	streamId string // ID of the stream

	sourceType string // Source type: WS or RTMP
	sourceURI  string // Source URI (eg: ws://host:port/channel/key or rtmp://host:port/channel/key)

	record bool // True if recording is enabled

	resolutions ResolutionList // List of resolutions for resizing

	previews PreviewsConfiguration // Configuration for making the previews

	mutex *sync.Mutex // Mutex to access the status data

	process *os.Process // Encoding process reference

	killed bool // True if the task was killed

	subStreams map[string]*SubStreamStatus // Sub-Streams
}

// Stores status for a specific sub-stream
type SubStreamStatus struct {
	resolution Resolution // Resolution

	livePlaylist          *HLS_PlayList // Live playlist
	livePlaylistAvailable bool          // True if live playlist is available

	vodPlaylist          *HLS_PlayList // VOD playlist
	vodPlaylistAvailable bool          // True if the current VOD playlist is available
	vodIndex             int           // VOD index

	fragmentCount int // Total number of fragments parsed and appended to the playlists

	previewsCount     int // Number of available previews
	previewsAvailable int // True if the previews stream is available

	fragmentsReady map[int]bool // Map to check when fragments are ready
}
