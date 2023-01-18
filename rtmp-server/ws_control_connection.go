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

type ControlServerConnection struct {
	connectionURL string
	connection    *websocket.Conn
	lock          *sync.Mutex
	nextRequestId uint64
	enabled       bool
}

// Initializes connection
func (c *ControlServerConnection) Initialize() {
	c.lock = &sync.Mutex{}
	c.nextRequestId = 0

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
