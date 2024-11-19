// HLS related task features

package main

import "fmt"

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
			vodFragmentBuffer:     make([]HLS_Fragment, 0),
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
		// Push fragments into the buffer
		subStream.vodFragmentBuffer = append(subStream.vodFragmentBuffer, newFragments...)
		// Update playlist
		task.updateVODInternal(subStream)
	} else {
		// Record is disabled, remove old fragments

		newRemovedFragmentCount := subStream.fragmentCount - (2 * task.server.hlsLivePlayListSize)

		if subStream.removedFragmentsCount < newRemovedFragmentCount {
			go task.RemoveFragments(subStream, subStream.removedFragmentsCount, newRemovedFragmentCount)
			subStream.removedFragmentsCount = newRemovedFragmentCount
		}
	}

	// Check fragments limit

	if subStream.fragmentCount >= task.server.hlsMaxFragmentCount {
		// Limit reached, kill process
		task.killed = true
		if task.process != nil {
			task.process.Kill()
		}
	}
}

// Updates the VOD playlist
// subStream - Reference to the sub-stream
func (task *EncodingTask) updateVODInternal(subStream *SubStreamStatus) {
	if len(subStream.vodFragmentBuffer) == 0 {
		return // Empty buffer
	}

	if subStream.vodPlaylist == nil {
		subStream.vodPlaylist = &HLS_PlayList{
			Version:        M3U8_DEFAULT_VERSION,
			TargetDuration: task.server.hlsTargetDuration,
			MediaSequence:  0,
			IsVOD:          true,
			IsEnded:        true,
			fragments:      make([]HLS_Fragment, 0),
		}
	}

	playlistHasChanged := false

	if len(subStream.vodPlaylist.fragments) >= task.server.hlsVODPlaylistMaxSize {
		if subStream.vodWriting {
			return
		}

		// Create new VOD playlist, the old one reached it's limit

		subStream.vodIndex++
		subStream.vodPlaylist = &HLS_PlayList{
			Version:        M3U8_DEFAULT_VERSION,
			TargetDuration: task.server.hlsTargetDuration,
			MediaSequence:  0,
			IsVOD:          true,
			IsEnded:        true,
			fragments:      make([]HLS_Fragment, 0),
		}
		subStream.vodPlaylistAvailable = false
	}

	for len(subStream.vodFragmentBuffer) > 0 && len(subStream.vodPlaylist.fragments) < task.server.hlsVODPlaylistMaxSize {
		// Append fragment
		subStream.vodPlaylist.fragments = append(subStream.vodPlaylist.fragments, subStream.vodFragmentBuffer[0])
		playlistHasChanged = true
		// Remove fragment from buffer
		subStream.vodFragmentBuffer = subStream.vodFragmentBuffer[1:]
	}

	if playlistHasChanged {
		// If changed, save the VOD playlist
		playlistData := []byte(subStream.vodPlaylist.Encode())

		if subStream.vodWriting {
			subStream.vodWritePending = true
			subStream.vodWriteData = playlistData
		} else {
			subStream.vodWriting = true
			go task.SaveVODPlaylist(subStream, playlistData)
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
		err := task.server.storage.WriteFileBytes(filePath, dataToWrite)

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

// Removes TS fragments
// subStream - The sub-stream reference
// fromIndex - Start of the range (Inclusive)
// toIndex - End of the range (exclusive)
func (task *EncodingTask) RemoveFragments(subStream *SubStreamStatus, fromIndex int, toIndex int) {
	for i := fromIndex; i < toIndex; i++ {
		filePath := "hls/" + task.channel + "/" + task.streamId + "/" + subStream.resolution.Encode() + "/" + fmt.Sprint(i) + ".ts"
		err := task.server.storage.RemoveFile(filePath)
		if err != nil {
			task.debug("Could not remove file: " + filePath + " | Error: " + err.Error())
		} else {
			task.debug("Removed file: " + filePath)
		}
	}
}

// Saves VOD playlist
// subStream - Reference to the sub-stream
// data - Data to write
func (task *EncodingTask) SaveVODPlaylist(subStream *SubStreamStatus, data []byte) {
	filePath := "hls/" + task.channel + "/" + task.streamId + "/" + subStream.resolution.Encode() + "/vod-" + fmt.Sprint(subStream.vodIndex) + ".m3u8"
	done := false

	dataToWrite := data

	for !done {
		err := task.server.storage.WriteFileBytes(filePath, dataToWrite)

		if err != nil {
			LogError(err)
		} else {
			task.OnVODPlaylistSaved(subStream)
		}

		task.mutex.Lock()

		if subStream.vodWritePending {
			subStream.vodWritePending = false
			dataToWrite = subStream.vodWriteData
			subStream.vodWriteData = nil
		} else {
			subStream.vodWriting = false
			done = true

			task.updateVODInternal(subStream)
		}

		task.mutex.Unlock()
	}
}

// Call after the VOD playlist is saved successfully
// subStream - Reference to the sub-stream
func (task *EncodingTask) OnVODPlaylistSaved(subStream *SubStreamStatus) {
	shouldAnnounce := false
	indexToAnnounce := 0
	task.mutex.Lock()

	if !subStream.vodPlaylistAvailable {
		subStream.vodPlaylistAvailable = true
		shouldAnnounce = true
		indexToAnnounce = subStream.vodIndex
	}

	task.mutex.Unlock()

	if shouldAnnounce {
		task.server.websocketControlConnection.SendStreamAvailable(task.channel, task.streamId, "HLS-VOD", subStream.resolution,
			"hls/"+task.channel+"/"+task.streamId+"/"+subStream.resolution.Encode()+"/vod-"+fmt.Sprint(indexToAnnounce)+".m3u8",
		)
	}
}
