// Control server connection

package main

import (
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

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

	externalIP := os.Getenv("EXTERNAL_IP")

	if externalIP != "" {
		headers.Set("x-external-ip", externalIP)
	}

	externalPort := os.Getenv("EXTERNAL_PORT")

	if externalPort != "" {
		headers.Set("x-custom-port", externalPort)
	}

	useSSL := os.Getenv("EXTERNAL_SSL")

	if useSSL == "YES" {
		headers.Set("x-ssl-use", "true")
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

	// After a connection is established, any previous encoding tasks must be stopped
	// TODO

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
func (c *ControlServerConnection) Send(msg WebsocketMessage) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.connection == nil {
		return false
	}

	c.connection.WriteMessage(websocket.TextMessage, []byte(msg.serialize()))

	if LOG_DEBUG_ENABLED {
		LogDebug("[WS-CONTROL] >>>\n" + string(msg.serialize()))
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
			c.OnDisconnect(err)
			return
		}

		_, message, err := conn.ReadMessage()

		if err != nil {
			c.OnDisconnect(err)
			return
		}

		msgStr := string(message)

		if LOG_DEBUG_ENABLED {
			LogDebug("[WS-CONTROL] <<<\n" + msgStr)
		}

		msg := parseWebsocketMessage(msgStr)

		c.ParseIncomingMessage(&msg)
	}
}

// Parses an incoming message
// msg - Received parsed message
func (c *ControlServerConnection) ParseIncomingMessage(msg *WebsocketMessage) {
	switch msg.method {
	case "ERROR":
		LogErrorMessage("[WS-CONTROL] Remote error. Code=" + msg.params["error-code"] + " / Details: " + msg.params["error-message"])
	}
}

// Sends heart-beat messages to keep the connection alive
func (c *ControlServerConnection) RunHeartBeatLoop() {
	for {
		time.Sleep(20 * time.Second)

		// Send heartbeat message
		heartbeatMessage := WebsocketMessage{
			method: "HEARTBEAT",
			params: nil,
			body:   "",
		}

		c.Send(heartbeatMessage)
	}
}
