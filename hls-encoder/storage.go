// File storage

package main

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	FILE_PERMISSION   = 0600 // Read/Write
	FOLDER_PERMISSION = 0700 // Read/Write/Run
)

var FILE_STORAGE_BASE_PATH = ""

// Initializes file storage config
func InitFileStorage() {
	FILE_STORAGE_BASE_PATH = os.Getenv("HLS_FILESYSTEM_PATH")

	if FILE_STORAGE_BASE_PATH == "" {
		LogWarning("HLS_FILESYSTEM_PATH is not set. By default the HLS files will be stored in the current working directory.")
	}
}

// Write a file to the HLS storage
// subPath - The path inside the file system
// data - Data to write
func WriteFile(subPath string, data io.ReadCloser) error {
	if strings.HasPrefix(subPath, "/") || strings.Contains(subPath, "..") {
		return errors.New("Insecure path. Cannot write the file.")
	}

	absolutePath := filepath.Join(FILE_STORAGE_BASE_PATH, subPath)

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
func WriteFileBytes(subPath string, data []byte) error {
	if strings.HasPrefix(subPath, "/") || strings.Contains(subPath, "..") {
		return errors.New("Insecure path. Cannot write the file.")
	}

	absolutePath := filepath.Join(FILE_STORAGE_BASE_PATH, subPath)

	LogDebug("Saving File: " + absolutePath)

	dir := filepath.Dir(absolutePath)

	os.MkdirAll(dir, FOLDER_PERMISSION)

	// Create a temp file and write the content

	tempFile := absolutePath + ".tmp"

	err := ioutil.WriteFile(tempFile, data, FILE_PERMISSION)

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
func RemoveFile(subPath string) error {
	if strings.HasPrefix(subPath, "/") || strings.Contains(subPath, "..") {
		return errors.New("Insecure path. Cannot write the file.")
	}

	absolutePath := filepath.Join(FILE_STORAGE_BASE_PATH, subPath)

	return os.Remove(absolutePath)
}
