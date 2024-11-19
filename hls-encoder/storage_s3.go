// AWS S3 storage system

package main

import (
	"bytes"
	"io"
	"os"

	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Credentials provider for the AWS client
type FileStorageAwsS3CredentialsProvider struct {
	accessKeyId     string
	secretAccessKey string
}

// Retrieves credentials
func (p *FileStorageAwsS3CredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     p.accessKeyId,
		SecretAccessKey: p.secretAccessKey,
		CanExpire:       false,
	}, nil
}

// File storage for AWS S3
type FileStorageAwsS3 struct {
	// Client
	client *s3.Client

	// Bucket name
	bucket string
}

// Creates instance of FileStorageAwsS3
func CreateFileStorageAwsS3() (*FileStorageAwsS3, error) {
	credentialsProvider := &FileStorageAwsS3CredentialsProvider{
		accessKeyId:     os.Getenv("AWS_ACCESS_KEY_ID"),
		secretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv("AWS_REGION")), config.WithCredentialsProvider(credentialsProvider))

	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)

	return &FileStorageAwsS3{
		client: client,
		bucket: os.Getenv("AWS_S3_BUCKET"),
	}, nil
}

// Write a file to the HLS storage
// subPath - The path inside the file system
// data - Data to write
func (fs *FileStorageAwsS3) WriteFile(subPath string, data io.Reader) error {
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

	_, err := fs.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:       &fs.bucket,
		Key:          &subPath,
		Body:         data,
		ContentType:  &contentType,
		CacheControl: &cacheControl,
	})

	return err
}

// Write a file to the HLS storage
// subPath - The path inside the file system
// data - Data to write
func (fs *FileStorageAwsS3) WriteFileBytes(subPath string, data []byte) error {
	return fs.WriteFile(subPath, bytes.NewReader(data))
}

// Removes a file from the HLS storage
// subPath - The path inside the file system
func (fs *FileStorageAwsS3) RemoveFile(subPath string) error {
	_, err := fs.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: &fs.bucket,
		Key:    &subPath,
	})

	return err
}
