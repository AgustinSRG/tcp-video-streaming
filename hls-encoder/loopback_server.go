// Loopback server

package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Serves HTTP request for the loopback server
// w - Response writer
// req - Client request
func (server *HLS_Encoder_Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	LogDebug(req.Method + " " + req.RequestURI)

	if req.Method != "PUT" {
		w.WriteHeader(200)
		fmt.Fprintf(w, "HLS encoding server - Version "+VERSION)
		return
	}

	// /{TYPE}/{Stream-Channel}/{Stream-ID}/{RESOLUTION}/{FILE}
	uriParts := strings.Split(req.RequestURI, "/")

	if len(uriParts) != 6 {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Bad request.")
		return
	}

	if uriParts[1] == "hls" {
		resolution, err := DecodeResolution(uriParts[4])

		if err != nil {
			LogError(err)
			w.WriteHeader(400)
			fmt.Fprintf(w, "Bad request: Invalid resolution")
			return
		}

		server.HandleRequestHLS(w, req, uriParts[2], uriParts[3], resolution, uriParts[5])
	} else if uriParts[1] == "img-preview" {
		server.HandleRequestImagePreview(w, req, uriParts[2], uriParts[3], DecodePreviewsConfiguration(uriParts[4], "-"), uriParts[5])
	} else {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Bad request: Invalid stream type")
		return
	}
}

// Handles HLS PUT requests
// w - Response writer
// req - Client request
// channel - Channel ID
// streamId - Stream ID
// resolution - Resolution
// file - File name
func (server *HLS_Encoder_Server) HandleRequestHLS(w http.ResponseWriter, req *http.Request, channel string, streamId string, resolution Resolution, file string) {
	task := server.GetTask(channel, streamId)

	if task == nil {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Task not found")
		return
	}

	if strings.HasSuffix(file, ".m3u8") {
		server.HandleRequestHLS_M3U8(w, req, task, resolution, file)
	} else if strings.HasSuffix(file, ".ts") {
		server.HandleRequestHLS_TS(w, req, task, resolution, file)
	} else {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Bad request: Invalid HLS file")
		return
	}
}

// Handles M3U8 playlist PUT requests
// w - Response writer
// req - Client request
// task - Reference to the task
// resolution - Resolution
// file - File name
func (server *HLS_Encoder_Server) HandleRequestHLS_M3U8(w http.ResponseWriter, req *http.Request, task *EncodingTask, resolution Resolution, file string) {
	// Read the body

	bodyData, err := io.ReadAll(req.Body)

	if err != nil {
		LogError(err)
		w.WriteHeader(500)
		fmt.Fprintf(w, "Internal server error.")
		return
	}

	// Decode playlist

	playList := DecodeHLSPlayList(string(bodyData))

	LogDebug("Decoded playlist, with " + fmt.Sprint(len(playList.fragments)) + " fragments")

	// Notice task

	task.OnPlaylistUpdate(resolution, playList)

	w.WriteHeader(200)
}

// Handles TS video PUT requests
// w - Response writer
// req - Client request
// task - Reference to the task
// resolution - Resolution
// file - File name
func (server *HLS_Encoder_Server) HandleRequestHLS_TS(w http.ResponseWriter, req *http.Request, task *EncodingTask, resolution Resolution, file string) {
	fileParts := strings.Split(file, ".")

	if len(fileParts) != 2 {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Bad request: Invalid TS file")
		return
	}

	fileIndex, err := strconv.Atoi(fileParts[0])

	if err != nil {
		LogError(err)
		w.WriteHeader(400)
		fmt.Fprintf(w, "Bad request: Invalid TS file")
		return
	}

	// Write file

	fragmentPath := "hls/" + task.channel + "/" + task.streamId + "/" + resolution.Encode() + "/" + file

	err = server.storage.WriteFile(fragmentPath, req.Body)

	if err != nil {
		LogError(err)
		w.WriteHeader(500)
		fmt.Fprintf(w, "Internal server error.")
		return
	}

	// Notice the task that the preview image is ready

	task.OnFragmentReady(resolution, fileIndex)

	w.WriteHeader(200)
}

// Handles Image previews PUT requests
// w - Response writer
// req - Client request
// channel - Channel ID
// streamId - Stream ID
// config - Previews configuration
// file - File name
func (server *HLS_Encoder_Server) HandleRequestImagePreview(w http.ResponseWriter, req *http.Request, channel string, streamId string, config PreviewsConfiguration, file string) {
	task := server.GetTask(channel, streamId)

	if task == nil {
		w.WriteHeader(404)
		fmt.Fprintf(w, "Task not found")
		return
	}

	if !strings.HasSuffix(file, ".jpg") {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Bad request: Invalid preview file")
		return
	}

	fileParts := strings.Split(file, ".")

	if len(fileParts) != 2 {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Bad request: Invalid preview file")
		return
	}

	fileIndex, err := strconv.Atoi(fileParts[0])

	if err != nil {
		LogError(err)
		w.WriteHeader(400)
		fmt.Fprintf(w, "Bad request: Invalid preview file")
		return
	}

	// Write preview

	previewPath := "img-preview/" + channel + "/" + streamId + "/" + config.Encode("-") + "/" + file

	err = server.storage.WriteFile(previewPath, req.Body)

	if err != nil {
		LogError(err)
		w.WriteHeader(500)
		fmt.Fprintf(w, "Internal server error.")
		return
	}

	// Notice the task that the preview image is ready

	task.OnPreviewImageReady(fileIndex)

	w.WriteHeader(200)
}
