package main

import "github.com/joho/godotenv"

func main() {
	godotenv.Load() // Load env vars

	InitLog()

	LogInfo("RTMP-Server (Version 1.0.0)")

	server := CreateRTMPServer()

	if server != nil {
		server.Start()
	}
}
