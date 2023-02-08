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

	previewsCount     int          // Number of available previews
	previewsAvailable int          // True if the previews stream is available
	previewsReady     map[int]bool // Map to check when fragments are ready
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

	fragmentsReady map[int]bool // Map to check when fragments are ready
}

// Logs a message for the task
func (task *EncodingTask) log(str string) {
	LogTaskStatus(task.channel, task.streamId, str)
}

// Logs a debug message for the task
func (task *EncodingTask) debug(str string) {
	LogDebugTask(task.channel, task.streamId, str)
}

func (task *EncodingTask) OnFragmentReady(resolution Resolution, fragmentIndex int) {
}

func (task *EncodingTask) OnPlaylistUpdate(resolution Resolution, playlist *HLS_PlayList) {
}

func (task *EncodingTask) OnPreviewImageReady(previewIndex int) {
}
