// Main

package main

func main() {
	LogInfo("Websocket streaming server (Version 1.0.0)")

	server := WS_Streaming_Server{}

	server.Initialize()
	server.Start()
}
