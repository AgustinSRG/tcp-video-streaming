// Websocket streaming server

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

type WS_Streaming_Server struct {
	wsUpgrader        *websocket.Upgrader
	controlConnection *ControlServerConnection

	nextSessionId uint64
	sessionIdLock *sync.Mutex

	ipCount      map[string]uint32
	mutexIpCount *sync.Mutex
	ipLimit      uint32
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
}

// Generates unique ID for each request
func (server *WS_Streaming_Server) getNewSessionId() uint64 {
	server.sessionIdLock.Lock()
	defer server.sessionIdLock.Unlock()

	server.nextSessionId++

	return server.nextSessionId
}

func (server *WS_Streaming_Server) Start() {
	server.controlConnection.Initialize() // Initialize control connection

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

func (node *WS_Streaming_Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	sessionId := node.getNewSessionId()

	ip, _, err := net.SplitHostPort(req.RemoteAddr)

	if err != nil {
		LogError(err)
		w.WriteHeader(200)
		fmt.Fprintf(w, "Websocket streaming server (Version 1.0.0)")
		return
	}

	LogRequest(sessionId, ip, ""+req.Method+" "+req.RequestURI)

	w.WriteHeader(200)
	fmt.Fprintf(w, "Websocket streaming server (Version 1.0.0)")
}
