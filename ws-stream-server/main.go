// Main

package main

import (
	"github.com/joho/godotenv"
)

const VERSION = "1.0.0"

func main() {
	godotenv.Load() // Load env vars

	InitLog()

	LogInfo("Started Websocket streaming server - Version " + VERSION)

	server := WS_Streaming_Server{}

	server.Initialize()
	server.Start()
}
