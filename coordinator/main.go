// Main

package main

import "github.com/joho/godotenv"

const VERSION = "1.0.0"

func main() {
	godotenv.Load() // Load env vars

	InitLog()

	LogInfo("Started Coordinator Streaming Server  - Version " + VERSION)

	server := Streaming_Coordinator_Server{}

	server.Initialize()
	server.Start()
}
