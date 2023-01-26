// Session

package main

import (
	"container/list"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Structure to store the bitrate status
type BitRateCache struct {
	intervalMs int64  // Interval of milliseconds to update
	lastUpdate int64  // Last time updated (unix millis)
	bytes      uint64 // The number of bytes received
}

const (
	DATA_STREAM_PACKET_BASE_SIZE = 16
)

// Chunk of a sta stream
type DataStreamChunk struct {
	data []byte // The data
	size int    // The size (bytes)
}

// Status for a streaming session
type WS_Streaming_Session struct {
	server *WS_Streaming_Server // Reference to the server

	conn *websocket.Conn // Websocket connection

	id uint64 // Session ID
	ip string // IP address of the client

	mutex *sync.Mutex // Mutex to control access to the session status data

	publishMutex *sync.Mutex // Mutex to control the publishing group

	closed bool // True if the connection is closed

	channel  string // Streaming channel ID
	key      string // Streaming key
	streamId string // Stream ID

	isPublishing bool // True if the connection is publishing a stream
	isPlaying    bool // True if the connection is receiving a stream
	isProbing    bool // True if the connection is probing the stream
	isIdling     bool // True if the connection is waiting for the stream

	chunksCount int64 // Total data chunks sent

	gopCache         *list.List // List to store the GOP cache
	gopCacheSize     int64      // Current GOP cache size
	gopCacheLimit    int64      // GOP cache size limit
	gopCacheDisabled bool       // True if the cache is currently disabled
	gopPlayNo        bool       // True if the client refuses to receive the cache packets
	gopPlayClear     bool       // True if the clients is requesting to clear the cache

	bitRate      uint64       // Bitrate (bit/ms)
	bitRateCache BitRateCache // Cache to compute bitrate
}

// Handles incoming connection
// sessionId - Session ID
// w - Writer to send the response
// req - Client request
// ip - Client IP
// channel - Streaming channel ID
// key - Streaming key
// connectionKind - Connection kind
func (server *WS_Streaming_Server) HandleStreamingSession(sessionId uint64, w http.ResponseWriter, req *http.Request, ip string, channel string, key string, connectionKind string) {
	var isPublishing bool
	var isPlaying bool
	var isProbing bool
	var gopPlayClear bool

	if !validateStreamIDString(channel) {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Invalid channel.")
		server.RemoveIP(ip)
		return
	}

	if !validateStreamIDString(key) {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Invalid key.")
		server.RemoveIP(ip)
		return
	}

	switch connectionKind {
	case "publish":
		isPublishing = true
		isPlaying = false
		isProbing = false
		gopPlayClear = false
	case "receive":
		isPublishing = false
		isPlaying = true
		isProbing = false
		gopPlayClear = false
	case "receive-clear-cache":
		isPublishing = false
		isPlaying = true
		isProbing = false
		gopPlayClear = true
	case "probe":
		isPublishing = false
		isPlaying = true
		isProbing = true
		gopPlayClear = false
	default:
		w.WriteHeader(404)
		fmt.Fprintf(w, "Invalid connection kind.")
		server.RemoveIP(ip)
		return
	}

	// Checks

	streamId := ""

	if isPublishing {
		// Publishing
		if server.isPublishing(channel) {
			w.WriteHeader(403)
			fmt.Fprintf(w, "Already publishing.")
			server.RemoveIP(ip)
			return
		}

		LogRequest(sessionId, ip, "PUBLISH REQUEST: '"+channel+"'")

		pubAccepted, publishStreamId := server.controlConnection.RequestPublish(channel, key, ip)
		if !pubAccepted {
			LogRequest(sessionId, ip, "Error: Invalid streaming key provided")
			w.WriteHeader(403)
			fmt.Fprintf(w, "Invalid Key.")
			server.RemoveIP(ip)
			return
		}

		streamId = publishStreamId
	} else {
		// Play
		if !checkSessionCanPlay(ip) {
			LogRequest(sessionId, ip, "Error: Cannot play: Not whitelisted")
			w.WriteHeader(403)
			fmt.Fprintf(w, "Cannot play: Not whitelisted.")
			server.RemoveIP(ip)
			return
		}
	}

	// Create session

	conn, err := server.wsUpgrader.Upgrade(w, req, nil)

	if err != nil {
		LogError(err)
		server.RemoveIP(ip)
		return
	}

	session := WS_Streaming_Session{
		id:               sessionId,
		server:           server,
		ip:               ip,
		conn:             conn,
		mutex:            &sync.Mutex{},
		publishMutex:     &sync.Mutex{},
		closed:           false,
		channel:          channel,
		key:              key,
		streamId:         streamId,
		isPublishing:     isPublishing,
		isPlaying:        isPlaying,
		isProbing:        isProbing,
		chunksCount:      0,
		gopPlayNo:        false,
		gopPlayClear:     gopPlayClear,
		gopCache:         list.New(),
		gopCacheDisabled: server.gopCacheLimit <= 0,
		gopCacheSize:     0,
		gopCacheLimit:    server.gopCacheLimit,
	}

	server.AddSession(&session) // Add session to the server

	// Run session
	go session.Run()
}

// Sends a text message to the client
// txt - Message contents
func (session *WS_Streaming_Session) SendText(txt string) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	return session.conn.WriteMessage(websocket.TextMessage, []byte(txt))
}

// Sends a chunk to the client
// chunkData - Chunk to send
func (session *WS_Streaming_Session) SendChunk(chunkData []byte) {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.isProbing && session.chunksCount > 0 {
		return
	}

	session.conn.WriteMessage(websocket.BinaryMessage, chunkData)
	session.chunksCount++

	if session.isProbing {
		session.conn.Close()
	}
}

// Periodically sends heartbeat messages to the client
// Loops until the connection is closed
func (session *WS_Streaming_Session) SendHeartBeatMessages() {
	for {
		time.Sleep(20 * time.Second)

		err := session.SendText("h")

		if err != nil {
			return
		}
	}
}

// Closes the connection
func (session *WS_Streaming_Session) Kill() {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.conn.Close()
}

// Runs the connection, reading the incoming messages until the connection is closed
func (session *WS_Streaming_Session) Run() {
	defer func() {
		if err := recover(); err != nil {
			switch x := err.(type) {
			case string:
				session.log("Error: " + x)
			case error:
				session.log("Error: " + x.Error())
			default:
				session.log("Connection Crashed!")
			}
		}
		session.log("Connection closed.")
		// Ensure connection is closed
		session.conn.Close()
		session.closed = true
		// Release resources
		session.onClose()
		// Remove connection
		session.server.RemoveIP(session.ip)
		session.server.RemoveSession(session.id)
	}()

	if session.isPublishing {
		session.server.SetPublisher(session.channel, session.key, session.streamId, session)
	} else {
		idle, err := session.server.AddPlayer(session.channel, session.key, session)

		if err != nil {
			session.log("Error: Invalid streaming key provided")
			session.SendText("ERROR: Invalid streaming key")
			return
		}

		if !idle {
			publisher := session.server.GetPublisher(session.channel)
			if publisher != nil {
				publisher.StartPlayer(session)
			}
		} else {
			session.log("PLAY IDLE '" + session.channel + "'")
		}
	}

	// Heartbeat
	go session.SendHeartBeatMessages()

	// Bitrate init

	session.bitRate = 0
	session.bitRateCache.bytes = 0
	session.bitRateCache.intervalMs = 1000
	session.bitRateCache.lastUpdate = time.Now().UnixMilli()

	// Read incoming messages
	for {
		err := session.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		if err != nil {
			return
		}

		msgType, message, err := session.conn.ReadMessage()

		if err != nil {
			return
		}

		if msgType == websocket.TextMessage {
			msgStr := string(message)

			if LOG_DEBUG_ENABLED {
				session.debug("<<< " + msgStr)
			}
		} else if msgType == websocket.BinaryMessage {
			if LOG_DEBUG_ENABLED {
				session.debug("Received chunk (" + fmt.Sprint(len(message)) + " bytes)")
			}

			if session.isPublishing {
				session.HandleChunk(message)
			}
		}

		// Bitrate
		now := time.Now().UnixMilli()
		session.bitRateCache.bytes += uint64(len(message))
		diff := now - session.bitRateCache.lastUpdate
		if diff >= session.bitRateCache.intervalMs {
			session.bitRate = uint64(math.Round(float64(session.bitRateCache.bytes) * 8 / float64(diff)))
			session.bitRateCache.bytes = 0
			session.bitRateCache.lastUpdate = now
			session.debug("Bitrate is now: " + strconv.Itoa(int(session.bitRate)))
		}
	}
}

// Logs a message for this connection
// str - message to log
func (session *WS_Streaming_Session) log(str string) {
	LogRequest(session.id, session.ip, str)
}

// Logs a debug message for this connection
// str - message to log
func (session *WS_Streaming_Session) debug(str string) {
	LogDebugSession(session.id, session.ip, str)
}

// Call after the connection is closed
func (session *WS_Streaming_Session) onClose() {
	if session.isPublishing {
		session.EndPublish()
		session.isPublishing = false
	} else if session.isPlaying {
		session.server.RemovePlayer(session.channel, session.key, session)
		session.isPlaying = false
		session.isIdling = false
	}
}
