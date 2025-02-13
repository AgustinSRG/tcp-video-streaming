// HTTP server

package main

import (
	"mime"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	CORS_INSECURE_MODE_ENABLED = false // Insecure CORS mode for development and testing
)

// Logging middleware to log requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log request
		LogRequest(r)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// CORS middleware, only applied when CORS_INSECURE_MODE_ENABLED = true
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS
		corsOrigin := r.Header.Get("origin")

		if corsOrigin == "" {
			corsOrigin = "*"
		}

		w.Header().Set("Access-Control-Allow-Origin", corsOrigin)

		allowMethods := r.Header.Get("access-control-request-method")

		if allowMethods != "" {
			w.Header().Set("Access-Control-Allow-Methods", allowMethods)
		}

		allowHeaders := r.Header.Get("access-control-request-headers")

		if allowHeaders != "" {
			w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
		}

		w.Header().Set("Access-Control-Max-Age", "86400")

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// Handler for the OPTIONS method, only when CORS_INSECURE_MODE_ENABLED = true
func corsHeadInsecure(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// Adds cache TTL to static asset requests
func cacheTTLAdd(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// Runs HTTP server
// Creates mux router and launches the listener
// NOTE: This method locks the thread forever, run with coroutine: go RunHTTPServer(p, b)
// port - Port to listen
// bindAddr - Bind address
func RunHTTPServer(port string, bindAddr string) {
	router := mux.NewRouter()

	// Logging middleware
	router.Use(loggingMiddleware)

	if CORS_INSECURE_MODE_ENABLED {
		LogWarning("CORS insecure mode enabled. Use this only for development")
		router.Use(corsMiddleware)
	}

	// Key verification API

	router.HandleFunc("/callbacks/key_verification", callback_keyVerification).Methods("POST")

	// Event callback

	router.HandleFunc("/callbacks/events", callback_eventsHandler).Methods("POST")

	// Control API

	router.HandleFunc("/api/control", api_getPublishingDetails).Methods("GET")

	router.HandleFunc("/api/control/create", api_createChannel).Methods("POST")
	router.HandleFunc("/api/control/chan/{channel}/check", api_checkKey).Methods("POST")
	router.HandleFunc("/api/control/chan/{channel}", api_configChannel).Methods("POST")
	router.HandleFunc("/api/control/chan/{channel}/key", api_refreshKey).Methods("POST")
	router.HandleFunc("/api/control/chan/{channel}/close", api_closeStream).Methods("POST")
	router.HandleFunc("/api/control/chan/{channel}/vod/{vod}/delete", api_deleteVOD).Methods("POST")
	router.HandleFunc("/api/control/chan/{channel}/delete", api_deleteChannel).Methods("POST")

	// Watch API

	router.HandleFunc("/api/watch/{channel}", api_getChannelStatus).Methods("GET")
	router.HandleFunc("/api/watch/{channel}/vod", api_listVODs).Methods("GET")
	router.HandleFunc("/api/watch/{channel}/vod/{vod}", api_getVOD).Methods("GET")

	// Serve HLS

	router.HandleFunc("/hls/{channel}/{id}/{resolution}/{file:[0-9a-zA-Z-]+}.m3u8", hls_servePlaylist).Methods("GET")
	router.HandleFunc("/hls/{channel}/{id}/{resolution}/{file:[0-9a-zA-Z-]+}.ts", hls_serveFragment).Methods("GET")

	// Serve image previews

	router.HandleFunc("/img-preview/{channel}/{id}/{resolution}/{file:[0-9a-zA-Z-]+}.json", hls_servePreviewsIndex).Methods("GET")
	router.HandleFunc("/img-preview/{channel}/{id}/{resolution}/{file:[0-9a-zA-Z-]+}.jpg", hls_servePreviewImage).Methods("GET")

	// Debug Api

	router.HandleFunc("/api/debug/capacity", api_getDebugCapacity).Methods("GET")
	router.HandleFunc("/api/debug/report", api_getDebugReport).Methods("GET")

	// Static frontend

	frontend_path := os.Getenv("FRONTEND_PATH")

	if frontend_path == "" {
		frontend_path = "../frontend/dist/"
	}

	if CORS_INSECURE_MODE_ENABLED {
		router.Use(corsHeadInsecure)
	}

	mime.AddExtensionType(".js", "text/javascript")

	router.Use(cacheTTLAdd)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(frontend_path)))

	// Run server

	bind_addr := bindAddr

	// Setup HTTP server
	var tcp_port int
	tcp_port = 80
	if port != "" {
		tcpp, e := strconv.Atoi(port)
		if e == nil {
			tcp_port = tcpp
		}
	}

	// Listen
	LogInfo("[HTTP] Listening on " + bind_addr + ":" + strconv.Itoa(tcp_port))
	errHTTP := http.ListenAndServe(bind_addr+":"+strconv.Itoa(tcp_port), handlers.CompressHandler(router))

	if errHTTP != nil {
		LogError(errHTTP)
		os.Exit(5)
	}
}
