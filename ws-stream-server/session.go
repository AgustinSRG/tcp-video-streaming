// Session

package main

import (
	"container/list"
	"crypto/subtle"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Cache to compute bit-rate
type BitRateCache struct {
	intervalMs int64
	lastUpdate int64
	bytes      uint64
}

const (
	DATA_STREAM_PACKET_BASE_SIZE = 16
)

type DataStreamChunk struct {
	data []byte
	size int
}

// Status for a streaming session
type WS_Streaming_Session struct {
	id uint64

	server       *WS_Streaming_Server
	conn         *websocket.Conn
	ip           string
	mutex        *sync.Mutex
	publishMutex *sync.Mutex
	closed       bool

	channel  string
	key      string
	streamId string

	isPublishing bool
	isPlaying    bool
	isProbing    bool
	isIdling     bool
	chunksCount  int64

	gopCache         *list.List
	gopCacheSize     int64
	gopCacheLimit    int64
	gopCacheDisabled bool
	gopPlayNo        bool
	gopPlayClear     bool

	bitRate      uint64
	bitRateCache BitRateCache
}

// Handles incoming connection
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
	}

	// Create session

	conn, err := server.wsUpgrader.Upgrade(w, req, nil)

	if err != nil {
		LogError(err)
		conn.Close()
		server.RemoveIP(ip)
		return
	}

	session := WS_Streaming_Session{
		id:               sessionId,
		server:           server,
		ip:               ip,
		conn:             nil,
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
	session.Run()
}

func (session *WS_Streaming_Session) SendText(txt string) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	return session.conn.WriteMessage(websocket.TextMessage, []byte(txt))
}

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

func (session *WS_Streaming_Session) SendHeartBeatMessages() {
	for {
		time.Sleep(20 * time.Second)

		err := session.SendText("h")

		if err != nil {
			return
		}
	}
}

func (session *WS_Streaming_Session) Kill() {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.conn.Close()
}

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

func (session *WS_Streaming_Session) log(str string) {
	LogRequest(session.id, session.ip, str)
}

func (session *WS_Streaming_Session) debug(str string) {
	LogDebugSession(session.id, session.ip, str)
}

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

func (session *WS_Streaming_Session) StartPlayer(player *WS_Streaming_Session) {
	session.publishMutex.Lock()
	defer session.publishMutex.Unlock()

	if !session.isPublishing {
		player.isPlaying = false
		player.isIdling = true
		player.log("PLAY IDLE '" + player.channel + "'")
		return
	}

	player.log("PLAY START '" + player.channel + "'")

	if !player.gopPlayNo && session.gopCache.Len() > 0 {
		for t := session.gopCache.Front(); t != nil; t = t.Next() {
			chunks := t.Value
			switch x := chunks.(type) {
			case *DataStreamChunk:
				player.SendChunk(x.data)
			}
		}
	}

	player.isPlaying = true
	player.isIdling = false

	if player.gopPlayClear {
		session.gopCache = list.New()
		session.gopCacheSize = 0
		session.gopCacheDisabled = true
	}
}

func (session *WS_Streaming_Session) StartIdlePlayers() {
	session.publishMutex.Lock()
	defer session.publishMutex.Unlock()

	// Start idle players
	idlePlayers := session.server.GetIdlePlayers(session.channel)

	for i := 0; i < len(idlePlayers); i++ {
		player := idlePlayers[i]
		if subtle.ConstantTimeCompare([]byte(session.key), []byte(player.key)) == 1 {
			player.log("PLAY START '" + player.channel + "'")

			if !player.gopPlayNo && session.gopCache.Len() > 0 {
				for t := session.gopCache.Front(); t != nil; t = t.Next() {
					chunks := t.Value
					switch x := chunks.(type) {
					case *DataStreamChunk:
						player.SendChunk(x.data)
					}
				}
			}

			player.isPlaying = true
			player.isIdling = false

			if player.gopPlayClear {
				session.gopCache = list.New()
				session.gopCacheSize = 0
				session.gopCacheDisabled = true
			}
		} else {
			player.log("Error: Invalid stream key provided")
			player.SendText("ERROR: Invalid streaming key")
			player.Kill()
		}
	}
}

func (session *WS_Streaming_Session) HandleChunk(data []byte) {
	chunkLength := len(data)
	chunk := DataStreamChunk{
		data: data,
		size: chunkLength,
	}

	session.publishMutex.Lock()
	defer session.publishMutex.Unlock()

	if !session.isPublishing {
		return
	}

	// GOP cache

	if !session.gopCacheDisabled {
		session.gopCache.PushBack(&chunk)
		session.gopCacheSize += int64(chunkLength) + DATA_STREAM_PACKET_BASE_SIZE

		for session.gopCacheSize > session.gopCacheLimit {
			toDelete := session.gopCache.Front()
			v := toDelete.Value
			switch x := v.(type) {
			case *DataStreamChunk:
				session.gopCacheSize -= int64(x.size)
			}
			session.gopCache.Remove(toDelete)
			session.gopCacheSize -= DATA_STREAM_PACKET_BASE_SIZE
		}
	}

	// Players

	players := session.server.GetPlayers(session.channel)

	for i := 0; i < len(players); i++ {
		if players[i].isPlaying {
			players[i].SendChunk(data)
		}
	}
}

func (session *WS_Streaming_Session) EndPublish() {
	session.publishMutex.Lock()
	defer session.publishMutex.Unlock()

	if session.isPublishing {

		session.log("PUBLISH END '" + session.channel + "'")

		players := session.server.GetPlayers(session.channel)

		for i := 0; i < len(players); i++ {
			players[i].isIdling = true
			players[i].isPlaying = false
			players[i].log("PLAY END '" + players[i].channel + "'")
			players[i].Kill()
		}

		session.server.RemovePublisher(session.channel)

		session.gopCache = list.New()

		session.isPublishing = false

		// Send event
		if session.server.controlConnection.PublishEnd(session.channel, session.streamId) {
			session.debug("Stop event sent")
		} else {
			session.debug("Could not send stop event")
		}
	}
}
