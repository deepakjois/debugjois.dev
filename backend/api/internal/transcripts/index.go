package transcripts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

const (
	BucketName     = "debugjois-dev-site"
	ObjectPrefix   = "transcripts/"
	IndexObjectKey = ObjectPrefix + "transcripts.json"
	LocationPrefix = "https://www.debugjois.dev/transcripts/"
)

type S3Client interface {
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type Index struct {
	Transcripts []ListItem `json:"transcripts"`
}

type ListItem struct {
	Title    string `json:"title"`
	Location string `json:"location"`
	Date     string `json:"date"`
}

type document struct {
	Podcast struct {
		Source struct {
			ShareTitle string `json:"share_title"`
		} `json:"source"`
		Podcast struct {
			Title string `json:"title"`
		} `json:"podcast"`
		Episode struct {
			Title         string `json:"title"`
			PublishedAt   string `json:"published_at"`
			PublishedDate string `json:"published_date"`
		} `json:"episode"`
	} `json:"podcast"`
	Deepgram json.RawMessage `json:"deepgram"`
}

type deepgramMetadataEnvelope struct {
	Metadata struct {
		Created string `json:"created"`
	} `json:"metadata"`
}

type indexState struct {
	ETag   string
	Exists bool
}

func BuildIndex(ctx context.Context, client S3Client, bucketName string) (Index, error) {
	keys, err := listTranscriptKeys(ctx, client, bucketName)
	if err != nil {
		return Index{}, err
	}

	items := make([]ListItem, 0, len(keys))
	for _, key := range keys {
		doc, err := readTranscriptDocument(ctx, client, bucketName, key)
		if err != nil {
			return Index{}, err
		}

		items = append(items, ListItem{
			Title:    transcriptTitle(doc, key),
			Location: LocationPrefix + path.Base(key),
			Date:     transcriptDate(doc),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Date != items[j].Date {
			return items[i].Date > items[j].Date
		}
		return items[i].Title < items[j].Title
	})

	return Index{Transcripts: items}, nil
}

func MarshalIndex(index Index) ([]byte, error) {
	body, err := json.Marshal(index)
	if err != nil {
		return nil, fmt.Errorf("marshal transcript index: %w", err)
	}
	return body, nil
}

func WriteIndex(ctx context.Context, client S3Client, bucketName string, body []byte) error {
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(IndexObjectKey),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("write transcript index to s3://%s/%s: %w", bucketName, IndexObjectKey, err)
	}

	return nil
}

func RefreshIndex(ctx context.Context, client S3Client, bucketName string) error {
	var lastErr error

	for attempt := 0; attempt < 2; attempt++ {
		state, err := readIndexState(ctx, client, bucketName)
		if err != nil {
			return err
		}

		index, err := BuildIndex(ctx, client, bucketName)
		if err != nil {
			return err
		}

		body, err := MarshalIndex(index)
		if err != nil {
			return err
		}

		err = putIndexConditionally(ctx, client, bucketName, body, state)
		if err == nil {
			return nil
		}
		if !isConditionalWriteConflict(err) {
			return err
		}

		lastErr = err
	}

	return fmt.Errorf("refresh transcript index after retry: %w", lastErr)
}

func listTranscriptKeys(ctx context.Context, client S3Client, bucketName string) ([]string, error) {
	var keys []string
	var continuationToken *string

	for {
		output, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucketName),
			Prefix:            aws.String(ObjectPrefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, fmt.Errorf("list transcript objects in s3://%s/%s: %w", bucketName, ObjectPrefix, err)
		}

		for _, object := range output.Contents {
			key := strings.TrimSpace(aws.ToString(object.Key))
			if key == "" || key == IndexObjectKey || !strings.HasSuffix(key, ".json") {
				continue
			}
			keys = append(keys, key)
		}

		if !aws.ToBool(output.IsTruncated) || output.NextContinuationToken == nil {
			break
		}
		continuationToken = output.NextContinuationToken
	}

	sort.Strings(keys)
	return keys, nil
}

func readTranscriptDocument(ctx context.Context, client S3Client, bucketName, key string) (document, error) {
	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return document{}, fmt.Errorf("read transcript from s3://%s/%s: %w", bucketName, key, err)
	}
	defer output.Body.Close()

	body, err := io.ReadAll(output.Body)
	if err != nil {
		return document{}, fmt.Errorf("read transcript body from s3://%s/%s: %w", bucketName, key, err)
	}

	var doc document
	if err := json.Unmarshal(body, &doc); err != nil {
		return document{}, fmt.Errorf("decode transcript JSON from s3://%s/%s: %w", bucketName, key, err)
	}

	return doc, nil
}

func readIndexState(ctx context.Context, client S3Client, bucketName string) (indexState, error) {
	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(IndexObjectKey),
	})
	if err != nil {
		if isMissingObjectError(err) {
			return indexState{}, nil
		}
		return indexState{}, fmt.Errorf("read transcript index state from s3://%s/%s: %w", bucketName, IndexObjectKey, err)
	}
	defer output.Body.Close()

	if _, err := io.Copy(io.Discard, output.Body); err != nil {
		return indexState{}, fmt.Errorf("read transcript index state from s3://%s/%s: %w", bucketName, IndexObjectKey, err)
	}

	return indexState{
		ETag:   strings.TrimSpace(aws.ToString(output.ETag)),
		Exists: true,
	}, nil
}

func putIndexConditionally(ctx context.Context, client S3Client, bucketName string, body []byte, state indexState) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(IndexObjectKey),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/json"),
	}

	if state.Exists {
		input.IfMatch = aws.String(state.ETag)
	} else {
		input.IfNoneMatch = aws.String("*")
	}

	_, err := client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("write transcript index to s3://%s/%s: %w", bucketName, IndexObjectKey, err)
	}

	return nil
}

func transcriptTitle(doc document, key string) string {
	for _, candidate := range []string{
		strings.TrimSpace(doc.Podcast.Episode.Title),
		strings.TrimSpace(doc.Podcast.Source.ShareTitle),
		strings.TrimSpace(doc.Podcast.Podcast.Title),
	} {
		if candidate != "" {
			return candidate
		}
	}

	return strings.TrimSuffix(path.Base(key), ".json")
}

func transcriptDate(doc document) string {
	if publishedDate := strings.TrimSpace(doc.Podcast.Episode.PublishedDate); publishedDate != "" {
		if _, err := time.Parse(time.DateOnly, publishedDate); err == nil {
			return publishedDate
		}
	}

	if publishedAt := strings.TrimSpace(doc.Podcast.Episode.PublishedAt); publishedAt != "" {
		if timestamp, ok := parseTimestamp(publishedAt); ok {
			return timestamp.Format(time.DateOnly)
		}
	}

	var deepgram deepgramMetadataEnvelope
	if len(doc.Deepgram) == 0 || json.Unmarshal(doc.Deepgram, &deepgram) != nil {
		return ""
	}

	if createdAt, ok := parseTimestamp(strings.TrimSpace(deepgram.Metadata.Created)); ok {
		return createdAt.Format(time.DateOnly)
	}

	return ""
}

func parseTimestamp(value string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		timestamp, err := time.Parse(layout, value)
		if err == nil {
			return timestamp, true
		}
	}

	return time.Time{}, false
}

func isMissingObjectError(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	switch apiErr.ErrorCode() {
	case "NoSuchKey", "NotFound":
		return true
	default:
		return false
	}
}

func isConditionalWriteConflict(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	switch apiErr.ErrorCode() {
	case "PreconditionFailed", "ConditionalRequestConflict":
		return true
	default:
		return false
	}
}
