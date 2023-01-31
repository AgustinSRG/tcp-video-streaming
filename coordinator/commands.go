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

/* Status report */

type ReportAPIResponse_StreamingServer struct {
	Id         uint64 `json:"id"`
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	SSL        bool   `json:"ssl"`
	ServerType string `json:"serverType"`
}

type ReportAPIResponse_Encoder struct {
	Id       uint64 `json:"id"`
	Capacity int    `json:"capacity"`
	Load     int    `json:"load"`
}

type ReportAPIResponse_ActiveStream struct {
	Channel      string `json:"channel"`
	StreamId     string `json:"streamId"`
	StreamServer uint64 `json:"streamServer"`
	Encoder      uint64 `json:"encoder"`
}

type ReportAPIResponse struct {
	StreamingServers []ReportAPIResponse_StreamingServer `json:"streamingServers"`
	Encoders         []ReportAPIResponse_Encoder         `json:"encoders"`
	ActiveStreams    []ReportAPIResponse_ActiveStream    `json:"activeStreams"`
}

// Runs report command
// w - Writer to send the response
// req - Client request
func (server *Streaming_Coordinator_Server) RunReportCommand(w http.ResponseWriter, req *http.Request) {
	authentication := req.Header.Get("Authorization")

	if !CheckCommandAuthentication(authentication) {
		w.WriteHeader(401)
		fmt.Fprintf(w, "Invalid authorization header.")
		return
	}

	w.Header().Add("Cache-Control", "no-cache")

	report := server.coordinator.GetReport()

	json, err := json.Marshal(report)

	if err != nil {
		LogError(err)
		w.WriteHeader(500)
		fmt.Fprintf(w, "ERROR: "+err.Error())
	}

	w.Header().Add("Content-Type", "application/json")

	w.WriteHeader(200)
	fmt.Fprintf(w, string(json))
}

// Generates a report of the current status
func (coord *Streaming_Coordinator) GetReport() ReportAPIResponse {
	channelList := make([]string, 0)
	streamingServers := make([]ReportAPIResponse_StreamingServer, 0)
	encoders := make([]ReportAPIResponse_Encoder, 0)

	coord.mutex.Lock()

	for channel := range coord.channels {
		channelList = append(channelList, channel)
	}

	for _, server := range coord.rtmpServers {
		streamingServers = append(streamingServers, ReportAPIResponse_StreamingServer{
			Id:         server.id,
			IP:         server.ip,
			Port:       server.port,
			SSL:        server.ssl,
			ServerType: "RTMP",
		})
	}

	for _, server := range coord.wssServers {
		streamingServers = append(streamingServers, ReportAPIResponse_StreamingServer{
			Id:         server.id,
			IP:         server.ip,
			Port:       server.port,
			SSL:        server.ssl,
			ServerType: "WS",
		})
	}

	for _, encoder := range coord.hlsEncoders {
		encoders = append(encoders, ReportAPIResponse_Encoder{
			Id:       encoder.id,
			Capacity: encoder.capacity,
			Load:     encoder.load,
		})
	}

	coord.mutex.Unlock()

	activeStreams := make([]ReportAPIResponse_ActiveStream, 0)

	for i := 0; i < len(channelList); i++ {
		channelData := coord.AcquireChannel(channelList[i])

		if !channelData.closed {
			activeStreams = append(activeStreams, ReportAPIResponse_ActiveStream{
				Channel:      channelData.id,
				StreamId:     channelData.id,
				StreamServer: channelData.publisher,
				Encoder:      channelData.encoder,
			})
		}

		coord.ReleaseChannel(channelData)
	}

	return ReportAPIResponse{
		ActiveStreams:    activeStreams,
		StreamingServers: streamingServers,
		Encoders:         encoders,
	}
}
