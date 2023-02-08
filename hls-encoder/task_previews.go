// Image previews related task features

package main

import "encoding/json"

// Call when a new preview image is ready
// previewIndex - Index of the new image
func (task *EncodingTask) OnPreviewImageReady(previewIndex int) {
	task.mutex.Lock()
	defer task.mutex.Unlock()

	if previewIndex < task.previewsCount {
		return // Ignore already added images
	}

	task.previewsReady[previewIndex] = true

	oldCount := task.previewsCount
	newCount := oldCount
	doneCounting := false

	for !doneCounting {
		if task.previewsReady[newCount] {
			delete(task.previewsReady, newCount)
			newCount++
		} else {
			doneCounting = true
		}
	}

	if oldCount != newCount {
		// Update new count
		task.previewsCount = newCount

		// Update index file
		newIndexFile := PreviewsIndexFile{
			IndexStart: 0,
			Count:      newCount,
			Pattern:    "%d.jpg",
		}

		data, err := json.Marshal(newIndexFile)

		if err != nil {
			LogError(err)
			return
		}

		if task.writingPreviewsIndex {
			task.pendingWritePreviewsIndex = true
			task.pendingPreviewsIndexContent = data
		} else {
			task.writingPreviewsIndex = true
			go task.SaveImagePreviewsIndex(data)
		}
	}
}

// Call after the image previews index is successfully saved
func (task *EncodingTask) OnImagePreviewsIndexSaved() {
	shouldAnnounce := false
	task.mutex.Lock()

	if !task.previewsAvailable {
		task.previewsAvailable = true
		shouldAnnounce = true
	}

	task.mutex.Unlock()

	if shouldAnnounce {
		task.server.websocketControlConnection.SendStreamAvailable(task.channel, task.streamId, "IMG-PREVIEW", Resolution{
			width:  task.previews.width,
			height: task.previews.height,
			fps:    task.previews.delaySeconds},
			"img-preview/"+task.channel+"/"+task.streamId+"/"+task.previews.Encode("-")+"/index.json",
		)
	}
}

// Saves image previews index
// data - Contents of the index file
func (task *EncodingTask) SaveImagePreviewsIndex(data []byte) {
	filePath := "img-preview/" + task.channel + "/" + task.streamId + "/" + task.previews.Encode("-") + "/index.json"
	done := false

	dataToWrite := data

	for !done {
		err := WriteFileBytes(filePath, dataToWrite)

		if err != nil {
			LogError(err)
		} else {
			task.OnImagePreviewsIndexSaved()
		}

		task.mutex.Lock()

		if task.pendingWritePreviewsIndex {
			task.pendingWritePreviewsIndex = false
			dataToWrite = task.pendingPreviewsIndexContent
			task.pendingPreviewsIndexContent = nil
		} else {
			task.writingPreviewsIndex = false
			done = true
		}

		task.mutex.Unlock()
	}
}
