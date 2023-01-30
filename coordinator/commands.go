// HTTP commands

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Runs stream close command
// w - Writer to send the response
// req - Client request
func (server *Streaming_Coordinator_Server) RunStreamCloseCommand(w http.ResponseWriter, req *http.Request) {
	authentication := req.Header.Get("Authorization")

	if !CheckCommandAuthentication(authentication) {
		w.WriteHeader(401)
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

// Response for the capacity API
type CapacityAPIResponse struct {
	Load         int `json:"load"`
	Capacity     int `json:"capacity"`
	EncoderCount int `json:"encoderCount"`
}

// Runs capacity get command
// w - Writer to send the response
// req - Client request
func (server *Streaming_Coordinator_Server) RunGetCapacityCommand(w http.ResponseWriter, req *http.Request) {
	authentication := req.Header.Get("Authorization")

	if !CheckCommandAuthentication(authentication) {
		w.WriteHeader(401)
		fmt.Fprintf(w, "Invalid authorization header.")
		return
	}

	w.Header().Add("Cache-Control", "no-cache")

	capacity := server.coordinator.GetCapacity()

	json, err := json.Marshal(capacity)

	if err != nil {
		LogError(err)
		w.WriteHeader(500)
		fmt.Fprintf(w, "ERROR: "+err.Error())
	}

	w.Header().Add("Content-Type", "application/json")

	w.WriteHeader(200)
	fmt.Fprintf(w, string(json))
}

// Computes current capacity and returns the information
func (coord *Streaming_Coordinator) GetCapacity() CapacityAPIResponse {
	coord.mutex.Lock()
	defer coord.mutex.Unlock()

	totalCapacity := 0
	totalLoad := 0
	encoderCount := 0

	for _, encoder := range coord.hlsEncoders {
		encoderCount++

		totalLoad += encoder.load

		if totalCapacity >= 0 {
			if encoder.capacity < 0 {
				totalCapacity = -1
			} else {
				totalCapacity += encoder.capacity
			}
		}
	}

	return CapacityAPIResponse{
		Capacity:     totalCapacity,
		Load:         totalLoad,
		EncoderCount: encoderCount,
	}
}
