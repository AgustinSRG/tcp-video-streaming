// Websocket streaming server

package main

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type WS_Streaming_Server struct {
	wsUpgrader        *websocket.Upgrader
	controlConnection *ControlServerConnection

	nextSessionId uint64
	sessionIdLock *sync.Mutex

	ipCount       map[string]uint32
	mutexIpCount  *sync.Mutex
	ipLimit       uint32
	gopCacheLimit int64

	mutex    *sync.Mutex
	sessions map[uint64]*WS_Streaming_Session
	channels map[string]*WS_Streaming_Channel

	staticTestHandler http.Handler
}

type WS_Streaming_Channel struct {
	channel       string
	key           string
	stream_id     string
	publisher     uint64
	is_publishing bool
	players       map[uint64]bool
}

func (server *WS_Streaming_Server) Initialize() {
	server.wsUpgrader = &websocket.Upgrader{}
	server.wsUpgrader.CheckOrigin = func(r *http.Request) bool { return true }

	server.nextSessionId = 0
	server.sessionIdLock = &sync.Mutex{}

	server.ipCount = make(map[string]uint32)
	server.mutexIpCount = &sync.Mutex{}

	server.controlConnection = &ControlServerConnection{}

	server.ipLimit = 4
	custom_ip_limit := os.Getenv("MAX_IP_CONCURRENT_CONNECTIONS")
	if custom_ip_limit != "" {
		cil, e := strconv.Atoi(custom_ip_limit)
		if e != nil {
			server.ipLimit = uint32(cil)
		}
	}

	server.gopCacheLimit = 256 * 1024 * 1024
	custom_gop_limit := os.Getenv("GOP_CACHE_SIZE_MB")
	if custom_gop_limit != "" {
		cgl, e := strconv.Atoi(custom_gop_limit)
		if e != nil {
			server.gopCacheLimit = int64(cgl) * 1024 * 1024
		}
	}

	server.mutex = &sync.Mutex{}
	server.sessions = make(map[uint64]*WS_Streaming_Session)
	server.channels = make(map[string]*WS_Streaming_Channel)

	testFrontStat, err := os.Stat("./test")

	if err == nil && testFrontStat.IsDir() && os.Getenv("DISABLE_TEST_CLIENT") != "YES" {
		server.staticTestHandler = http.FileServer(http.Dir("./test/"))
		LogInfo("Test client is available. To disable it, set DISABLE_TEST_CLIENT=YES")
	} else {
		server.staticTestHandler = nil
	}
}

// Generates unique ID for each request
func (server *WS_Streaming_Server) getNewSessionId() uint64 {
	server.sessionIdLock.Lock()
	defer server.sessionIdLock.Unlock()

	server.nextSessionId++

	return server.nextSessionId
}

func (server *WS_Streaming_Server) Start() {
	server.controlConnection.Initialize(server) // Initialize control connection

	var wg sync.WaitGroup

	wg.Add(2)

	go server.runHTTPServer(&wg)
	go server.runHTTPSecureServer(&wg)

	wg.Wait()
}

func (server *WS_Streaming_Server) runHTTPServer(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	bind_addr := os.Getenv("BIND_ADDRESS")

	// Setup HTTP server
	var tcp_port int
	tcp_port = 80
	customTCPPort := os.Getenv("HTTP_PORT")
	if customTCPPort != "" {
		tcpp, e := strconv.Atoi(customTCPPort)
		if e == nil {
			tcp_port = tcpp
		}
	}

	// Listen
	LogInfo("[HTTP] Listening on " + bind_addr + ":" + strconv.Itoa(tcp_port))
	errHTTP := http.ListenAndServe(bind_addr+":"+strconv.Itoa(tcp_port), server)

	if errHTTP != nil {
		LogError(errHTTP)
	}
}

func (server *WS_Streaming_Server) runHTTPSecureServer(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	bind_addr := os.Getenv("BIND_ADDRESS")

	// Setup HTTPS server
	var ssl_port int
	ssl_port = 443
	customSSLPort := os.Getenv("SSL_PORT")
	if customSSLPort != "" {
		sslp, e := strconv.Atoi(customSSLPort)
		if e == nil {
			ssl_port = sslp
		}
	}

	certFile := os.Getenv("SSL_CERT")
	keyFile := os.Getenv("SSL_KEY")

	if certFile != "" && keyFile != "" {
		// Listen
		LogInfo("[SSL] Listening on " + bind_addr + ":" + strconv.Itoa(ssl_port))
		errSSL := http.ListenAndServeTLS(bind_addr+":"+strconv.Itoa(ssl_port), certFile, keyFile, server)

		if errSSL != nil {
			LogError(errSSL)
		}
	}
}

func (server *WS_Streaming_Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	sessionId := server.getNewSessionId()

	ip, _, err := net.SplitHostPort(req.RemoteAddr)

	if err != nil {
		LogError(err)
		w.WriteHeader(200)
		fmt.Fprintf(w, "Websocket streaming server (Version 1.0.0)")
		return
	}

	LogRequest(sessionId, ip, ""+req.Method+" "+req.RequestURI)

	parts := strings.Split(req.RequestURI, "/")

	if parts == nil || len(parts) != 4 {
		if server.staticTestHandler != nil {
			server.staticTestHandler.ServeHTTP(w, req)
		} else if req.RequestURI == "/" {
			w.WriteHeader(200)
			fmt.Fprintf(w, "Websocket streaming server (Version 1.0.0)")
		} else {
			w.WriteHeader(404)
			fmt.Fprintf(w, "Not found.")
		}
		return
	}

	if !server.isIPExempted(ip) {
		if !server.AddIP(ip) {
			w.WriteHeader(429)
			fmt.Fprintf(w, "Too many requests.")
			LogRequest(sessionId, ip, "Connection rejected: Too many requests")
			return
		}
	}

	go server.HandleStreamingSession(sessionId, w, req, ip, parts[1], parts[2], parts[3])
}

func (server *WS_Streaming_Server) AddSession(s *WS_Streaming_Session) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	server.sessions[s.id] = s
}

func (server *WS_Streaming_Server) RemoveSession(id uint64) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	delete(server.sessions, id)
}

func (server *WS_Streaming_Server) isPublishing(channel string) bool {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	return server.channels[channel] != nil && server.channels[channel].is_publishing
}

func (server *WS_Streaming_Server) GetPublisher(channel string) *WS_Streaming_Session {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	if server.channels[channel] == nil {
		return nil
	}

	if !server.channels[channel].is_publishing {
		return nil
	}

	id := server.channels[channel].publisher
	return server.sessions[id]
}

func (server *WS_Streaming_Server) SetPublisher(channel string, key string, stream_id string, s *WS_Streaming_Session) bool {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	if server.channels[channel] != nil && server.channels[channel].is_publishing {
		return false
	}

	if server.channels[channel] == nil {
		c := WS_Streaming_Channel{
			channel:       channel,
			key:           key,
			stream_id:     stream_id,
			is_publishing: true,
			publisher:     s.id,
			players:       make(map[uint64]bool),
		}
		server.channels[channel] = &c
	} else {
		server.channels[channel].key = key
		server.channels[channel].stream_id = stream_id
		server.channels[channel].is_publishing = true
		server.channels[channel].publisher = s.id
	}

	return true
}

func (server *WS_Streaming_Server) RemovePublisher(channel string) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	if server.channels[channel] == nil {
		return
	}

	server.channels[channel].publisher = 0
	server.channels[channel].is_publishing = false

	players := server.channels[channel].players

	for sid := range players {
		player := server.sessions[sid]
		if player != nil {
			player.isIdling = true
			player.isPlaying = false
		}
	}

	if !server.channels[channel].is_publishing && len(server.channels[channel].players) == 0 {
		delete(server.channels, channel)
	}
}

func (server *WS_Streaming_Server) GetIdlePlayers(channel string) []*WS_Streaming_Session {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	if server.channels[channel] == nil {
		return make([]*WS_Streaming_Session, 0)
	}

	players := server.channels[channel].players

	playersToStart := make([]*WS_Streaming_Session, 0)

	for sid := range players {
		player := server.sessions[sid]
		if player != nil && player.isIdling {
			playersToStart = append(playersToStart, player)
		}
	}

	return playersToStart
}

func (server *WS_Streaming_Server) GetPlayers(channel string) []*WS_Streaming_Session {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	if server.channels[channel] == nil {
		return make([]*WS_Streaming_Session, 0)
	}

	players := server.channels[channel].players

	playersToStart := make([]*WS_Streaming_Session, 0)

	for sid := range players {
		player := server.sessions[sid]
		if player != nil && player.isPlaying {
			playersToStart = append(playersToStart, player)
		}
	}

	return playersToStart
}

func (server *WS_Streaming_Server) AddPlayer(channel string, key string, s *WS_Streaming_Session) (bool, error) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	if server.channels[channel] == nil {
		c := WS_Streaming_Channel{
			channel:       channel,
			key:           key,
			stream_id:     "",
			is_publishing: false,
			publisher:     0,
			players:       make(map[uint64]bool),
		}
		server.channels[channel] = &c
	}

	if server.channels[channel].is_publishing {
		if subtle.ConstantTimeCompare([]byte(key), []byte(server.channels[channel].key)) == 1 {
			s.isIdling = false
		} else {
			return false, errors.New("Invalid key")
		}
	} else {
		s.isIdling = true
	}

	server.channels[channel].players[s.id] = true

	return s.isIdling, nil
}

func (server *WS_Streaming_Server) RemovePlayer(channel string, key string, s *WS_Streaming_Session) {
	if server.channels[channel] == nil {
		return
	}

	delete(server.channels[channel].players, s.id)

	s.isIdling = false
	s.isPlaying = false

	if !server.channels[channel].is_publishing && len(server.channels[channel].players) == 0 {
		delete(server.channels, channel)
	}
}
