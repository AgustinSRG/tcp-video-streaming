// Persistence of active streams

package main

import (
	"io/ioutil"
	"os"
	"strings"
)

const ACTIVE_STREAMS_TMP_FILE = "active_streams.tmp"
const ACTIVE_STREAMS_FILE = "active_streams.txt"

// Loads the list of active streams from a file
func (coord *Streaming_Coordinator) LoadPastActiveStreams() {
	coord.mutex.Lock()
	defer coord.mutex.Unlock()

	content, err := ioutil.ReadFile(ACTIVE_STREAMS_FILE)

	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")

	for i := 0; i < len(lines); i++ {
		parts := strings.Split(lines[i], ":")

		if len(parts) != 2 {
			continue
		}

		channel := parts[0]
		streamId := parts[1]

		coord.activeStreams[channel+":"+streamId] = true

		event := &PendingStreamClosedEvent{
			channel:   channel,
			streamId:  streamId,
			cancelled: false,
		}

		coord.pendingStreamClosedEvents[streamId] = event

		go SendStreamClosedEvent(coord, event)
	}
}

// Saves the current list of active streams to a file
func (coord *Streaming_Coordinator) SavePastActiveStreams() {
	str := ""

	for stream := range coord.activeStreams {
		str += stream + "\n"
	}

	if coord.savingActiveStreams {
		coord.pendingSaveActiveStreams = true
		coord.pendingSaveActiveStreamsContent = str
	} else {
		coord.savingActiveStreams = true
		go coord.SavePastActiveStreamsInternal([]byte(str))
	}
}

// Internal method to save the active streams to the file
// content - Content to save
func (coord *Streaming_Coordinator) SavePastActiveStreamsInternal(content []byte) {
	done := false
	toSave := content

	for !done {
		err := ioutil.WriteFile(ACTIVE_STREAMS_TMP_FILE, toSave, FILE_PERMISSION)

		if err != nil {
			LogError(err)
		} else {
			err = os.Rename(ACTIVE_STREAMS_TMP_FILE, ACTIVE_STREAMS_FILE)

			if err != nil {
				LogError(err)
			}
		}

		coord.mutex.Lock()

		if coord.pendingSaveActiveStreams {
			toSave = []byte(coord.pendingSaveActiveStreamsContent)
			coord.pendingSaveActiveStreams = false
			coord.pendingSaveActiveStreamsContent = ""
		} else {
			coord.savingActiveStreams = false
			done = true
		}

		coord.mutex.Unlock()
	}
}
