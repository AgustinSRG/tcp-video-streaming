// Logs

package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var LOG_MUTEX = sync.Mutex{}

var LOG_DEBUG_ENABLED = false
var LOG_TASKS_ENABLED = false

func InitLog() {
	LOG_DEBUG_ENABLED = (os.Getenv("LOG_DEBUG") == "YES")
	LOG_TASKS_ENABLED = (os.Getenv("LOG_TASK_STATUS") != "NO")
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

func LogTaskStatus(channel string, streamId string, line string) {
	if LOG_TASKS_ENABLED {
		LogLine("[TASK] [" + channel + "/" + streamId + "]" + line)
	}
}

func LogDebug(line string) {
	if LOG_DEBUG_ENABLED {
		LogLine("[DEBUG] " + line)
	}
}

func LogDebugTask(channel string, streamId string, line string) {
	if LOG_DEBUG_ENABLED {
		LogLine("[DEBUG] [TASK] [" + channel + "/" + streamId + "]" + line)
	}
}
