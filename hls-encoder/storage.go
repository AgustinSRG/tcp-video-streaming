// File storage

package main

import (
	"errors"
	"io"
	"os"
)

// File storage system interface
type FileStorageSystem interface {
	// Write a file to the HLS storage
	// subPath - The path inside the file system
	// data - Data to write
	WriteFile(subPath string, data io.ReadCloser) error

	// Write a file to the HLS storage
	// subPath - The path inside the file system
	// data - Data to write
	WriteFileBytes(subPath string, data []byte) error

	// Removes a file from the HLS storage
	// subPath - The path inside the file system
	RemoveFile(subPath string) error
}

// Creates file storage system
func CreateFileStorageSystem() (FileStorageSystem, error) {
	storageType := os.Getenv("HLS_STORAGE_TYPE")

	switch storageType {
	case "":
	case "FILESYSTEM":
		return CreateFileStorageFileSystem()
	}

	return nil, errors.New("unknown storage type: " + storageType)
}
