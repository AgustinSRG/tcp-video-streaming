// File storage (Azure Blob Storage)

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
)

// File storage for Azure Boh storage
type FileStorageAzureBlobStorage struct {
	// Client
	client *azblob.Client

	// Container
	container string
}

// Creates instance of FileStorageAzureBlobStorage
func NewFileStorageAzureBlobStorage() (*FileStorageAzureBlobStorage, error) {
	accountName := os.Getenv("AZURE_STORAGE_ACCOUNT")

	if accountName == "" {
		LogWarning("AZURE_STORAGE_ACCOUNT is empty")
		return nil, errors.New("storage account name not specified")
	}

	url := "https://" + accountName + ".blob.core.windows.net/"

	credential, err := azidentity.NewClientSecretCredential(os.Getenv("AZURE_TENANT_ID"), os.Getenv("AZURE_CLIENT_ID"), os.Getenv("AZURE_CLIENT_SECRET"), nil)

	if err != nil {
		return nil, err
	}

	client, err := azblob.NewClient(url, credential, nil)

	if err != nil {
		return nil, err
	}

	return &FileStorageAzureBlobStorage{
		client:    client,
		container: os.Getenv("AZURE_STORAGE_CONTAINER"),
	}, nil
}

// Write a file to the HLS storage
// subPath - The path inside the file system
// data - Data to write
func (fs *FileStorageAzureBlobStorage) WriteFile(subPath string, data io.Reader) error {
	ext := getFileExtension(subPath)

	contentType := "application/octet-stream"
	cacheControl := "max-age=31536000"

	switch ext {
	case "json":
		contentType = "application/json"
		cacheControl = "no-cache"
	case "jpg":
		contentType = "image/jpg"
	case "m3u8":
		contentType = "application/x-mpegURL"
		cacheControl = "no-cache"
	case "ts":
		contentType = "video/mp2t"
	}

	_, err := fs.client.UploadStream(context.Background(), fs.container, subPath, data, &azblob.UploadStreamOptions{
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType:  &contentType,
			BlobCacheControl: &cacheControl,
		},
	})

	return err
}

// Write a file to the HLS storage
// subPath - The path inside the file system
// data - Data to write
func (fs *FileStorageAzureBlobStorage) WriteFileBytes(subPath string, data []byte) error {
	return fs.WriteFile(subPath, bytes.NewReader(data))
}

// Removes a file from the HLS storage
// subPath - The path inside the file system
func (fs *FileStorageAzureBlobStorage) RemoveFile(subPath string) error {
	_, err := fs.client.DeleteBlob(context.Background(), fs.container, subPath, nil)

	return err
}
