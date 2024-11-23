// HLS Websocket CDN publisher

package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

const HEARTBEAT_MSG_PERIOD_SECONDS = 30

// CDN server
type CdnServer struct {
	// Server URL
	url string

	// Last time the server failed
	lastTimeError int64
}

// CDN publish controller
type CdnPublishController struct {
	// Enabled?
	enabled bool

	// List of servers
	servers []CdnServer

	// Secret to generate tokens to push
	pushSecret string

	// Mutex for the struct
	mu *sync.Mutex
}

// Creates instance of CdnPublishController
func NewCdnPublishController() *CdnPublishController {
	enabled := os.Getenv("HLS_WS_CDN_ENABLED") == "YES"

	pushSecret := os.Getenv("HLS_WS_CDN_PUSH_SECRET")

	serverList := make([]CdnServer, 0)

	serverListStr := os.Getenv("HLS_WS_CDN_URL")

	if serverListStr != "" {
		serverListStrSplit := strings.Split(serverListStr, " ")

		for _, url := range serverListStrSplit {
			serverList = append(serverList, CdnServer{
				url:           url,
				lastTimeError: 0,
			})
		}
	}

	if enabled && len(serverList) == 0 {
		LogWarning("HLS_WS_CDN_ENABLED is set to YES, but no servers were provided (HLS_WS_CDN_URL). The HLS CDN publish service is disabled.")
	}

	return &CdnPublishController{
		enabled:    enabled,
		servers:    serverList,
		pushSecret: pushSecret,
		mu:         &sync.Mutex{},
	}
}

// Gets true if the CDN service is enabled
func (pc *CdnPublishController) IsEnabled() bool {
	return pc.enabled && len(pc.servers) > 0
}

const CDN_SERVER_ERROR_WAIT_TIME = 10 * 1000

// Gets the URL of a CDN server
func (pc *CdnPublishController) GetServerUrl() (string, *CdnServer) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	availableServerList := make([]*CdnServer, 0)

	now := time.Now().UnixMilli()

	for _, s := range pc.servers {
		if now-s.lastTimeError <= CDN_SERVER_ERROR_WAIT_TIME {
			availableServerList = append(availableServerList, &s)
		}
	}

	if len(availableServerList) == 1 {
		return availableServerList[1].url, availableServerList[1]
	} else if len(availableServerList) > 1 {
		i := rand.IntN(len(availableServerList))
		return availableServerList[i].url, availableServerList[1]
	}

	url := ""
	lastErrorTime := now
	var serverRef *CdnServer = nil

	for _, s := range pc.servers {
		if s.lastTimeError < lastErrorTime {
			url = s.url
			lastErrorTime = s.lastTimeError
			serverRef = &s
		}
	}

	return url, serverRef
}

// Reports server failure
func (pc *CdnPublishController) ReportServerFailure(s *CdnServer) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	s.lastTimeError = time.Now().Unix()
}

// Gets the authentication token for pushing the stream
func (pc *CdnPublishController) GetPushToken(streamId string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "PUSH:" + streamId,
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(pc.pushSecret))

	if err != nil {
		LogErrorMessage("Error signing token: " + err.Error())
	}

	return tokenString
}

// Fragment pending to be sent
type CdnPublisherPendingFragment struct {
	// Fragment duration
	Duration float32

	// Fragment data
	Data []byte
}

// CDN publisher
type CdnPublisher struct {
	// Mutex for the struct
	mu *sync.Mutex

	// Controller
	controller *CdnPublishController

	// Task
	task *EncodingTask

	// Status of the sub-stream
	subStream *SubStreamStatus

	// Resolution
	resolution Resolution

	// Map to store fragments data before sending
	fragmentsData map[int]([]byte)

	// Fragments queue
	fragmentsQueue []CdnPublisherPendingFragment

	// Socket
	socket *websocket.Conn

	// True if closed
	closed bool

	// True if ready
	ready bool

	// Channel to interrupt the heartbeat process
	heartbeatInterruptChannel chan bool
}

// Creates CDN publisher
func (controller *CdnPublishController) CreateCdnPublisher(task *EncodingTask, subStream *SubStreamStatus) *CdnPublisher {
	if !controller.enabled {
		return nil
	}

	publisher := &CdnPublisher{
		mu:                        &sync.Mutex{},
		controller:                controller,
		task:                      task,
		subStream:                 subStream,
		resolution:                subStream.resolution,
		fragmentsData:             make(map[int][]byte),
		fragmentsQueue:            make([]CdnPublisherPendingFragment, 0),
		socket:                    nil,
		closed:                    false,
		ready:                     false,
		heartbeatInterruptChannel: make(chan bool, 1),
	}

	return publisher
}

// Gets CDN stream ID given the task and the resolution
func (pub *CdnPublisher) GetCdnStreamId() string {
	return "hls/" + pub.task.channel + "/" + pub.task.streamId + "/" + pub.resolution.Encode() + "/live.m3u8"
}

func (pub *CdnPublisher) SendHeartbeatMessages() {
	heartbeatMessage := CdnWebsocketProtocolMessage{
		MessageType: "H",
	}

	for {
		select {
		case <-time.After(time.Duration(HEARTBEAT_MSG_PERIOD_SECONDS) * time.Second):
			pub.SendMessage(&heartbeatMessage)
		case <-pub.heartbeatInterruptChannel:
			return
		}
	}
}

// Send message
func (pub *CdnPublisher) SendMessage(msg *CdnWebsocketProtocolMessage) {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	if pub.closed || pub.socket == nil {
		return
	}

	pub.socket.WriteMessage(websocket.TextMessage, []byte(msg.Serialize()))
}

// Internal function to send the fragment message
func (pub *CdnPublisher) sendFragmentInternal(duration float32, data []byte) {
	msg := CdnWebsocketProtocolMessage{
		MessageType: "F",
		Parameters: map[string]string{
			"duration": fmt.Sprint(duration),
		},
	}

	pub.socket.WriteMessage(websocket.TextMessage, []byte(msg.Serialize()))
	pub.socket.WriteMessage(websocket.BinaryMessage, data)
}

// Stores fragment data
func (pub *CdnPublisher) StoreFragmentData(index int, data []byte) {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	if pub.closed {
		return
	}

	pub.fragmentsData[index] = data
}

// Logs message
func (pub *CdnPublisher) log(msg string) {
	pub.task.log("[CDN Publisher] [Resolution: " + pub.resolution.Encode() + "] " + msg)
}

// Send a fragment, or keeps it in the queue
func (pub *CdnPublisher) SendFragment(index int, duration float32) {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	if pub.closed {
		return
	}

	data := pub.fragmentsData[index]

	if data == nil {
		pub.log("[WARNING] Missing data of fragment " + fmt.Sprint(index))
		return
	}

	delete(pub.fragmentsData, index)

	if len(data) == 0 {
		return // Skip empty fragment
	}

	if pub.ready {
		pub.sendFragmentInternal(duration, data)
	} else {
		if len(pub.fragmentsQueue) >= pub.task.server.hlsLivePlayListSize && len(pub.fragmentsQueue) > 0 {
			pub.fragmentsQueue = append(pub.fragmentsQueue[1:], CdnPublisherPendingFragment{
				Duration: duration,
				Data:     data,
			})
		} else {
			pub.fragmentsQueue = append(pub.fragmentsQueue, CdnPublisherPendingFragment{
				Duration: duration,
				Data:     data,
			})
		}
	}
}

// Returns true if the publisher is closed
func (pub *CdnPublisher) IsClosed() bool {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	return pub.closed
}

// Called when the connection is opened
func (pub *CdnPublisher) OnConnected(socket *websocket.Conn) {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	if pub.closed {
		return
	}

	pub.socket = socket
}

// Closes connection to the CDN server
func (pub *CdnPublisher) Ready() {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	if pub.closed {
		return
	}

	pub.ready = true

	for _, f := range pub.fragmentsQueue {
		pub.sendFragmentInternal(f.Duration, f.Data)
	}

	pub.fragmentsQueue = make([]CdnPublisherPendingFragment, 0)
}

// Call when disconnected from the server
func (pub *CdnPublisher) OnDisconnected() {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	if pub.closed {
		return
	}

	pub.ready = false
	pub.socket = nil
}

// Closes connection to the CDN server
func (pub *CdnPublisher) Close() {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	if pub.closed {
		return
	}

	if pub.socket != nil {
		// Send close message
		closeMessage := CdnWebsocketProtocolMessage{
			MessageType: "CLOSE",
		}

		pub.socket.WriteMessage(websocket.TextMessage, []byte(closeMessage.Serialize()))

		// Close connection
		pub.socket.Close()
		pub.socket = nil
	}

	pub.closed = true
	pub.ready = false

	// Interrupt heartbeat
	pub.heartbeatInterruptChannel <- true
}

// Starts the publisher
func (pub *CdnPublisher) Start() {
	go pub.Run()
	go pub.SendHeartbeatMessages()
}

// Limit (in bytes) for text messages (to prevent DOS attacks)
const TEXT_MSG_READ_LIMIT = 1600

// Runs the publisher
func (pub *CdnPublisher) Run() {
	for !pub.IsClosed() {
		url, serverRef := pub.controller.GetServerUrl()

		socket, _, err := websocket.DefaultDialer.Dial(url, nil)

		if pub.IsClosed() {
			return
		}

		if err != nil {
			pub.log("Could not connect to " + url + " | " + err.Error())

			if serverRef != nil {
				pub.controller.ReportServerFailure(serverRef)
			}

			time.Sleep(1 * time.Second) // Wait a second to try again
			continue
		}

		// Connected, send authentication

		cdnStreamId := pub.GetCdnStreamId()

		authMessage := CdnWebsocketProtocolMessage{
			MessageType: "PUSH",
			Parameters: map[string]string{
				"stream": cdnStreamId,
				"auth":   pub.controller.GetPushToken(cdnStreamId),
			},
		}

		socket.WriteMessage(websocket.TextMessage, []byte(authMessage.Serialize()))

		// Connected

		pub.OnConnected(socket)

		var closedWithError = false

		// Read incoming messages

		for !pub.IsClosed() {
			err := socket.SetReadDeadline(time.Now().Add(HEARTBEAT_MSG_PERIOD_SECONDS * 2 * time.Second))

			if err != nil {
				if !pub.IsClosed() {
					pub.log("Error: " + err.Error())
				}
				break // Closed
			}

			socket.SetReadLimit(TEXT_MSG_READ_LIMIT)

			mt, message, err := socket.ReadMessage()

			if err != nil {
				if !pub.IsClosed() {
					pub.log("Error: " + err.Error())
				}
				break // Closed
			}

			if mt != websocket.TextMessage {
				continue
			}

			parsedMessage := ParseCdnWebsocketProtocolMessage(string(message))

			switch parsedMessage.MessageType {
			case "E":
				pub.log("Error from CDN. Code: " + parsedMessage.GetParameter("code") + ", Message: " + parsedMessage.GetParameter("message"))
				closedWithError = true
			case "OK":
				// Ready
				pub.Ready()
				pub.task.OnCdnConnectionReady(pub.subStream)
			}
		}

		pub.OnDisconnected()

		if closedWithError && serverRef != nil {
			pub.controller.ReportServerFailure(serverRef)
			time.Sleep(1 * time.Second) // Wait a second to try again
		}
	}
}
