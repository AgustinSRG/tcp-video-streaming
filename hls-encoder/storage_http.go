// HTTP storage system

package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// File storage by sending HTTP requests
type FileStorageHttp struct {
	// Base URL
	url string

	// Auth header
	authHeader string
}

// Creates new instance of FileStorageHttp
func CreateFileStorageHttp() (*FileStorageHttp, error) {
	urlStr := os.Getenv("HLS_STORAGE_HTTP_URL")

	u, err := url.Parse(urlStr)

	if err != nil {
		LogWarning("Invalid URL provided by HLS_STORAGE_HTTP_URL")
		return nil, err
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		LogWarning("Invalid URL provided by HLS_STORAGE_HTTP_URL")
		return nil, errors.New("url scheme must be https or http")
	}

	authMethod := strings.ToUpper(os.Getenv("HLS_STORAGE_HTTP_AUTH"))
	authorization := ""

	switch authMethod {
	case "BASIC":
		user := os.Getenv("HLS_STORAGE_HTTP_USER")
		password := os.Getenv("HLS_STORAGE_HTTP_PASSWORD")
		authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
	case "BEARER":
		token := os.Getenv("HLS_STORAGE_HTTP_TOKEN")
		authorization = "Bearer " + token
	case "CUSTOM":
		authorization = os.Getenv("HLS_STORAGE_HTTP_AUTH_CUSTOM")
	}

	return &FileStorageHttp{
		url:        urlStr,
		authHeader: authorization,
	}, nil
}

// Write a file to the HLS storage
// subPath - The path inside the file system
// data - Data to write
func (fs *FileStorageHttp) WriteFile(subPath string, data io.Reader) error {
	joinedUrl, err := url.JoinPath(fs.url, subPath)

	if err != nil {
		return err
	}

	client := &http.Client{}

	req, err := http.NewRequest("PUT", joinedUrl, data)

	if err != nil {
		return err
	}

	if fs.authHeader != "" {
		req.Header.Set("Authorization", fs.authHeader)
	}

	LogDebug("HTTP request: PUT " + joinedUrl)

	res, err := client.Do(req)

	if err != nil {
		return err
	}

	if res.StatusCode == 200 {
		return nil
	} else {
		return errors.New("status code " + fmt.Sprint(res.StatusCode) + " for " + joinedUrl)
	}
}

// Write a file to the HLS storage
// subPath - The path inside the file system
// data - Data to write
func (fs *FileStorageHttp) WriteFileBytes(subPath string, data []byte) error {
	return fs.WriteFile(subPath, bytes.NewReader(data))
}

// Removes a file from the HLS storage
// subPath - The path inside the file system
func (fs *FileStorageHttp) RemoveFile(subPath string) error {
	joinedUrl, err := url.JoinPath(fs.url, subPath)

	if err != nil {
		return err
	}

	client := &http.Client{}

	req, err := http.NewRequest("DELETE", joinedUrl, nil)

	if err != nil {
		return err
	}

	if fs.authHeader != "" {
		req.Header.Set("Authorization", fs.authHeader)
	}

	LogDebug("HTTP request: DELETE " + joinedUrl)

	res, err := client.Do(req)

	if err != nil {
		return err
	}

	if res.StatusCode == 200 || res.StatusCode == 404 {
		return nil
	} else {
		return errors.New("status code " + fmt.Sprint(res.StatusCode) + " for " + joinedUrl)
	}
}
