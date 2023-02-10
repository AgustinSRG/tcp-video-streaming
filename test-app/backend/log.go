// Logs

package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

var LOG_MUTEX = sync.Mutex{}

var LOG_DEBUG_ENABLED = false
var LOG_REQUESTS_ENABLED = false

func InitLog() {
	LOG_DEBUG_ENABLED = (os.Getenv("LOG_DEBUG") == "YES")
	LOG_REQUESTS_ENABLED = (os.Getenv("LOG_REQUESTS") != "NO")
}

func LogLine(line string) {
	tm := time.Now()
	LOG_MUTEX.Lock()
	defer LOG_MUTEX.Unlock()
	fmt.Printf("[%s] %s\n", tm.Format("2006-01-02 15:04:05"), line)
}

func LogWarning(line string) {
	LogLine("[WARNING] " + line)
}

func LogInfo(line string) {
	LogLine("[INFO] " + line)
}

func LogError(err error) {
	LogLine("[ERROR] " + err.Error())
}

func LogErrorMessage(err string) {
	LogLine("[ERROR] " + err)
}

func LogRequest(r *http.Request) {
	if LOG_REQUESTS_ENABLED {
		LogLine("[REQUEST] (From: " + GetClientIP(r) + ") " + r.Method + " " + r.URL.Path)
	}
}

func LogDebug(line string) {
	if LOG_DEBUG_ENABLED {
		LogLine("[DEBUG] " + line)
	}
}
