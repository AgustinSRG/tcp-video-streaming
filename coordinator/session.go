// Websocket session

package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	messages "github.com/AgustinSRG/tcp-video-streaming/common/message"
	"github.com/gorilla/websocket"
)

const (
	SESSION_TYPE_RTMP = 1
	SESSION_TYPE_WSS  = 2
	SESSION_TYPE_HLS  = 3
)

// Websocket session for control
type ControlSession struct {
	server *Streaming_Coordinator_Server // Reference to the server

	conn *websocket.Conn // Websocket connection

	id uint64 // Session ID
	ip string // IP address of the client

	externalIP   string // External IP of the streaming server
	externalPort int    // External port of the streaming server
	usesSSL      bool   // True if the streaming server uses SSL

	mutex *sync.Mutex // Mutex to control access to the session status data

	closed bool // True if the connection is closed

	sessionType int // Type of control session

	associatedChannels map[string]bool // List of associated channels

	encoderRegistered bool // True if the encoder was registered
}

// Creates a session
// server - Reference to the server
// conn - Websocket connection
// id - Session ID
// ip - Client IP
// sessionType - Type of control session
func CreateSession(server *Streaming_Coordinator_Server, conn *websocket.Conn, id uint64, ip string, sessionType int) *ControlSession {
	session := ControlSession{
		server:             server,
		conn:               conn,
		id:                 id,
		ip:                 ip,
		mutex:              &sync.Mutex{},
		closed:             false,
		sessionType:        sessionType,
		externalIP:         ip,
		externalPort:       0,
		usesSSL:            false,
		associatedChannels: make(map[string]bool),
		encoderRegistered:  false,
	}

	switch sessionType {
	case SESSION_TYPE_RTMP:
		session.externalPort = 1935
	case SESSION_TYPE_WSS:
		session.externalPort = 80
	}

	return &session
}

// Logs a message for this connection
// str - message to log
func (session *ControlSession) log(str string) {
	LogRequest(session.id, session.ip, str)
}

// Logs a debug message for this connection
// str - message to log
func (session *ControlSession) debug(str string) {
	LogDebugSession(session.id, session.ip, str)
}

// Sends a text message to the client
// txt - Message contents
func (session *ControlSession) Send(msg messages.WebsocketMessage) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	if LOG_DEBUG_ENABLED {
		session.debug(">>> \n" + msg.Serialize())
	}

	return session.conn.WriteMessage(websocket.TextMessage, []byte(msg.Serialize()))
}

// Periodically sends heartbeat messages to the client
// Loops until the connection is closed
func (session *ControlSession) SendHeartBeatMessages() {
	for !session.closed {
		time.Sleep(20 * time.Second)

		// Send heartbeat message
		heartbeatMessage := messages.WebsocketMessage{
			Method: "HEARTBEAT",
		}

		err := session.Send(heartbeatMessage)

		if err != nil {
			return
		}
	}
}

// Closes the connection
func (session *ControlSession) Kill() {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.conn.Close()
}

// Runs the connection, reading the incoming messages until the connection is closed
func (session *ControlSession) Run() {
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
		// session.onClose()
		// Remove connection
		session.server.RemoveSession(session.id)
	}()

	// Heartbeat
	go session.SendHeartBeatMessages()

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

			msg := messages.ParseWebsocketMessage(msgStr)

			session.HandleMessage(msg)
		}
	}
}

// Handles incoming websocket message
// msg - The message
func (session *ControlSession) HandleMessage(msg messages.WebsocketMessage) {
	switch msg.Method {
	case "ERROR":
		session.log("ERROR / CODE=" + msg.GetParam("Error-Code") + " / MSG=" + msg.GetParam("Error-Message"))
	case "PUBLISH-REQUEST":
		session.HandlePublishRequest(msg.GetParam("Request-ID"), msg.GetParam("Stream-Channel"), msg.GetParam("Stream-Key"), msg.GetParam("User-IP"))
	case "PUBLISH-END":
		session.HandlePublishEnd(msg.GetParam("Stream-Channel"), msg.GetParam("Stream-ID"))
	case "REGISTER":
		capacity, err := strconv.ParseInt(msg.GetParam("Capacity"), 10, 32)

		if err != nil {
			LogError(err)
			return
		}

		session.HandleEncoderRegister(int(capacity))
	case "STREAM-AVAILABLE":
		session.HandleStreamAvailable(msg.GetParam("Stream-Channel"), msg.GetParam("Stream-ID"), msg.GetParam("Stream-Type"), msg.GetParam("Resolution"), msg.GetParam("Index-file"))
	case "STREAM-CLOSED":
		session.HandleStreamClosed(msg.GetParam("Stream-Channel"), msg.GetParam("Stream-ID"))
	}
}

// Generates the URL for a streaming server
// channel - The channel
// key - The streaming key
// Returns the URL
func (session *ControlSession) GeneratePublishSourceURL(channel string, key string) string {
	switch session.sessionType {
	case SESSION_TYPE_RTMP:
		if session.usesSSL {
			return "rtmps://" + session.externalIP + ":" + fmt.Sprint(session.externalPort) + "/" + channel + "/" + key
		} else {
			return "rtmp://" + session.externalIP + ":" + fmt.Sprint(session.externalPort) + "/" + channel + "/" + key
		}
	case SESSION_TYPE_WSS:
		if session.usesSSL {
			return "wss://" + session.externalIP + ":" + fmt.Sprint(session.externalPort) + "/" + channel + "/" + key
		} else {
			return "ws://" + session.externalIP + ":" + fmt.Sprint(session.externalPort) + "/" + channel + "/" + key
		}
	default:
		return ""
	}
}

// Associates a channel to the session
// channel - The channel
func (session *ControlSession) AssociateChannel(channel string) {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.associatedChannels[channel] = true
}

// Disassociates a channel from the session
// channel - The channel
func (session *ControlSession) DisassociateChannel(channel string) {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	delete(session.associatedChannels, channel)
}

// Gets the list of associated channels
// Returns the list of channel IDs
func (session *ControlSession) GetAssociatedChannels() []string {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	result := make([]string, 0)

	for channel := range session.associatedChannels {
		result = append(result, channel)
	}

	return result
}
