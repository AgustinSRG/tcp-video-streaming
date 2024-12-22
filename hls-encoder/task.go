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

	hasStarted bool // True if the encoding started

	killed bool // True if the task was killed

	subStreams map[string]*SubStreamStatus // Sub-Streams

	previewsCount               int          // Number of available previews
	previewsAvailable           bool         // True if the previews stream is available
	previewsReady               map[int]bool // Map to check when fragments are ready
	writingPreviewsIndex        bool         // True if the task is writing the previews index
	pendingWritePreviewsIndex   bool         // True if there is a pending write for the previews index
	pendingPreviewsIndexContent []byte       // Content to write the previews index
}

// Stores previews metadata
type PreviewsIndexFile struct {
	IndexStart int    `json:"index_start"` // Index of the first image
	Count      int    `json:"count"`       // Number of available images
	Pattern    string `json:"pattern"`     // Image files pattern
}

// Stores status for a specific sub-stream
type SubStreamStatus struct {
	resolution Resolution // Resolution

	livePlaylist          *HLS_PlayList // Live playlist
	livePlaylistAvailable bool          // True if live playlist is available
	liveWriting           bool          // True if the live playlist is being written
	liveWritePending      bool          // True if the live playlist is pending of being written
	liveWriteData         []byte        // Data to write to the live playlist

	cdnPublisher      *CdnPublisher // CDN publisher
	cdnPublisherReady bool          // CDN publisher ready

	vodPlaylist          *HLS_PlayList  // VOD playlist
	vodPlaylistAvailable bool           // True if the current VOD playlist is available
	vodIndex             int            // VOD index
	vodTime              float64        // VOD time (seconds)
	vodStartTime         float64        // VOD start time (seconds)
	vodWriting           bool           // True if the vod playlist is being written
	vodWritePending      bool           // True if the vod playlist is pending of being written
	vodWriteData         []byte         // Data to write to the vod playlist
	vodFragmentBuffer    []HLS_Fragment // Buffer to temporally store the fragments before pushing them to the VOD playlists

	fragmentCount         int // Total number of fragments parsed and appended to the playlists
	removedFragmentsCount int // Number of removed fragments

	fragments      map[int]*HLS_Fragment // List of fragments extracted from the M3U8 file
	fragmentsReady map[int]bool          // Map to check when fragments are ready
}

// Logs a message for the task
func (task *EncodingTask) log(str string) {
	LogTaskStatus(task.channel, task.streamId, str)
}

// Logs a debug message for the task
func (task *EncodingTask) debug(str string) {
	LogDebugTask(task.channel, task.streamId, str)
}

// Call when encoding progress is made
func (task *EncodingTask) OnEncodingProgress() {
	task.mutex.Lock()
	defer task.mutex.Unlock()

	task.hasStarted = true
}

// Call after the encoding process ended
func (task *EncodingTask) OnEncodingEnded() {
	task.mutex.Lock()
	defer task.mutex.Unlock()

	if !task.hasStarted {
		return
	}

	// For each resolution, set the live playlist to ended and save it

	for _, subStream := range task.subStreams {
		if subStream.cdnPublisher != nil {
			subStream.cdnPublisher.Close()
		}

		if subStream.livePlaylist == nil {
			continue
		}

		subStream.livePlaylist.IsEnded = true

		livePlayListData := []byte(subStream.livePlaylist.Encode())

		if subStream.liveWriting {
			subStream.liveWritePending = true
			subStream.liveWriteData = livePlayListData
		} else {
			subStream.liveWriting = true
			go task.SaveLivePlaylist(subStream, livePlayListData)
		}
	}
}
