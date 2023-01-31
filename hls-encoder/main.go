// Main

package main

import "github.com/joho/godotenv"

const VERSION = "1.0.0"

func main() {
	godotenv.Load() // Load env vars

	InitLog()

	LogInfo("Started HLS encoder worker - Version " + VERSION)

	server := &HLS_Encoder_Server{}

	server.Initialize()
	server.Start()
}
