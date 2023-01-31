// HLS encoder server

package main

import "sync"

// Stores the status data of the HLS encoder
type HLS_Encoder_Server struct {
	websocketControlConnection *ControlServerConnection // Connection to the coordinator server

	mutex *sync.Mutex // Mutex to access the status data
}

// Initializes the encoder
func (server *HLS_Encoder_Server) Initialize() {
	server.mutex = &sync.Mutex{}

	server.websocketControlConnection = &ControlServerConnection{}
}

// Starts all services
func (server *HLS_Encoder_Server) Start() {
	server.websocketControlConnection.Initialize(server)
}
