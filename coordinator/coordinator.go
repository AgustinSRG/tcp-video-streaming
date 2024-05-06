// Streaming coordinator

package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// Stores the streaming coordination data
type Streaming_Coordinator struct {
	channels map[string]*StreamingChannel // Map of channels

	activeStreams map[string]bool // List of active streams, encoded as channel:stream, they are added here as soon as a stream-available event is sent, they are removed after the stream-closed event is sent

	rtmpServers map[uint64]*Streaming_RTMP_Server // Map of RTMP servers
	wssServers  map[uint64]*Streaming_WSS_Server  // Map of Websocket streaming servers

	hlsEncoders map[uint64]*HLS_Encoder_Server // Map of HLS encoders

	mutex *sync.Mutex // Mutex to access the data

	nextStreamId  uint32      // ID for the next stream
	streamIdMutex *sync.Mutex // Mutex to ensure stream IDs are unique

	pendingStreamClosedEvents map[string]*PendingStreamClosedEvent // List of stream closed event being sent

	savingActiveStreams             bool   // True if saving active streams
	pendingSaveActiveStreams        bool   // True if there is pending active streams to save
	pendingSaveActiveStreamsContent string // Content to save in the pending streams file
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

	coord.pendingSaveActiveStreams = false
	coord.pendingSaveActiveStreamsContent = ""

	coord.LoadPastActiveStreams()
}

// Acquires the access to a streaming channel data struct
// Only a single thread can access
// channelId - Id of the channel
// Returns a ref to the streaming channel
// Must call ReleaseChannel after the thread is done with the struct
func (coord *Streaming_Coordinator) AcquireChannel(channelId string) *StreamingChannel {
	coord.mutex.Lock()

	channel := coord.channels[channelId]

	if channel == nil {
		channel = &StreamingChannel{
			id:            channelId,
			usageCount:    0,
			mutex:         &sync.Mutex{},
			streamId:      "",
			publishMethod: 0,
			publisher:     0,
			encoder:       0,
			nextEventId:   0,
			pendingEvents: make(map[uint64]*PendingStreamAvailableEvent),
			closed:        true,
		}

		coord.channels[channelId] = channel
	}

	channel.usageCount++

	coord.mutex.Unlock()

	channel.mutex.Lock()

	return channel
}

// Releases a stream channel data struct
// channel - Channel data struct
func (coord *Streaming_Coordinator) ReleaseChannel(channel *StreamingChannel) {
	channel.usageCount--

	mustDelete := (channel.usageCount <= 0 && channel.closed)

	channel.mutex.Unlock()

	if mustDelete {
		coord.mutex.Lock()

		// No threads wanting it, and it's closed

		delete(coord.channels, channel.id)

		coord.mutex.Unlock()
	}
}

// Generates an unique stream ID
func (coord *Streaming_Coordinator) GenerateStreamID() string {
	idBytes := make([]byte, 16)

	binary.BigEndian.PutUint64(idBytes[0:8], uint64(time.Now().UnixMilli()))

	coord.mutex.Lock()

	coord.nextStreamId++

	binary.BigEndian.PutUint32(idBytes[8:12], coord.nextStreamId)

	coord.mutex.Unlock()

	_, err := rand.Read(idBytes[12:])

	if err != nil {
		LogError(err)
	}

	return strings.ToLower(hex.EncodeToString(idBytes))
}

// Registers encoder server
// id - Server ID
// capacity - Server capacity
func (coord *Streaming_Coordinator) RegisterEncoder(id uint64, capacity int) {
	coord.mutex.Lock()
	defer coord.mutex.Unlock()

	coord.hlsEncoders[id] = &HLS_Encoder_Server{
		id:       id,
		capacity: capacity,
		load:     0,
	}
}

// Deregister encoder server
// id - Server ID
func (coord *Streaming_Coordinator) DeregisterEncoder(id uint64) {
	coord.mutex.Lock()
	defer coord.mutex.Unlock()

	delete(coord.hlsEncoders, id)
}

// Removes streaming server
// sessionType - Type of server
// id - Server ID
// ip - Server IP
// port - Server port
// ssl - True if the server uses SSL
func (coord *Streaming_Coordinator) RegisterStreamingServer(sessionType int, id uint64, ip string, port int, ssl bool) {
	coord.mutex.Lock()
	defer coord.mutex.Unlock()

	switch sessionType {
	case SESSION_TYPE_RTMP:
		coord.rtmpServers[id] = &Streaming_RTMP_Server{
			id:   id,
			ip:   ip,
			port: port,
			ssl:  ssl,
		}
	case SESSION_TYPE_WSS:
		coord.wssServers[id] = &Streaming_WSS_Server{
			id:   id,
			ip:   ip,
			port: port,
			ssl:  ssl,
		}
	}
}

// Removes streaming server
// sessionType - Type of server
// id - Server ID
func (coord *Streaming_Coordinator) DeregisterStreamingServer(sessionType int, id uint64) {
	coord.mutex.Lock()
	defer coord.mutex.Unlock()

	switch sessionType {
	case SESSION_TYPE_RTMP:
		delete(coord.rtmpServers, id)
	case SESSION_TYPE_WSS:
		delete(coord.wssServers, id)
	}
}

// Adds active stream to the list
// channel - The channel
// streamId - Stream ID
func (coord *Streaming_Coordinator) AddActiveStream(channel string, streamId string) {
	id := channel + ":" + streamId

	coord.mutex.Lock()
	defer coord.mutex.Unlock()

	if !coord.activeStreams[id] {
		coord.activeStreams[id] = true
		coord.SavePastActiveStreams()
	}
}

// Call when an active stream is closed (HLS encoder process ends)
// channel - The channel
// streamId - Stream ID
func (coord *Streaming_Coordinator) OnActiveStreamClosed(channel string, streamId string) {
	id := channel + ":" + streamId

	coord.mutex.Lock()
	defer coord.mutex.Unlock()

	if coord.activeStreams[id] && coord.pendingStreamClosedEvents[streamId] == nil {
		event := &PendingStreamClosedEvent{
			channel:   channel,
			streamId:  streamId,
			cancelled: false,
		}

		coord.pendingStreamClosedEvents[streamId] = event

		go SendStreamClosedEvent(coord, event)
	}
}

// Removes an active stream from the list
// channel - The channel
// streamId - Stream ID
func (coord *Streaming_Coordinator) RemoveActiveStream(channel string, streamId string) {
	id := channel + ":" + streamId

	coord.mutex.Lock()
	defer coord.mutex.Unlock()

	delete(coord.activeStreams, id)

	coord.SavePastActiveStreams()
}
