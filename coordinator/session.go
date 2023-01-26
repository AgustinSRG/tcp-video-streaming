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

	mutex *sync.Mutex // Mutex to control access to the session status data

	closed bool // True if the connection is closed

	sessionType int
}

func CreateSession(server *Streaming_Coordinator_Server, conn *websocket.Conn, id uint64, ip string, sessionType int) *ControlSession {
	session := ControlSession{
		server:      server,
		conn:        conn,
		id:          id,
		ip:          ip,
		mutex:       &sync.Mutex{},
		closed:      false,
		sessionType: sessionType,
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

func (session *ControlSession) HandleMessage(msg WebsocketMessage) {

}
