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
	WriteFile(subPath string, data io.Reader) error

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
	case "", "FILESYSTEM":
		return CreateFileStorageFileSystem()
	case "HTTP", "HTTPS":
		return CreateFileStorageHttp()
	case "S3":
		return CreateFileStorageAwsS3()
	case "AZ", "AZURE", "AZURE_BLOB_STORAGE":
		return NewFileStorageAzureBlobStorage()
	default:
		return nil, errors.New("unknown storage type: " + storageType)
	}
}
