// Streaming server session

package main

import messages "github.com/AgustinSRG/tcp-video-streaming/common/message"

// Handles PUBLISH-REQUEST message
// requestId - Request ID
// channel - The channel
// key  - The streaming key
// ip - User IP
func (session *ControlSession) HandlePublishRequest(requestId string, channel string, key string, ip string) {
	if session.sessionType != SESSION_TYPE_RTMP && session.sessionType != SESSION_TYPE_WSS {
		return
	}

	if !validateStreamIDString(channel) {
		session.SendPublishDeny(requestId, channel)
		return
	}

	if !validateStreamIDString(key) {
		session.SendPublishDeny(requestId, channel)
		return
	}

	keyValid, resolutionList, record, previewsConfig := ValidateStreamKey(channel, key, ip)
	if !keyValid {
		session.SendPublishDeny(requestId, channel)
		return
	}

	streamId := session.server.coordinator.GenerateStreamID()

	// Change coordinator status data

	channelData := session.server.coordinator.AcquireChannel(channel)

	if !channelData.closed {
		// Already publishing
		session.server.coordinator.ReleaseChannel(channelData)
		session.SendPublishDeny(requestId, channel)
		return
	}

	channelData.closed = false
	channelData.publisher = session.id
	switch session.sessionType {
	case SESSION_TYPE_RTMP:
		channelData.publishMethod = PUBLISH_METHOD_RTMP
	case SESSION_TYPE_WSS:
		channelData.publishMethod = PUBLISH_METHOD_WS
	default:
		channelData.closed = true
		session.server.coordinator.ReleaseChannel(channelData)
		session.SendPublishDeny(requestId, channel)
		return
	}
	session.AssociateChannel(channel)

	// Find an encoder and assign it
	encoderServer := session.server.AssignAvailableEncoder()
	if encoderServer == nil {
		channelData.closed = true
		session.server.coordinator.ReleaseChannel(channelData)
		session.SendPublishDeny(requestId, channel)
		return
	}

	encoderServer.AssociateChannel(channel)
	channelData.encoder = encoderServer.id

	encoderServer.SendEncodeStart(channel, streamId, channelData.publishMethod, session.GeneratePublishSourceURL(channel, key), resolutionList, record, previewsConfig)

	// Release channel data
	session.server.coordinator.ReleaseChannel(channelData)

	// Accepted
	session.SendPublishAccept(requestId, channel, streamId)
}

// Handles a PUBLISH-END message
// channel - The channel
// streamId - The stream ID
func (session *ControlSession) HandlePublishEnd(channel string, streamId string) {
	if session.sessionType != SESSION_TYPE_RTMP && session.sessionType != SESSION_TYPE_WSS {
		return
	}

	channelData := session.server.coordinator.AcquireChannel(channel)

	if channelData.closed {
		session.server.coordinator.ReleaseChannel(channelData)
		return
	}

	channelData.closed = true

	// Disassociate the channel from the session
	session.DisassociateChannel(channel)

	// Find encoder and notice it
	encoderId := channelData.encoder
	encoderSession := session.server.GetSession(encoderId)

	if encoderSession != nil {
		encoderSession.SendEncodeStop(channel, streamId)
		encoderSession.DisassociateChannel(channel)
	}

	session.server.coordinator.ReleaseChannel(channelData)
}

// Sends a PUBLISH-DENY message
// requestId - The request ID
// channel - The channel
func (session *ControlSession) SendPublishDeny(requestId string, channel string) {
	params := make(map[string]string)

	params["Request-ID"] = requestId
	params["Stream-Channel"] = channel

	msg := messages.WebsocketMessage{
		Method: "PUBLISH-DENY",
		Params: params,
	}

	err := session.Send(msg)

	if err != nil {
		LogError(err)
	}
}

// Sends a PUBLISH-ACCEPT message
// requestId - The request ID
// channel - The channel
// streamId - The stream ID
func (session *ControlSession) SendPublishAccept(requestId string, channel string, streamId string) {
	params := make(map[string]string)

	params["Request-ID"] = requestId
	params["Stream-Channel"] = channel
	params["Stream-ID"] = streamId

	msg := messages.WebsocketMessage{
		Method: "PUBLISH-ACCEPT",
		Params: params,
	}

	err := session.Send(msg)

	if err != nil {
		LogError(err)
	}
}

// Sends a STREAM-KILL message
// channel - The channel
// streamId - The stream ID
func (session *ControlSession) SendStreamKill(channel string, streamId string) {
	params := make(map[string]string)

	params["Stream-Channel"] = channel
	params["Stream-ID"] = streamId

	msg := messages.WebsocketMessage{
		Method: "STREAM-KILL",
		Params: params,
	}

	err := session.Send(msg)

	if err != nil {
		LogError(err)
	}
}
