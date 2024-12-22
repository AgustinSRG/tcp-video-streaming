// HLS encoder session

package main

import (
	"fmt"

	messages "github.com/AgustinSRG/go-simple-rpc-message"
)

// Handles REGISTER message
// capacity - Encoder capacity
func (session *ControlSession) HandleEncoderRegister(capacity int) {
	if session.sessionType != SESSION_TYPE_HLS {
		return
	}

	session.server.coordinator.RegisterEncoder(session.id, capacity)

	session.log("REGISTERED ENCODER / CAPACITY: " + fmt.Sprint(capacity))

	session.encoderRegistered = true
}

// Handles STREAM-AVAILABLE message
// channel - The channel
// streamId - The stream ID
// streamType - The stream type (`HLS-LIVE`, `HLS-VOD` or `IMG-PREVIEW`)
// resolution - The resolution ({WIDTH}x{HEIGHT}-{FPS})
// indexFile - Full path to the index file in the shared file system
func (session *ControlSession) HandleStreamAvailable(channel string, streamId string, streamType string, resolution string, startTimeStr string, indexFile string) {
	if session.sessionType != SESSION_TYPE_HLS {
		return
	}

	// Register active stream

	session.server.coordinator.AddActiveStream(channel, streamId)

	// Send event to application

	channelData := session.server.coordinator.AcquireChannel(channel)
	defer session.server.coordinator.ReleaseChannel(channelData)

	channelData.nextEventId++

	eventId := channelData.nextEventId

	event := &PendingStreamAvailableEvent{
		id:         eventId,
		channel:    channel,
		streamId:   streamId,
		streamType: streamType,
		resolution: resolution,
		indexFile:  indexFile,
		startTime:  startTimeStr,
		cancelled:  false,
	}

	channelData.pendingEvents[channelData.nextEventId] = event

	go SendStreamAvailableEvent(channelData, event)

	startTimeStrDisplay := startTimeStr

	if startTimeStrDisplay == "" {
		startTimeStrDisplay = "0"
	}

	session.log("STREAM-AVAILABLE: " + channel + "/" + streamId + " | TYPE=" + streamType + " | RESOLUTION=" + resolution + " | START-TIME: " + startTimeStrDisplay + " | INDEX=" + indexFile)
}

// Handles STREAM-CLOSED message
// channel - The channel
// streamId - The stream ID
func (session *ControlSession) HandleStreamClosed(channel string, streamId string) {
	if session.sessionType != SESSION_TYPE_HLS {
		return
	}

	session.server.coordinator.OnActiveStreamClosed(channel, streamId)

	channelData := session.server.coordinator.AcquireChannel(channel)
	defer session.server.coordinator.ReleaseChannel(channelData)

	if !channelData.closed && channelData.encoder == session.id {
		// Find publisher and kill the stream session
		publisherId := channelData.publisher
		pubSession := session.server.GetSession(publisherId)

		if pubSession != nil {
			pubSession.SendStreamKill(channelData.id, channelData.streamId)
		}
	}

	// Cancel any stream-available events
	for _, event := range channelData.pendingEvents {
		event.cancelled = true
	}

	session.log("STREAM-CLOSED: " + channel + "/" + streamId)
}

// Sends ENCODE-START message
// channel - The channel
// streamId - The stream ID
// publishType - The publishing type
// publishSourceURL - The source URL
// resolutionList - List of resolutions
// record - True to record
// previewsConfig - Image previews configuration
func (session *ControlSession) SendEncodeStart(channel string, streamId string, publishType int, publishSourceURL string, resolutionList ResolutionList, record bool, previewsConfig PreviewsConfiguration) {
	params := make(map[string]string)

	params["Stream-Channel"] = channel
	params["Stream-ID"] = streamId

	switch publishType {
	case PUBLISH_METHOD_RTMP:
		params["Stream-Source-Type"] = "RTMP"
	case PUBLISH_METHOD_WS:
		params["Stream-Source-Type"] = "WS"
	}

	params["Stream-Source-URI"] = publishSourceURL

	params["Resolutions"] = resolutionList.Encode()

	if record {
		params["Record"] = "True"
	} else {
		params["Record"] = "False"
	}

	params["Previews"] = previewsConfig.Encode()

	msg := messages.RPCMessage{
		Method: "ENCODE-START",
		Params: params,
	}

	err := session.Send(msg)

	if err != nil {
		LogError(err)
	}
}

// Sends ENCODE-STOP message
// channel - The channel
// streamId - The stream ID
func (session *ControlSession) SendEncodeStop(channel string, streamId string) {
	params := make(map[string]string)

	params["Stream-Channel"] = channel
	params["Stream-ID"] = streamId

	msg := messages.RPCMessage{
		Method: "ENCODE-STOP",
		Params: params,
	}

	err := session.Send(msg)

	if err != nil {
		LogError(err)
	}
}
