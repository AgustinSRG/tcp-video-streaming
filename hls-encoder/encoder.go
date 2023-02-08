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

	tasks map[string]*EncodingTask // List of active encoding tasks. Map (channel:streamId) -> Task
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
	LogDebug(req.Method + " " + req.RequestURI)

	if req.Method == "PUT" {
		w.WriteHeader(200)
		return
	}

	w.WriteHeader(200)
	fmt.Fprintf(w, "HLS encoding server - Version "+VERSION)
}

// Kills every single active task
// Run only if the connection to the coordinator server is lost
func (server *HLS_Encoder_Server) KillAllActiveTasks() {
	tasksToKill := make([]*EncodingTask, 0)

	server.mutex.Lock()

	for _, task := range server.tasks {
		tasksToKill = append(tasksToKill, task)
	}

	server.mutex.Unlock()

	for i := 0; i < len(tasksToKill); i++ {
		tasksToKill[i].Kill()
	}
}

// Creates a new encoding task
// channel - Channel ID
// streamId - Stream ID
// sourceType - Source type. Can be: RTMP or WS
// sourceURI - Source URI. With the host, port, stream channel and key
// resolutions - List of resolutions to resize the video stream
// record - True if recording is enabled
// previews - Configuration for making stream previews
func (server *HLS_Encoder_Server) CreateTask(channel string, streamId string, sourceType string, sourceURI string, resolutions ResolutionList, record bool, previews PreviewsConfiguration) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	taskId := channel + ":" + streamId

	if server.tasks[taskId] != nil {
		return // Already created
	}

	newTask := &EncodingTask{
		server:      server,
		channel:     channel,
		streamId:    streamId,
		sourceType:  sourceType,
		sourceURI:   sourceURI,
		resolutions: resolutions,
		record:      record,
		previews:    previews,
		killed:      false,
		mutex:       &sync.Mutex{},
		process:     nil,
		subStreams:  make(map[string]*SubStreamStatus),
	}

	server.tasks[taskId] = newTask
	server.load++

	LogTaskStatus(channel, streamId, "Task created | Server load: "+fmt.Sprint(server.load))
	if LOG_DEBUG_ENABLED {
		LogDebugTask(channel, streamId, "Task details: sourceType="+sourceType+" | sourceURI="+sourceURI+" | resolutions="+resolutions.Encode()+" | record="+fmt.Sprint(record)+" | previews="+previews.Encode("-"))
	}

	go newTask.Run()
}

// Removes an encoding task
// Call only after it has finished
// channel - Channel ID
// streamId - Stream ID
func (server *HLS_Encoder_Server) RemoveTask(channel string, streamId string) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	taskId := channel + ":" + streamId

	if server.tasks[taskId] != nil {
		delete(server.tasks, taskId)
		server.load--
		LogTaskStatus(channel, streamId, "Task removed | Server load: "+fmt.Sprint(server.load))
	}
}

// Finds an encoding task
// channel - Channel ID
// streamId - Stream ID
// Returns a reference to the task, or nil
func (server *HLS_Encoder_Server) GetTask(channel string, streamId string) *EncodingTask {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	return server.tasks[channel+":"+streamId]
}
