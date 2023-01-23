// Main

package main

import (
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load() // Load env vars

	InitLog()

	LogInfo("Websocket streaming server (Version 1.0.0)")

	server := WS_Streaming_Server{}

	server.Initialize()
	server.Start()
}
