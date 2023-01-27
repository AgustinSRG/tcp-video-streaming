// HTTP commands

package main

import (
	"fmt"
	"net/http"
)

// Runs stream close command
// w - Writer to send the response
// req - Client request
func (server *Streaming_Coordinator_Server) RunStreamCloseCommand(w http.ResponseWriter, req *http.Request) {
	authentication := req.Header.Get("Authorization")

	if !CheckCommandAuthentication(authentication) {
		w.WriteHeader(403)
		fmt.Fprintf(w, "Invalid authorization header.")
		return
	}

	channel := req.Header.Get("x-streaming-channel")
	streamId := req.Header.Get("x-streaming-id")

	server.KillStream(channel, streamId)

	w.WriteHeader(200)
	fmt.Fprintf(w, "SUCCESS")
}

// Kills a running stream
// channel - Channel ID
// streamId - Stream ID
func (server *Streaming_Coordinator_Server) KillStream(channel string, streamId string) {
	channelData := server.coordinator.AcquireChannel(channel)

	if channelData.closed {
		server.coordinator.ReleaseChannel(channelData)
		return
	}

	publishSession := server.GetSession(channelData.publisher)

	server.coordinator.ReleaseChannel(channelData)

	if publishSession != nil {
		publishSession.SendStreamKill(channel, streamId)
	}
}
