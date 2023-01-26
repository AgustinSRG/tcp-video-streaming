// Websocket session

package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Websocket session for control
type ControlSession struct {
	server *Streaming_Coordinator_Server // Reference to the server

	conn *websocket.Conn // Websocket connection

	id uint64 // Session ID
	ip string // IP address of the client

	mutex *sync.Mutex // Mutex to control access to the session status data

	closed bool // True if the connection is closed
}
