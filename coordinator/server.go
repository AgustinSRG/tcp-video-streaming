// Websocket server

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

// Stores the status information of the coordinator websocket server
type Streaming_Coordinator_Server struct {
	wsUpgrader *websocket.Upgrader // Upgrader to handle websocket connections

	nextSessionId uint64      // ID for the next session
	sessionIdLock *sync.Mutex // Mutex to ensure session IDs are unique

	sessions map[uint64]*ControlSession // List of active websocket sessions

	mutex *sync.Mutex // Mutex to control the access to the status data (sessions)

	coordinator *Streaming_Coordinator
}

// Initializes the server
func (server *Streaming_Coordinator_Server) Initialize() {
	server.wsUpgrader = &websocket.Upgrader{}
	server.wsUpgrader.CheckOrigin = func(r *http.Request) bool { return true }

	server.nextSessionId = 0
	server.sessionIdLock = &sync.Mutex{}

	server.mutex = &sync.Mutex{}
	server.sessions = make(map[uint64]*ControlSession)

	server.coordinator = &Streaming_Coordinator{}
	server.coordinator.Initialize()
}

// Generates unique ID for each request
func (server *Streaming_Coordinator_Server) getNewSessionId() uint64 {
	server.sessionIdLock.Lock()
	defer server.sessionIdLock.Unlock()

	server.nextSessionId++

	return server.nextSessionId
}

// Starts the server
func (server *Streaming_Coordinator_Server) Start() {
	var wg sync.WaitGroup

	wg.Add(2)

	go server.runHTTPServer(&wg)
	go server.runHTTPSecureServer(&wg)

	wg.Wait()
}

// Runs the HTTP server
// wg - Waiting group
func (server *Streaming_Coordinator_Server) runHTTPServer(wg *sync.WaitGroup) {
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
func (server *Streaming_Coordinator_Server) runHTTPSecureServer(wg *sync.WaitGroup) {
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

// Handles each HTTP request
// w - Writer to send the response
// req - Client request
func (server *Streaming_Coordinator_Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	sessionId := server.getNewSessionId()

	ip, _, err := net.SplitHostPort(req.RemoteAddr)

	if err != nil {
		LogError(err)
		w.WriteHeader(200)
		fmt.Fprintf(w, "Coordinator streaming server - Version "+VERSION)
		return
	}

	LogRequest(sessionId, ip, ""+req.Method+" "+req.RequestURI)

	if req.RequestURI == "/" {
		w.WriteHeader(200)
		fmt.Fprintf(w, "Coordinator streaming server - Version "+VERSION)
	} else if req.RequestURI == "/ws/control/rtmp" {
		authToken := req.Header.Get("x-control-auth-token")
		if !ValidateAuthenticationToken(authToken, RTMP_AUTH_SUBJECT) {
			w.WriteHeader(403)
			fmt.Fprintf(w, "Invalid authentication token.")
			LogRequest(sessionId, ip, "Invalid authentication token.")
			return
		}

		conn, err := server.wsUpgrader.Upgrade(w, req, nil)

		if err != nil {
			LogDebugSession(sessionId, ip, "Error: "+err.Error())
			return
		}

		session := CreateSession(server, conn, sessionId, ip, SESSION_TYPE_RTMP)

		go session.Run()
	} else if req.RequestURI == "/ws/control/wss" {
		authToken := req.Header.Get("x-control-auth-token")
		if !ValidateAuthenticationToken(authToken, WSS_AUTH_SUBJECT) {
			w.WriteHeader(403)
			fmt.Fprintf(w, "Invalid authentication token.")
			LogRequest(sessionId, ip, "Invalid authentication token.")
			return
		}

		conn, err := server.wsUpgrader.Upgrade(w, req, nil)

		if err != nil {
			LogDebugSession(sessionId, ip, "Error: "+err.Error())
			return
		}

		session := CreateSession(server, conn, sessionId, ip, SESSION_TYPE_WSS)

		go session.Run()
	} else if req.RequestURI == "/ws/control/hls" {
		authToken := req.Header.Get("x-control-auth-token")
		if !ValidateAuthenticationToken(authToken, HLS_AUTH_SUBJECT) {
			w.WriteHeader(403)
			fmt.Fprintf(w, "Invalid authentication token.")
			LogRequest(sessionId, ip, "Invalid authentication token.")
			return
		}

		conn, err := server.wsUpgrader.Upgrade(w, req, nil)

		if err != nil {
			LogDebugSession(sessionId, ip, "Error: "+err.Error())
			return
		}

		session := CreateSession(server, conn, sessionId, ip, SESSION_TYPE_HLS)

		go session.Run()
	} else {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Not found.")
	}
}

// Adds a session to the list
// s - Session
func (server *Streaming_Coordinator_Server) AddSession(s *ControlSession) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	server.sessions[s.id] = s
}

// Removes a session from the list
// id - Session ID
func (server *Streaming_Coordinator_Server) RemoveSession(id uint64) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	delete(server.sessions, id)
}
