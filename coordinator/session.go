// Websocket session

package main

import (
	"sync"
	"time"

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
}

// Creates a session
// server - Reference to the server
// conn - Websocket connection
// id - Session ID
// ip - Client IP
// sessionType - Type of control session
func CreateSession(server *Streaming_Coordinator_Server, conn *websocket.Conn, id uint64, ip string, sessionType int) *ControlSession {
	session := ControlSession{
		server:       server,
		conn:         conn,
		id:           id,
		ip:           ip,
		mutex:        &sync.Mutex{},
		closed:       false,
		sessionType:  sessionType,
		externalIP:   ip,
		externalPort: 0,
		usesSSL:      false,
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
func (session *ControlSession) Send(msg WebsocketMessage) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	if LOG_DEBUG_ENABLED {
		session.debug(">>> \n" + msg.serialize())
	}

	return session.conn.WriteMessage(websocket.TextMessage, []byte(msg.serialize()))
}

// Periodically sends heartbeat messages to the client
// Loops until the connection is closed
func (session *ControlSession) SendHeartBeatMessages() {
	for !session.closed {
		time.Sleep(20 * time.Second)

		// Send heartbeat message
		heartbeatMessage := WebsocketMessage{
			method: "HEARTBEAT",
			params: nil,
			body:   "",
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

			msg := parseWebsocketMessage(msgStr)

			session.HandleMessage(msg)
		}
	}
}

// Handles incoming websocket message
// msg - The message
func (session *ControlSession) HandleMessage(msg WebsocketMessage) {
	switch msg.method {
	case "ERROR":
		session.log("ERROR / CODE=" + msg.GetParam("Error-Code") + " / MSG=" + msg.GetParam("Error-Message"))
	case "PUBLISH-REQUEST":
		session.HandlePublishRequest(msg.GetParam("Request-ID"), msg.GetParam("Stream-Channel"), msg.GetParam("Stream-Key"), msg.GetParam("User-IP"))
	case "PUBLISH-END":
		session.HandlePublishEnd(msg.GetParam("Stream-Channel"), msg.GetParam("Stream-ID"))
	}
}

func (session *ControlSession) HandlePublishRequest(requestId string, channel string, key string, ip string) {
	if !validateStreamIDString(channel) {
		session.SendPublishDeny(requestId, channel)
		return
	}

	if !validateStreamIDString(key) {
		session.SendPublishDeny(requestId, channel)
		return
	}

	if !ValidateStreamKey(channel, key, ip) {
		session.SendPublishDeny(requestId, channel)
		return
	}

	streamId := session.server.coordinator.GenerateStreamID()

	// Change coordinator status data

	channelData := session.server.coordinator.AcquireChannel(channel)

	if !channelData.closed {
		// Already publishing
		session.server.coordinator.ReleaseChannel(channelData)
		session.SendPublishDeny(requestId, channel)
		return
	}

	channelData.closed = false
	channelData.publisher = session.id
	switch session.sessionType {
	case SESSION_TYPE_RTMP:
		channelData.publishMethod = PUBLISH_METHOD_RTMP
	case SESSION_TYPE_WSS:
		channelData.publishMethod = PUBLISH_METHOD_WS
	default:
		channelData.closed = true
		session.server.coordinator.ReleaseChannel(channelData)
		session.SendPublishDeny(requestId, channel)
		return
	}

	// Find an encoder and assign it (TODO)

	// Release channel data
	session.server.coordinator.ReleaseChannel(channelData)

	// Accepted
	session.SendPublishAccept(requestId, channel, streamId)
}

func (session *ControlSession) HandlePublishEnd(channel string, streamId string) {
	channelData := session.server.coordinator.AcquireChannel(channel)

	if channelData.closed {
		session.server.coordinator.ReleaseChannel(channelData)
		return
	}

	channelData.closed = true

	// Find encoder and notice it
	encoderId := channelData.encoder
	encoderSession := session.server.GetSession(encoderId)

	if encoderSession != nil {
		encoderSession.SendEncodeStop(channel, streamId)
	}

	session.server.coordinator.ReleaseChannel(channelData)
}

func (session *ControlSession) SendPublishDeny(requestId string, channel string) {
	params := make(map[string]string)

	params["Request-ID"] = requestId
	params["Stream-Channel"] = channel

	msg := WebsocketMessage{
		method: "PUBLISH-DENY",
		params: params,
		body:   "",
	}

	err := session.Send(msg)

	if err != nil {
		LogError(err)
	}
}

func (session *ControlSession) SendPublishAccept(requestId string, channel string, streamId string) {
	params := make(map[string]string)

	params["Request-ID"] = requestId
	params["Stream-Channel"] = channel
	params["Stream-ID"] = streamId

	msg := WebsocketMessage{
		method: "PUBLISH-ACCEPT",
		params: params,
		body:   "",
	}

	err := session.Send(msg)

	if err != nil {
		LogError(err)
	}
}

func (session *ControlSession) SendStreamKill(channel string, streamId string) {
	params := make(map[string]string)

	params["Stream-Channel"] = channel
	params["Stream-ID"] = streamId

	msg := WebsocketMessage{
		method: "STREAM-KILL",
		params: params,
		body:   "",
	}

	err := session.Send(msg)

	if err != nil {
		LogError(err)
	}
}

func (session *ControlSession) SendEncodeStop(channel string, streamId string) {
	params := make(map[string]string)

	params["Stream-Channel"] = channel
	params["Stream-ID"] = streamId

	msg := WebsocketMessage{
		method: "ENCODE-STOP",
		params: params,
		body:   "",
	}

	err := session.Send(msg)

	if err != nil {
		LogError(err)
	}
}
