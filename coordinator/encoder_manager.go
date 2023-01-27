// Encoders management

package main

// Search in the list of available HLS encoders and assigns the stream to one
// Returns the control session, or nil, if none available
func (server *Streaming_Coordinator_Server) AssignAvailableEncoder() *ControlSession {
	selectedEncoder := uint64(0)
	currentLoad := 0
	encoderIsAvailable := false

	server.coordinator.mutex.Lock()

	for id, encoder := range server.coordinator.hlsEncoders {
		if encoder.capacity >= 0 && encoder.load >= encoder.capacity {
			continue
		}

		if !encoderIsAvailable {
			encoderIsAvailable = true
			selectedEncoder = id
			currentLoad = encoder.load
		} else if encoder.load < currentLoad {
			selectedEncoder = id
			currentLoad = encoder.load
		}
	}

	if encoderIsAvailable {
		server.coordinator.hlsEncoders[selectedEncoder].load++
	}

	server.coordinator.mutex.Unlock()

	if !encoderIsAvailable {
		return nil
	}

	return server.GetSession(selectedEncoder)
}

// Releases an encoder server, reducing the load by one
// encoderId - ID of the encoder
func (server *Streaming_Coordinator_Server) ReleaseEncoder(encoderId uint64) {
	server.coordinator.mutex.Lock()

	if server.coordinator.hlsEncoders[encoderId] != nil {
		server.coordinator.hlsEncoders[encoderId].load--
	}

	server.coordinator.mutex.Unlock()

}
