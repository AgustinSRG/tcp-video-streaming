// File storage

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var FILE_STORAGE_BASE_PATH = ""

// Initializes file storage config
func InitFileStorage() {
	FILE_STORAGE_BASE_PATH = os.Getenv("HLS_FILESYSTEM_PATH")

	if FILE_STORAGE_BASE_PATH == "" {
		LogWarning("HLS_FILESYSTEM_PATH is not set. By default the HLS files will be stored in the current working directory.")
	}
}

// Removes a file from the HLS storage
// subPath - The path inside the file system
func RemoveFile(subPath string) error {
	if strings.HasPrefix(subPath, "/") || strings.Contains(subPath, "..") {
		return errors.New("Insecure path. Cannot write the file.")
	}

	absolutePath := filepath.Join(FILE_STORAGE_BASE_PATH, subPath)

	return os.Remove(absolutePath)
}

// Removes a folder from the HLS storage
// subPath - The path inside the file system
func RemoveFolder(subPath string) error {
	if strings.HasPrefix(subPath, "/") || strings.Contains(subPath, "..") {
		return errors.New("Insecure path. Cannot write the file.")
	}

	absolutePath := filepath.Join(FILE_STORAGE_BASE_PATH, subPath)

	return os.RemoveAll(absolutePath)
}

func RemoveVOD(channel string, vod VODStreaming) {
	err := RemoveFolder("hls/" + channel + "/" + vod.StreamId + "/")

	if err != nil {
		LogError(err)
	}

	if vod.HasPreviews {
		err = RemoveFolder("img-preview/" + channel + "/" + vod.StreamId + "/")

		if err != nil {
			LogError(err)
		}
	}
}

func RemoveChannel(channel string) {
	err := RemoveFolder("hls/" + channel + "/")

	if err != nil {
		LogError(err)
	}

	err = RemoveFolder("img-preview/" + channel + "/")

	if err != nil {
		LogError(err)
	}
}
