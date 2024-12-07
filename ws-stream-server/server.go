// Websocket streaming server

package main

import (
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tls_certificate_loader "github.com/AgustinSRG/go-tls-certificate-loader"
	"github.com/gorilla/websocket"
)

// Status data for the streaming server
type WS_Streaming_Server struct {
	wsUpgrader *websocket.Upgrader // Upgrader to handle websocket connections

	controlConnection *ControlServerConnection // Connection to the coordinator server

	nextSessionId uint64      // ID for the next session
	sessionIdLock *sync.Mutex // Mutex to ensure session IDs are unique

	ipLimit      uint32            // Max active connections allowed for each IP address
	ipCount      map[string]uint32 // IP -> Active connection count
	mutexIpCount *sync.Mutex       // Mutex to control access to ipCount

	gopCacheLimit int64 // GOP cache size limit (bytes)

	sessions map[uint64]*WS_Streaming_Session // List of active sessions
	channels map[string]*WS_Streaming_Channel // List of active streaming channels

	mutex *sync.Mutex // Mutex to control the access to the status data (channels, sessions)

	staticTestHandler http.Handler // Static HTTP handles for browser testing
}

// Stores the status data for a streaming channel
type WS_Streaming_Channel struct {
	channel       string          // Channel ID
	key           string          // Streaming key
	stream_id     string          // Stream ID
	publisher     uint64          // Session ID of the publisher
	is_publishing bool            // True if the channel is being published
	players       map[uint64]bool // List of sessions playing the stream
}

// Initializes the streaming server
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

// Starts the server
func (server *WS_Streaming_Server) Start() {
	server.controlConnection.Initialize(server) // Initialize control connection

	var wg sync.WaitGroup

	wg.Add(2)

	go server.runHTTPServer(&wg)
	go server.runHTTPSecureServer(&wg)

	wg.Wait()
}

// Runs the HTTP server
// wg - Waiting group
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

// Runs the HTTPS server
// wg - Waiting group
func (server *WS_Streaming_Server) runHTTPSecureServer(wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	bind_addr := os.Getenv("BIND_ADDRESS")

	// Setup HTTPS server
	var port int
	port = 443
	customSSLPort := os.Getenv("SSL_PORT")
	if customSSLPort != "" {
		sslp, e := strconv.Atoi(customSSLPort)
		if e == nil {
			port = sslp
		}
	}

	certFile := os.Getenv("SSL_CERT")
	keyFile := os.Getenv("SSL_KEY")

	if certFile == "" || keyFile == "" {
		return
	}

	var sslReloadSeconds = 60
	customSslReloadSeconds := os.Getenv("SSL_CHECK_RELOAD_SECONDS")
	if customSslReloadSeconds != "" {
		n, e := strconv.Atoi(customSslReloadSeconds)
		if e == nil {
			sslReloadSeconds = n
		}
	}

	certificateLoader, err := tls_certificate_loader.NewTlsCertificateLoader(tls_certificate_loader.TlsCertificateLoaderConfig{
		CertificatePath:   certFile,
		KeyPath:           keyFile,
		CheckReloadPeriod: time.Duration(sslReloadSeconds) * time.Second,
		OnReload: func() {
			LogInfo("Reloaded SSL certificates")
		},
		OnError: func(err error) {
			LogErrorMessage("Error loading SSL key pair: " + err.Error())
		},
	})

	if err != nil {
		LogErrorMessage("Error starting HTTPS server: " + err.Error())
		return
	}

	defer certificateLoader.Close()

	// Setup HTTPS server

	tlsServer := http.Server{
		Addr:    bind_addr + ":" + strconv.Itoa(port),
		Handler: server,
		TLSConfig: &tls.Config{
			GetCertificate: certificateLoader.GetCertificate,
		},
	}

	// Listen

	LogInfo("[HTTPS] Listening on " + bind_addr + ":" + strconv.Itoa(port))

	errSSL := tlsServer.ListenAndServeTLS("", "")

	if errSSL != nil {
		LogErrorMessage("Error starting HTTPS server: " + errSSL.Error())
	}
}

// Handles each HTTP request
// w - Writer to send the response
// req - Client request
func (server *WS_Streaming_Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	sessionId := server.getNewSessionId()

	ip, _, err := net.SplitHostPort(req.RemoteAddr)

	if err != nil {
		LogError(err)
		w.WriteHeader(200)
		fmt.Fprintf(w, "Websocket streaming server - Version "+VERSION)
		return
	}

	LogRequest(sessionId, ip, ""+req.Method+" "+req.RequestURI)

	parts := strings.Split(req.RequestURI, "/")

	if parts == nil || len(parts) != 4 {
		if server.staticTestHandler != nil {
			server.staticTestHandler.ServeHTTP(w, req)
		} else if req.RequestURI == "/" {
			w.WriteHeader(200)
			fmt.Fprintf(w, "Websocket streaming server - Version "+VERSION)
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

	server.HandleStreamingSession(sessionId, w, req, ip, parts[1], parts[2], parts[3])
}

// Adds a session to the list
// s - Session
func (server *WS_Streaming_Server) AddSession(s *WS_Streaming_Session) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	server.sessions[s.id] = s
}

// Removes a session from the list
// id - Session ID
func (server *WS_Streaming_Server) RemoveSession(id uint64) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	delete(server.sessions, id)
}

// Checks if there is an active stream being published on a given channel
// channel - Channel ID
// Returns true if active publishing
func (server *WS_Streaming_Server) isPublishing(channel string) bool {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	return server.channels[channel] != nil && server.channels[channel].is_publishing
}

// Obtains a reference to the session that is publishing on a given channel
// channel - The channel ID
// Returns the reference, or nil
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

// Sets a publisher and a stream for a given channel
// channel - The channel ID
// key - The channel key
// stream_id - The stream ID
// s - The session that is publishing
// Returns true if success, false if there was another session publishing
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

// Removes the current publisher for a given channel
// channel - The channel ID
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

// Obtains the list of idle players for a given channel
// channel - The channel ID
// Returns the list of sessions waiting to play the stream
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

// Obtains the list of players for a given channel
// channel - The channel ID
// Returns the list of sessions playing the stream
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

// Adds a player to a given channel
// channel - The channel ID
// key - The channel key used by the player
// s - The session
// Returns:
//
//	idling - True if the channel was not active, so the player becomes idle. False means the player can begin receiving the stream
//	err - Error. If not nil, it means the channel of the key are not valid
func (server *WS_Streaming_Server) AddPlayer(channel string, key string, s *WS_Streaming_Session) (idling bool, err error) {
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

// Removes a player from a channel
// channel - The channel ID
// s - The session
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

// Kills any sessions publishing streams
func (server *WS_Streaming_Server) KillAllActivePublishers() {
	activePublishers := make([]*WS_Streaming_Session, 0)

	server.mutex.Lock()

	for _, channel := range server.channels {
		if channel == nil || !channel.is_publishing {
			continue
		}

		session := server.sessions[channel.publisher]

		if session != nil {
			activePublishers = append(activePublishers, session)
		}
	}

	server.mutex.Unlock()

	for i := 0; i < len(activePublishers); i++ {
		activePublishers[i].Kill()
	}
}
