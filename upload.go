package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type UploadCmd struct {
	SourceDir string `help:"Source directory to upload" default:"build"`
	Bucket    string `help:"S3 bucket name" default:"debugjois-dev-site"`
	DryRun    bool   `help:"Perform a dry run without actually uploading files" default:"false" name:"dryrun"`
}

func (u *UploadCmd) Run() error {
	ctx := context.Background()

	// load default AWS config
	sess, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %v", err)
	}

	client := s3.NewFromConfig(sess)

	// Get existing S3 objects
	objects, err := getS3Objects(ctx, client, u.Bucket)
	if err != nil {
		return fmt.Errorf("failed to get existing S3 objects: %v", err)
	}

	// Create a buffered channel for files
	filesChan := make(chan string, 100)

	// Use atomic counter for error tracking
	var errorCount atomic.Int64

	// Declare WaitGroup
	var wg sync.WaitGroup

	// Create a pool of 10 worker goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range filesChan {
				key := filepath.ToSlash(path)
				if err := u.upload(ctx, client, path, key, objects); err != nil {
					log.Printf("Error processing %s: %v", key, err)
					errorCount.Add(1)
				}
			}
		}()
	}

	// Walk the source directory using fs.WalkDir
	err = fs.WalkDir(os.DirFS(u.SourceDir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			filesChan <- path
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking the path %s: %v", u.SourceDir, err)
	}

	close(filesChan)
	wg.Wait()

	if errorCount.Load() > 0 {
		return fmt.Errorf("%d errors occurred during upload", errorCount.Load())
	}

	if u.DryRun {
		fmt.Println("Dry run complete!")
	} else {
		fmt.Println("Upload complete!")
	}
	return nil
}

func (u *UploadCmd) upload(ctx context.Context, client *s3.Client, path, key string, objects map[string]string) error {
	// Construct the full file path using SourceDir
	file := filepath.Join(u.SourceDir, path)

	// Check if file needs to be uploaded
	content, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	contentMD5 := fmt.Sprintf("%x", md5.Sum(content))
	if etag, exists := objects[key]; exists && etag == contentMD5 {
		// fmt.Printf("Skipped (unchanged): %s\n", key)
		return nil // File hasn't changed, skip upload
	}

	// Determine content type
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" && filepath.Ext(path) == "" {
		// If no extension, set to text/html
		contentType = "text/html"
	}

	if u.DryRun {
		fmt.Printf("Would upload: %s (Content-Type: %s)\n", key, contentType)
		return nil
	}

	// Upload file
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(u.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return err
	}

	fmt.Printf("Uploaded: %s\n", key)
	return nil
}

func getS3Objects(ctx context.Context, client *s3.Client, bucket string) (map[string]string, error) {
	result := make(map[string]string)
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			result[*obj.Key] = strings.Trim(*obj.ETag, "\"")
		}
	}
	return result, nil
}
