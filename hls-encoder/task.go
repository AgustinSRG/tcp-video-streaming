// Encoding task

package main

import (
	"encoding/json"
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
	task.mutex.Lock()
	defer task.mutex.Unlock()
}

func (task *EncodingTask) OnPlaylistUpdate(resolution Resolution, playlist *HLS_PlayList) {
	task.mutex.Lock()
	defer task.mutex.Unlock()
}

func (task *EncodingTask) OnPreviewImageReady(previewIndex int) {
	task.mutex.Lock()
	defer task.mutex.Unlock()

	if previewIndex < task.previewsCount {
		return // Ignore already added images
	}

	task.previewsReady[previewIndex] = true

	oldCount := task.previewsCount
	newCount := oldCount
	doneCounting := false

	for !doneCounting {
		if task.previewsReady[newCount+1] {
			delete(task.previewsReady, newCount+1)
			newCount++
		} else {
			doneCounting = true
		}
	}

	if oldCount != newCount {
		// Update new count
		task.previewsCount = newCount

		// Update index file
		newIndexFile := PreviewsIndexFile{
			IndexStart: 0,
			Count:      newCount,
			Pattern:    "%d.jpg",
		}

		data, err := json.Marshal(newIndexFile)

		if err != nil {
			LogError(err)
			return
		}

		task.writingPreviewsIndex = true
		go task.SaveImagePreviewsIndex(data)
	}
}

// Call after the image previews index is successfully saved
func (task *EncodingTask) OnImagePreviewsIndexSaved() {
	shouldAnnounce := false
	task.mutex.Lock()

	if !task.previewsAvailable {
		task.previewsAvailable = true
		shouldAnnounce = true
	}

	task.mutex.Unlock()

	if shouldAnnounce {
		task.server.websocketControlConnection.SendStreamAvailable(task.channel, task.streamId, "IMG-PREVIEW", Resolution{
			width:  task.previews.width,
			height: task.previews.height,
			fps:    task.previews.delaySeconds},
			"img-preview/"+task.channel+"/"+task.streamId+"/"+task.previews.Encode("-")+"/index.json",
		)
	}
}

// Saves image previews index
// data - Contents of the index file
func (task *EncodingTask) SaveImagePreviewsIndex(data []byte) {
	filePath := "img-preview/" + task.channel + "/" + task.streamId + "/" + task.previews.Encode("-") + "/index.json"
	done := false

	dataToWrite := data

	for !done {
		err := WriteFileBytes(filePath, dataToWrite)

		if err != nil {
			LogError(err)
		} else {
			task.OnImagePreviewsIndexSaved()
		}

		task.mutex.Lock()

		if task.pendingWritePreviewsIndex {
			task.pendingWritePreviewsIndex = false
			dataToWrite = task.pendingPreviewsIndexContent
			task.pendingPreviewsIndexContent = nil
		} else {
			task.writingPreviewsIndex = false
			done = true
		}

		task.mutex.Unlock()
	}
}
