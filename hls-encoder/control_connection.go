// Control server connection

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	messages "github.com/AgustinSRG/tcp-video-streaming/common/message"
	"github.com/gorilla/websocket"
)

// Status data of the connection with the coordinator server
type ControlServerConnection struct {
	server *HLS_Encoder_Server // Reference to the RTMP server

	connectionURL string          // Connection URL
	connection    *websocket.Conn // Websocket connection

	lock *sync.Mutex // Mutex to control access to this struct

	nextRequestId uint64 // ID for the next request ID

	requests map[string]*ControlServerPendingRequest // Pending requests. Map: ID -> Request status data

	enabled bool // True if the connection is enabled (will reconnect)
}

// Status data for a pending request
type ControlServerPendingRequest struct {
	waiter chan PublishResponse // Channel to wait for the response
}

// Response for a publish request
type PublishResponse struct {
	accepted bool   // True if accepted, false if denied
	streamId string // If accepted, the stream ID
}

// Initializes connection
// server - Reference to the HLS encoder server
func (c *ControlServerConnection) Initialize(server *HLS_Encoder_Server) {
	c.server = server
	c.lock = &sync.Mutex{}
	c.nextRequestId = 0
	c.requests = make(map[string]*ControlServerPendingRequest)

	baseURL := os.Getenv("CONTROL_BASE_URL")

	if baseURL == "" {
		LogWarning("CONTROL_BASE_URL not provided. The encoding server will not work if not connected to a coordinator server.")
		c.enabled = false
		return
	}

	connectionURL, err := url.Parse(baseURL)
	if err != nil {
		LogError(err)
		LogWarning("CONTROL_BASE_URL not provided. The encoding server will not work if not connected to a coordinator server.")
		c.enabled = false
		return
	}
	pathURL, err := url.Parse("/ws/control/hls")
	if err != nil {
		LogError(err)
		LogWarning("CONTROL_BASE_URL not provided. The encoding server will not work if not connected to a coordinator server.")
		c.enabled = false
		return
	}

	c.connectionURL = connectionURL.ResolveReference(pathURL).String()
	c.enabled = true

	go c.Connect()
	go c.RunHeartBeatLoop()
}

// Connect to the websocket server
func (c *ControlServerConnection) Connect() {
	c.lock.Lock()

	if c.connection != nil {
		c.lock.Unlock()
		return // Already connected
	}

	LogInfo("[WS-CONTROL] Connecting to " + c.connectionURL)

	headers := http.Header{}

	authToken := MakeWebsocketAuthenticationToken()

	if authToken != "" {
		headers.Set("x-control-auth-token", authToken)
	}

	conn, _, err := websocket.DefaultDialer.Dial(c.connectionURL, headers)

	if err != nil {
		c.lock.Unlock()
		LogErrorMessage("[WS-CONTROL] Connection error: " + err.Error())
		go c.Reconnect()
		return
	}

	c.connection = conn

	c.lock.Unlock()

	// Right after connecting, send the REGISTER message
	c.SendRegister(c.server.capacity)

	// After a connection is established, any previous encoding tasks must be stopped
	c.server.KillAllActiveTasks()

	go c.RunReaderLoop(conn)
}

// Waits 10 seconds and reconnects
func (c *ControlServerConnection) Reconnect() {
	LogInfo("[WS-CONTROL] Waiting 10 seconds to reconnect.")
	time.Sleep(10 * time.Second)
	c.Connect()
}

// Called when disconnected
// err - Disconnection error
func (c *ControlServerConnection) OnDisconnect(err error) {
	c.lock.Lock()
	c.connection = nil
	LogInfo("[WS-CONTROL] Disconnected: " + err.Error())
	c.lock.Unlock()

	go c.Connect() // Reconnect
}

// Sends a message
// msg - The message
// Returns true if the message was successfully sent
func (c *ControlServerConnection) Send(msg messages.WebsocketMessage) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.connection == nil {
		return false
	}

	c.connection.WriteMessage(websocket.TextMessage, []byte(msg.Serialize()))

	if LOG_DEBUG_ENABLED {
		LogDebug("[WS-CONTROL] >>>\n" + string(msg.Serialize()))
	}

	return true
}

// Generates a new request-id
func (c *ControlServerConnection) GetNextRequestId() uint64 {
	c.lock.Lock()
	defer c.lock.Unlock()

	requestId := c.nextRequestId

	c.nextRequestId++

	return requestId
}

// Reads messages until the connection is finished
// conn - Websocket connection
func (c *ControlServerConnection) RunReaderLoop(conn *websocket.Conn) {
	for {
		err := conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		if err != nil {
			conn.Close()
			c.OnDisconnect(err)
			return
		}

		_, message, err := conn.ReadMessage()

		if err != nil {
			conn.Close()
			c.OnDisconnect(err)
			return
		}

		msgStr := string(message)

		if LOG_DEBUG_ENABLED {
			LogDebug("[WS-CONTROL] <<<\n" + msgStr)
		}

		msg := messages.ParseWebsocketMessage(msgStr)

		c.ParseIncomingMessage(&msg)
	}
}

// Parses an incoming message
// msg - Received parsed message
func (c *ControlServerConnection) ParseIncomingMessage(msg *messages.WebsocketMessage) {
	switch msg.Method {
	case "ERROR":
		LogErrorMessage("[WS-CONTROL] Remote error. Code=" + msg.GetParam("Error-Code") + " / Details: " + msg.GetParam("Error-Message"))
	case "ENCODE-START":
		c.ReceiveEncodeStart(msg.GetParam("Stream-Channel"), msg.GetParam("Stream-ID"), msg.GetParam("Stream-Source-Type"), msg.GetParam("Stream-Source-URI"), DecodeResolutionsList(msg.GetParam("Resolutions")), strings.ToLower(msg.GetParam("Record")) == "true", DecodePreviewsConfiguration(msg.GetParam("Previews"), ","))
	case "ENCODE-STOP":
		c.ReceiveEncodeStop(msg.GetParam("Stream-Channel"), msg.GetParam("Stream-ID"))
	}
}

// Sends heart-beat messages to keep the connection alive
func (c *ControlServerConnection) RunHeartBeatLoop() {
	for {
		time.Sleep(20 * time.Second)

		// Send heartbeat message
		heartbeatMessage := messages.WebsocketMessage{
			Method: "HEARTBEAT",
		}

		c.Send(heartbeatMessage)
	}
}

// Sends REGISTER message
// capacity - Server capacity
func (c *ControlServerConnection) SendRegister(capacity int) bool {
	msgParams := make(map[string]string)

	msgParams["Capacity"] = fmt.Sprint(capacity)

	msg := messages.WebsocketMessage{
		Method: "REGISTER",
		Params: msgParams,
	}

	return c.Send(msg)
}

// Sends STREAM-AVAILABLE message
// channel - Channel ID
// streamId - Stream ID
// streamType - Sub-stream type. Can be HLS-LIVE, HLS-VOD or IMG-PREVIEW
// resolution - Video resolution
// indexFile - Stream index file
func (c *ControlServerConnection) SendStreamAvailable(channel string, streamId string, streamType string, resolution Resolution, indexFile string) bool {
	msgParams := make(map[string]string)

	msgParams["Stream-Channel"] = channel
	msgParams["Stream-ID"] = streamId
	msgParams["Stream-Type"] = streamType
	msgParams["Resolution"] = resolution.Encode()
	msgParams["Index-file"] = indexFile

	msg := messages.WebsocketMessage{
		Method: "STREAM-AVAILABLE",
		Params: msgParams,
	}

	return c.Send(msg)
}

// Sends STREAM-CLOSED message
// channel - Channel ID
// streamId - Stream ID
func (c *ControlServerConnection) SendStreamClosed(channel string, streamId string) bool {
	msgParams := make(map[string]string)

	msgParams["Stream-Channel"] = channel
	msgParams["Stream-ID"] = streamId

	msg := messages.WebsocketMessage{
		Method: "STREAM-CLOSED",
		Params: msgParams,
	}

	return c.Send(msg)
}

// Receives an ENCODE-START message
// channel - Channel ID
// streamId - Stream ID
// sourceType - Source type. Can be: RTMP or WS
// sourceURI - Source URI. With the host, port, stream channel and key
// resolutions - List of resolutions to resize the video stream
// record - True if recording is enabled
// previews - Configuration for making stream previews
func (c *ControlServerConnection) ReceiveEncodeStart(channel string, streamId string, sourceType string, sourceURI string, resolutions ResolutionList, record bool, previews PreviewsConfiguration) {
	c.server.CreateTask(channel, streamId, sourceType, sourceURI, resolutions, record, previews)
}

// Receives an ENCODE-STOP message
// channel - Channel ID
// streamId - Stream ID
func (c *ControlServerConnection) ReceiveEncodeStop(channel string, streamId string) {
	task := c.server.GetTask(channel, streamId)

	if task != nil {
		LogTaskStatus(channel, streamId, "Killing task...")
		task.Kill()
	}
}
