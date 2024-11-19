// File storage (File system)

package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	FILE_PERMISSION   = 0600 // Read/Write
	FOLDER_PERMISSION = 0700 // Read/Write/Run
)

// File system storage
type FileStorageFileSystem struct {
	path string
}

// Creates instance of FileStorageFileSystem
func CreateFileStorageFileSystem() (*FileStorageFileSystem, error) {
	path := os.Getenv("HLS_FILESYSTEM_PATH")

	if path == "" {
		LogWarning("HLS_FILESYSTEM_PATH is not set. By default the HLS files will be stored in the current working directory.")
	}

	return &FileStorageFileSystem{
		path: path,
	}, nil
}

// Write a file to the HLS storage
// subPath - The path inside the file system
// data - Data to write
func (fs *FileStorageFileSystem) WriteFile(subPath string, data io.ReadCloser) error {
	if strings.HasPrefix(subPath, "/") || strings.Contains(subPath, "..") {
		return errors.New("insecure path: cannot write the file")
	}

	absolutePath := filepath.Join(fs.path, subPath)

	LogDebug("Saving File: " + absolutePath)

	dir := filepath.Dir(absolutePath)

	os.MkdirAll(dir, FOLDER_PERMISSION)

	// Create a temp file and write the content

	tempFile := absolutePath + ".tmp"

	f, err := os.OpenFile(tempFile, os.O_WRONLY|os.O_CREATE, FILE_PERMISSION)

	if err != nil {
		return err
	}

	_, err = io.Copy(f, data)

	if err != nil {
		f.Close()
		os.Remove(tempFile)
		return err
	}

	err = f.Close()

	if err != nil {
		os.Remove(tempFile)
		return err
	}

	// Move the temp file to the original location

	err = os.Rename(tempFile, absolutePath)

	if err != nil {
		os.Remove(tempFile)
		return err
	}

	return nil
}

// Write a file to the HLS storage
// subPath - The path inside the file system
// data - Data to write
func (fs *FileStorageFileSystem) WriteFileBytes(subPath string, data []byte) error {
	if strings.HasPrefix(subPath, "/") || strings.Contains(subPath, "..") {
		return errors.New("insecure path: cannot write the file")
	}

	absolutePath := filepath.Join(fs.path, subPath)

	LogDebug("Saving File: " + absolutePath)

	dir := filepath.Dir(absolutePath)

	os.MkdirAll(dir, FOLDER_PERMISSION)

	// Create a temp file and write the content

	tempFile := absolutePath + ".tmp"

	err := os.WriteFile(tempFile, data, FILE_PERMISSION)

	if err != nil {
		return err
	}

	// Move the temp file to the original location

	err = os.Rename(tempFile, absolutePath)

	if err != nil {
		os.Remove(tempFile)
		return err
	}

	return nil
}

// Removes a file from the HLS storage
// subPath - The path inside the file system
func (fs *FileStorageFileSystem) RemoveFile(subPath string) error {
	if strings.HasPrefix(subPath, "/") || strings.Contains(subPath, "..") {
		return errors.New("insecure path: cannot write the file")
	}

	absolutePath := filepath.Join(fs.path, subPath)

	return os.Remove(absolutePath)
}
