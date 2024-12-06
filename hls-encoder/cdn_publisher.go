// HLS Websocket CDN publisher

package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"time"

	client_publisher "github.com/AgustinSRG/hls-websocket-cdn/client-publisher"
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
	servers []*CdnServer

	// Secret to generate tokens to push
	pushSecret string

	// Mutex for the struct
	mu *sync.Mutex
}

// Creates instance of CdnPublishController
func NewCdnPublishController() *CdnPublishController {
	enabled := os.Getenv("HLS_WS_CDN_ENABLED") == "YES"

	pushSecret := os.Getenv("HLS_WS_CDN_PUSH_SECRET")

	serverList := make([]*CdnServer, 0)

	serverListStr := os.Getenv("HLS_WS_CDN_URL")

	if serverListStr != "" {
		serverListStrSplit := strings.Split(serverListStr, " ")

		for _, url := range serverListStrSplit {
			serverList = append(serverList, &CdnServer{
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
func (pc *CdnPublishController) GetServerUrl() string {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	availableServerList := make([]*CdnServer, 0)

	now := time.Now().UnixMilli()

	for _, s := range pc.servers {
		if now-s.lastTimeError <= CDN_SERVER_ERROR_WAIT_TIME {
			availableServerList = append(availableServerList, s)
		}
	}

	if len(availableServerList) == 1 {
		return availableServerList[1].url
	} else if len(availableServerList) > 1 {
		i := rand.IntN(len(availableServerList))
		return availableServerList[i].url
	}

	url := ""
	lastErrorTime := now

	for _, s := range pc.servers {
		if s.lastTimeError < lastErrorTime {
			url = s.url
			lastErrorTime = s.lastTimeError
		}
	}

	return url
}

// Reports server failure
func (pc *CdnPublishController) ReportServerFailure(url string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	for _, s := range pc.servers {
		if s.url == url {
			s.lastTimeError = time.Now().Unix()
		}
	}
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

	// Socket
	client *client_publisher.HlsWebSocketPublisher
}

// Creates CDN publisher
func (controller *CdnPublishController) CreateCdnPublisher(task *EncodingTask, subStream *SubStreamStatus) *CdnPublisher {
	if !controller.enabled {
		return nil
	}

	publisher := &CdnPublisher{
		mu:            &sync.Mutex{},
		controller:    controller,
		task:          task,
		subStream:     subStream,
		resolution:    subStream.resolution,
		fragmentsData: make(map[int][]byte),
		client: client_publisher.NewHlsWebSocketPublisher(client_publisher.HlsWebSocketPublisherConfiguration{
			GetServerUrl: controller.GetServerUrl,
			StreamId:     "hls/" + task.channel + "/" + task.streamId + "/" + subStream.resolution.Encode() + "/live.m3u8",
			AuthSecret:   controller.pushSecret,
			OnReady: func() {
				task.OnCdnConnectionReady(subStream)
			},
			OnError: func(url, msg string) {
				task.log("[CDN Publisher] [ERROR] [URL: " + url + "] " + msg)
				controller.ReportServerFailure(url)
			},
		}),
	}

	return publisher
}

// Stores fragment data
func (pub *CdnPublisher) StoreFragmentData(index int, data []byte) {
	pub.mu.Lock()
	defer pub.mu.Unlock()

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

	data := pub.fragmentsData[index]

	if data == nil {
		pub.log("[WARNING] Missing data of fragment " + fmt.Sprint(index))
		return
	}

	delete(pub.fragmentsData, index)

	if len(data) == 0 {
		return // Skip empty fragment
	}

	pub.client.SendFragment(duration, data)
}

// Closes connection to the CDN server
func (pub *CdnPublisher) Close() {
	pub.client.Close()
}
