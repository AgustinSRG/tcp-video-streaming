// HLS related task features

package main

// Gets or create the sub stream object for the specified resolution
// resolution - Stream video resolution
// Returns a reference to the sub-stream
func (task *EncodingTask) getSubStream(resolution Resolution) *SubStreamStatus {
	subStreamId := resolution.Encode()
	if task.subStreams[subStreamId] == nil {
		task.subStreams[subStreamId] = &SubStreamStatus{
			resolution:            resolution,
			livePlaylist:          nil,
			livePlaylistAvailable: false,
			liveWriting:           false,
			liveWritePending:      false,
			liveWriteData:         nil,
			vodPlaylist:           nil,
			vodPlaylistAvailable:  false,
			vodIndex:              0,
			vodWriting:            false,
			vodWritePending:       false,
			vodWriteData:          nil,
			fragmentCount:         0,
			removedFragmentsCount: 0,
			fragments:             make(map[int]*HLS_Fragment),
			fragmentsReady:        make(map[int]bool),
		}
	}
	return task.subStreams[subStreamId]
}

// Call when a new TS fragment is ready
// resolution - Stream video resolution
// fragmentIndex - Index of the fragment
func (task *EncodingTask) OnFragmentReady(resolution Resolution, fragmentIndex int) {
	task.mutex.Lock()
	defer task.mutex.Unlock()

	subStream := task.getSubStream(resolution)

	if fragmentIndex < subStream.fragmentCount {
		return
	}

	if fragmentIndex >= task.server.hlsMaxFragmentCount {
		return
	}

	subStream.fragmentsReady[fragmentIndex] = true

	task.updateHLSInternal(subStream)
}

// Call when a new M3U8 playlist if received
// resolution - Stream video resolution
// playlist - Received playlist
func (task *EncodingTask) OnPlaylistUpdate(resolution Resolution, playlist *HLS_PlayList) {
	task.mutex.Lock()
	defer task.mutex.Unlock()

	subStream := task.getSubStream(resolution)

	for i := 0; i < len(playlist.fragments); i++ {
		if playlist.fragments[i].Index < subStream.fragmentCount {
			continue
		}

		if playlist.fragments[i].Index >= task.server.hlsMaxFragmentCount {
			continue
		}

		subStream.fragments[playlist.fragments[i].Index] = &playlist.fragments[i]
	}

	task.updateHLSInternal(subStream)
}

// Updates the playlists if required
// subStream - Reference to the sub-stream
func (task *EncodingTask) updateHLSInternal(subStream *SubStreamStatus) {
	// Compute the new fragment count
	newFragments := make([]HLS_Fragment, 0)
	oldFragmentCount := subStream.fragmentCount
	newFragmentCount := oldFragmentCount
	doneCounting := false

	for !doneCounting {
		nextFragment := subStream.fragments[newFragmentCount]
		if nextFragment != nil && subStream.fragmentsReady[newFragmentCount] {
			newFragments = append(newFragments, *nextFragment)
			delete(subStream.fragmentsReady, newFragmentCount)
			delete(subStream.fragments, newFragmentCount)
			newFragmentCount++
		} else {
			doneCounting = true
		}
	}

	if oldFragmentCount == newFragmentCount {
		return
	}

	subStream.fragmentCount = newFragmentCount

	// Update HLS Live playlist

	if subStream.livePlaylist == nil {
		subStream.livePlaylist = &HLS_PlayList{
			Version:        M3U8_DEFAULT_VERSION,
			TargetDuration: task.server.hlsTargetDuration,
			MediaSequence:  0,
			IsVOD:          false,
			IsEnded:        false,
			fragments:      make([]HLS_Fragment, 0),
		}
	}

	livePlaylist := subStream.livePlaylist

	for i := 0; i < len(newFragments); i++ {
		livePlaylist.fragments = append(livePlaylist.fragments, newFragments[i])

		if len(livePlaylist.fragments) > task.server.hlsLivePlayListSize {
			livePlaylist.fragments = livePlaylist.fragments[1:]
		}
	}

	if len(livePlaylist.fragments) > 0 {
		livePlaylist.MediaSequence = livePlaylist.fragments[0].Index
	} else {
		livePlaylist.MediaSequence = 0
	}

	livePlayListData := []byte(livePlaylist.Encode())

	if subStream.liveWriting {
		subStream.liveWritePending = true
		subStream.liveWriteData = livePlayListData
	} else {
		subStream.liveWriting = true
		go task.SaveLivePlaylist(subStream, livePlayListData)
	}

	// Update VOD playlist
	if task.record {
		// TODO
	} else {
		// Record is disabled, remove old fragments

		newRemovedFragmentCount := subStream.fragmentCount - (2 * task.server.hlsLivePlayListSize)

		if subStream.removedFragmentsCount < newRemovedFragmentCount {
			// TODO
			subStream.removedFragmentsCount = newRemovedFragmentCount
		}
	}
}

// Saves live playlist
// subStream - Reference to the sub-stream
// data - Data to write
func (task *EncodingTask) SaveLivePlaylist(subStream *SubStreamStatus, data []byte) {
	filePath := "hls/" + task.channel + "/" + task.streamId + "/" + subStream.resolution.Encode() + "/live.m3u8"
	done := false

	dataToWrite := data

	for !done {
		err := WriteFileBytes(filePath, dataToWrite)

		if err != nil {
			LogError(err)
		} else {
			task.OnLivePlaylistSaved(subStream)
		}

		task.mutex.Lock()

		if subStream.liveWritePending {
			subStream.liveWritePending = false
			dataToWrite = subStream.liveWriteData
			subStream.liveWriteData = nil
		} else {
			subStream.liveWriting = false
			done = true
		}

		task.mutex.Unlock()
	}
}

// Call after the live playlist is saved successfully
// subStream - Reference to the sub-stream
func (task *EncodingTask) OnLivePlaylistSaved(subStream *SubStreamStatus) {
	shouldAnnounce := false
	task.mutex.Lock()

	if !subStream.livePlaylistAvailable {
		subStream.livePlaylistAvailable = true
		shouldAnnounce = true
	}

	task.mutex.Unlock()

	if shouldAnnounce {
		task.server.websocketControlConnection.SendStreamAvailable(task.channel, task.streamId, "HLS-LIVE", subStream.resolution,
			"hls/"+task.channel+"/"+task.streamId+"/"+subStream.resolution.Encode()+"/live.m3u8",
		)
	}
}
