// Server HLS

package main

import (
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
)

func hls_servePlaylist(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)

	channel := vars["channel"]

	if !validateStreamIDString(channel) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	streamId := vars["id"]

	if !validateStreamIDString(streamId) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	resolution, err := DecodeResolution(vars["resolution"])

	if err != nil {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	fileName := vars["file"] + ".m3u8"

	basePath := FILE_STORAGE_BASE_PATH

	if basePath == "" {
		ReturnAPIError(response, 404, "NOT_FOUND", "File not found.")
		return
	}

	fullPath := filepath.Join(basePath, "hls", channel, streamId, resolution.Encode(), fileName)

	response.Header().Add("Cache-Control", "no-cache") // No cache for playlists
	response.Header().Add("Content-Type", "application/x-mpegURL")

	http.ServeFile(response, request, fullPath)
}

func hls_serveFragment(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)

	channel := vars["channel"]

	if !validateStreamIDString(channel) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	streamId := vars["id"]

	if !validateStreamIDString(streamId) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	resolution, err := DecodeResolution(vars["resolution"])

	if err != nil {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	fileName := vars["file"] + ".ts"

	basePath := FILE_STORAGE_BASE_PATH

	if basePath == "" {
		ReturnAPIError(response, 404, "NOT_FOUND", "File not found.")
		return
	}

	fullPath := filepath.Join(basePath, "hls", channel, streamId, resolution.Encode(), fileName)

	response.Header().Add("Cache-Control", "max-age=31536000")
	response.Header().Add("Content-Type", "video/MP2T")

	http.ServeFile(response, request, fullPath)
}

func hls_servePreviewsIndex(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)

	channel := vars["channel"]

	if !validateStreamIDString(channel) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	streamId := vars["id"]

	if !validateStreamIDString(streamId) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	resolution, err := DecodeResolution(vars["resolution"])

	if err != nil {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	fileName := vars["file"] + ".json"

	basePath := FILE_STORAGE_BASE_PATH

	if basePath == "" {
		ReturnAPIError(response, 404, "NOT_FOUND", "File not found.")
		return
	}

	fullPath := filepath.Join(basePath, "img-preview", channel, streamId, resolution.Encode(), fileName)

	response.Header().Add("Cache-Control", "no-cache") // No cache for index files
	response.Header().Add("Content-Type", "application/json")

	http.ServeFile(response, request, fullPath)
}

func hls_servePreviewImage(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)

	channel := vars["channel"]

	if !validateStreamIDString(channel) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	streamId := vars["id"]

	if !validateStreamIDString(streamId) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	resolution, err := DecodeResolution(vars["resolution"])

	if err != nil {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	fileName := vars["file"] + ".jpg"

	basePath := FILE_STORAGE_BASE_PATH

	if basePath == "" {
		ReturnAPIError(response, 404, "NOT_FOUND", "File not found.")
		return
	}

	fullPath := filepath.Join(basePath, "img-preview", channel, streamId, resolution.Encode(), fileName)

	response.Header().Add("Cache-Control", "max-age=31536000")
	response.Header().Add("Content-Type", "image/jpeg")

	http.ServeFile(response, request, fullPath)
}
