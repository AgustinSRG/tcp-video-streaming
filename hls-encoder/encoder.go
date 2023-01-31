// HLS encoder server

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
)

// Stores the status data of the HLS encoder
type HLS_Encoder_Server struct {
	websocketControlConnection *ControlServerConnection // Connection to the coordinator server

	capacity int // Server capacity
	load     int // Server load

	mutex *sync.Mutex // Mutex to access the status data

	loopBackPort int // Port of the loopback HTTP listener (randomly chosen)

	tasks map[string]*EncodingTask // List of active encoding tasks
}

// Encoding task data
type EncodingTask struct {
	channel  string
	streamId string
}

// Initializes the encoder
func (server *HLS_Encoder_Server) Initialize() {
	server.mutex = &sync.Mutex{}

	server.load = 0

	server.capacity = -1

	customCapacity := os.Getenv("SERVER_CAPACITY")
	if customCapacity != "" {
		cap, e := strconv.Atoi(customCapacity)
		if e == nil {
			server.capacity = cap
		}
	}

	server.tasks = make(map[string]*EncodingTask)

	server.websocketControlConnection = &ControlServerConnection{}
}

// Starts all services
func (server *HLS_Encoder_Server) Start() {
	// Setup loopback server
	loopBackServer := &http.Server{Addr: "127.0.0.1:0", Handler: server}

	ln, err := net.Listen("tcp", loopBackServer.Addr)
	if err != nil {
		LogErrorMessage("Fatal: Could not setup loopback server")
		LogError(err)
		os.Exit(1)
		return
	}

	server.loopBackPort = ln.Addr().(*net.TCPAddr).Port

	LogInfo("Loopback server listening on port " + fmt.Sprint(server.loopBackPort))

	// Start connection with the control server
	server.websocketControlConnection.Initialize(server)

	err = loopBackServer.Serve(ln)

	if err != nil {
		LogError(err)
	}
}

func (server *HLS_Encoder_Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)
	fmt.Fprintf(w, "HLS encoding server - Version "+VERSION)
}
