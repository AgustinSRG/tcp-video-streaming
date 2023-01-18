// Control server connection

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ControlServerConnection struct {
	server        *RTMPServer
	connectionURL string
	connection    *websocket.Conn
	lock          *sync.Mutex
	nextRequestId uint64
	enabled       bool

	requests map[string]*ControlServerPendingRequest
}

type ControlServerPendingRequest struct {
	waiter chan PublishResponse
}

type PublishResponse struct {
	accepted bool
	streamId string
}

// Initializes connection
func (c *ControlServerConnection) Initialize(server *RTMPServer) {
	c.server = server
	c.lock = &sync.Mutex{}
	c.nextRequestId = 0
	c.requests = make(map[string]*ControlServerPendingRequest)

	baseURL := os.Getenv("CONTROL_BASE_URL")

	if baseURL == "" {
		LogWarning("CONTROL_BASE_URL not provided. The server will run in stand-alone mode.")
		c.enabled = false
		return
	}

	connectionURL, err := url.Parse(baseURL)
	if err != nil {
		LogError(err)
		LogWarning("CONTROL_BASE_URL not provided. The server will run in stand-alone mode.")
		c.enabled = false
		return
	}
	pathURL, err := url.Parse("/ws/control/rtmp")
	if err != nil {
		LogError(err)
		LogWarning("CONTROL_BASE_URL not provided. The server will run in stand-alone mode.")
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

	conn, _, err := websocket.DefaultDialer.Dial(c.connectionURL, headers)

	if err != nil {
		c.lock.Unlock()
		LogErrorMessage("[WS-CONTROL] Connection error: " + err.Error())
		go c.Reconnect()
		return
	}

	c.connection = conn

	c.lock.Unlock()

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
	case "PUBLISH-ACCEPT":
		c.OnPublishAccept(msg.params["request-id"], msg.params["stream-id"])
	case "PUBLISH-DENY":
		c.OnPublishDeny(msg.params["request-id"])
	case "STREAM-KILL":
		c.OnPublishDeny(msg.params["request-id"])
	}
}

func (c *ControlServerConnection) OnPublishAccept(requestId string, streamId string) {
	c.lock.Lock()
	req := c.requests[requestId]
	c.lock.Unlock()

	if req == nil {
		return
	}

	res := PublishResponse{
		accepted: true,
		streamId: streamId,
	}

	req.waiter <- res
}

func (c *ControlServerConnection) OnPublishDeny(requestId string) {
	c.lock.Lock()
	req := c.requests[requestId]
	c.lock.Unlock()

	if req == nil {
		return
	}

	res := PublishResponse{
		accepted: false,
		streamId: "",
	}

	req.waiter <- res
}

func (c *ControlServerConnection) OnStreamKill(channel string, streamId string) {
	if streamId == "*" || streamId == "" {
		publisher := c.server.GetPublisher(channel)

		if publisher != nil {
			publisher.Kill()
		}
	} else {
		publisher := c.server.GetPublisher(channel)

		if publisher != nil && publisher.stream_id == streamId {
			publisher.Kill()
		}
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

// Requests publishing to the coordinator server
// channel - RTMP channel ID
// key - Publishing key
// userIP - IP address of the user
// Returns:
//    - accepted - True if the key was accepted
//    - streamId - Contains the Stream ID if accepted
// This method waits for the server to return a response
func (c *ControlServerConnection) RequestPublish(channel string, key string, userIP string) (accepted bool, streamId string) {
	if !c.enabled {
		return true, ""
	}

	requestId := fmt.Sprint(c.GetNextRequestId())

	request := ControlServerPendingRequest{
		waiter: make(chan PublishResponse),
	}

	msgParams := make(map[string]string)

	msgParams["Request-ID"] = requestId
	msgParams["Stream-Channel"] = channel
	msgParams["Stream-Key"] = key
	msgParams["User-IP"] = userIP

	msg := WebsocketMessage{
		method: "PUBLISH-REQUEST",
		params: msgParams,
		body:   "",
	}

	c.lock.Lock()
	c.requests[requestId] = &request
	c.lock.Unlock()

	success := c.Send(msg)

	if !success {
		c.lock.Lock()
		delete(c.requests, requestId)
		c.lock.Unlock()

		return false, ""
	}

	time.AfterFunc(20*time.Second, func() { request.waiter <- PublishResponse{accepted: false, streamId: ""} }) // Timeout

	res := <-request.waiter // Wait

	c.lock.Lock()
	delete(c.requests, requestId)
	c.lock.Unlock()

	return res.accepted, res.streamId
}

// Send Publish-End message to the coordinator server
// channel - Streaming channel
// streamId - Streaming session ID
func (c *ControlServerConnection) PublishEnd(channel string, streamId string) bool {
	msgParams := make(map[string]string)

	msgParams["Stream-Channel"] = channel
	msgParams["Stream-ID"] = streamId

	msg := WebsocketMessage{
		method: "PUBLISH-END",
		params: msgParams,
		body:   "",
	}

	return c.Send(msg)
}
