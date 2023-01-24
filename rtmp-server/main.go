// Main

package main

import "github.com/joho/godotenv"

const VERSION = "1.0.0"

func main() {
	godotenv.Load() // Load env vars

	InitLog() // Initializes log utils

	LogInfo("RTMP Server started - Version " + VERSION)

	server := CreateRTMPServer()

	if server != nil {
		server.Start()
	}
}
