// HLS encoder session

package main

import "fmt"

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
func (session *ControlSession) HandleStreamAvailable(channel string, streamId string, streamType string, resolution string, indexFile string) {
	if session.sessionType != SESSION_TYPE_HLS {
		return
	}

	session.log("STREAM-AVAILABLE: " + channel + "/" + streamId + " | TYPE=" + streamType + " | RESOLUTION=" + resolution + " | INDEX=" + indexFile)
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

	msg := WebsocketMessage{
		method: "ENCODE-START",
		params: params,
		body:   "",
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

	msg := WebsocketMessage{
		method: "ENCODE-STOP",
		params: params,
		body:   "",
	}

	err := session.Send(msg)

	if err != nil {
		LogError(err)
	}
}
