// Streaming coordinator

package main

import (
	"io/ioutil"
	"strings"
	"sync"
)

// Stores the streaming coordination data
type Streaming_Coordinator struct {
	channels map[string]*StreamingChannel // Map of channels

	activeStreams map[string]bool // List of active streams, encoded as channel:stream, they are added here as soon as a stream-available event is sent, they are removed after the stream-closed event is sent

	rtmpServers map[uint64]*Streaming_RTMP_Server // Map of RTMP servers
	wssServers  map[uint64]*Streaming_WSS_Server  // Map of Websocket streaming servers

	hlsEncoders map[uint64]*HLS_Encoder_Server // Map of HLS encoders

	mutex *sync.Mutex // Mutex to access the data

	nextStreamId  uint64      // ID for the next stream
	streamIdMutex *sync.Mutex // Mutex to ensure stream IDs are unique

	pendingStreamClosedEvents map[string]*PendingStreamClosedEvent // List of stream closed event being sent
}

const (
	PUBLISH_METHOD_RTMP = 1
	PUBLISH_METHOD_WS   = 2
)

// Stores the status of a streaming channel
type StreamingChannel struct {
	id string // Channel ID

	usageCount int         // Number of threads using the struct
	mutex      *sync.Mutex // Mutex to access the data

	streamId string // Current stream ID

	publishMethod int // Publish method, use PUBLISH_METHOD_RTMP or PUBLISH_METHOD_WS

	publisher uint64 // Id of the server where the publisher is connected
	encoder   uint64 // ID of the HLS encoder assigned to the stream

	nextEventId   uint64                                  // Id for the next stream-available event
	pendingEvents map[uint64]*PendingStreamAvailableEvent // Pending stream-available events

	closed bool // True if the channel is closed
}

// Stores the information for sending Stream-Available events
type PendingStreamAvailableEvent struct {
	id uint64 // ID of the event call

	channel  string // Channel ID
	streamId string // Stream ID

	streamType string // Stream type: HLS-LIVE, HLS-VOD, IMG-PREVIEW
	resolution string // Resolution: {WIDTH}x{HEIGHT}-{FPS}
	indexFile  string // The index file path

	cancelled bool // True if the event got cancelled
}

// Stores information about a RTMP streaming server
type Streaming_RTMP_Server struct {
	id uint64 // ID

	ip   string // Server IP
	port int    // Server port
	ssl  bool   // True if uses SSL
}

// Stores information about a websocket streaming server
type Streaming_WSS_Server struct {
	id uint64 // ID

	ip   string // Server IP
	port int    // Server port
	ssl  bool   // True if uses SSL
}

// Stores information about a HLS encoder
type HLS_Encoder_Server struct {
	id uint64 // ID

	capacity int // Server capacity (number of streams it can handle in parallel)

	load int // Current server load (number of streams being handled)
}

// Stores the information for sending Stream-Closed events
type PendingStreamClosedEvent struct {
	channel  string // Channel ID
	streamId string // Stream ID

	cancelled bool // True if the event got cancelled
}

// Initializes the coordinator status data
func (coord *Streaming_Coordinator) Initialize() {
	coord.mutex = &sync.Mutex{}

	coord.channels = make(map[string]*StreamingChannel)

	coord.activeStreams = make(map[string]bool)

	coord.rtmpServers = make(map[uint64]*Streaming_RTMP_Server)
	coord.wssServers = make(map[uint64]*Streaming_WSS_Server)

	coord.hlsEncoders = make(map[uint64]*HLS_Encoder_Server)

	coord.nextStreamId = 0
	coord.streamIdMutex = &sync.Mutex{}

	coord.pendingStreamClosedEvents = make(map[string]*PendingStreamClosedEvent)

	coord.LoadPastActiveStreams()
}

const ACTIVE_STREAMS_TMP_FILE = "active_streams.tmp"

// Loads the list of active streams from a file
func (coord *Streaming_Coordinator) LoadPastActiveStreams() {
	content, err := ioutil.ReadFile(ACTIVE_STREAMS_TMP_FILE)

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

		coord.pendingStreamClosedEvents[streamId] = &PendingStreamClosedEvent{
			channel:   channel,
			streamId:  streamId,
			cancelled: false,
		}
	}

	// TODO: Start event senders
}

// Saves the current list of active streams to a file
func (coord *Streaming_Coordinator) SavePastActiveStreams() {
	str := ""

	for stream := range coord.activeStreams {
		str += stream + "\n"
	}

	err := ioutil.WriteFile(ACTIVE_STREAMS_TMP_FILE, []byte(str), FILE_PERMISSION)

	if err != nil {
		LogError(err)
	}
}
