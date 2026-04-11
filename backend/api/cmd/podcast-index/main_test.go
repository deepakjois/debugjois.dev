package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/deepakjois/debugjois.dev/backend/api/internal/transcripts"
)

type fakeTranscriptIndexS3Client struct {
	listOutputs []*s3.ListObjectsV2Output
	objects     map[string]fakeS3Object
	listCalls   int
	putBucket   string
	putKey      string
	putBody     []byte
	putType     string
}

type fakeS3Object struct {
	body string
	etag string
}

func (f *fakeTranscriptIndexS3Client) ListObjectsV2(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if f.listCalls >= len(f.listOutputs) {
		return &s3.ListObjectsV2Output{}, nil
	}
	output := f.listOutputs[f.listCalls]
	f.listCalls++
	return output, nil
}

func (f *fakeTranscriptIndexS3Client) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	object, ok := f.objects[aws.ToString(input.Key)]
	if !ok {
		return nil, fmt.Errorf("unexpected key %q", aws.ToString(input.Key))
	}
	return &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader(object.body)),
		ETag: aws.String(object.etag),
	}, nil
}

func (f *fakeTranscriptIndexS3Client) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	body, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	f.putBucket = aws.ToString(input.Bucket)
	f.putKey = aws.ToString(input.Key)
	f.putBody = body
	f.putType = aws.ToString(input.ContentType)
	return &s3.PutObjectOutput{}, nil
}

func TestRunWritesTranscriptIndexWhenRequested(t *testing.T) {
	originalFactory := newTranscriptIndexS3ClientFunc
	defer func() {
		newTranscriptIndexS3ClientFunc = originalFactory
	}()

	client := &fakeTranscriptIndexS3Client{
		listOutputs: []*s3.ListObjectsV2Output{{
			Contents: newS3Objects("transcripts/example-episode.json"),
		}},
		objects: map[string]fakeS3Object{
			"transcripts/example-episode.json": {
				body: mustMarshalTranscriptJSON(t, map[string]any{
					"podcast": map[string]any{
						"podcast": map[string]any{"title": "Example Podcast"},
						"episode": map[string]any{
							"title":          "Example Episode",
							"published_date": "2026-04-06",
						},
					},
					"deepgram": map[string]any{"metadata": map[string]any{"created": "2026-04-08T19:18:58.618Z"}},
				}),
			},
		},
	}
	newTranscriptIndexS3ClientFunc = func(context.Context, string) (transcripts.S3Client, error) {
		return client, nil
	}

	var stdout bytes.Buffer
	if err := run(context.Background(), []string{"--write"}, &stdout); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if client.putBucket != transcripts.BucketName {
		t.Fatalf("unexpected bucket %q", client.putBucket)
	}
	if client.putKey != transcripts.IndexObjectKey {
		t.Fatalf("unexpected key %q", client.putKey)
	}
	if client.putType != "application/json" {
		t.Fatalf("unexpected content type %q", client.putType)
	}

	var written transcripts.Index
	if err := json.Unmarshal(client.putBody, &written); err != nil {
		t.Fatalf("unmarshal written body: %v", err)
	}
	if len(written.Transcripts) != 1 || written.Transcripts[0].Title != "Example Episode" {
		t.Fatalf("unexpected written transcripts %#v", written.Transcripts)
	}

	var printed transcripts.Index
	if err := json.Unmarshal(stdout.Bytes(), &printed); err != nil {
		t.Fatalf("unmarshal stdout body: %v", err)
	}
	if len(printed.Transcripts) != 1 || printed.Transcripts[0].Location == "" {
		t.Fatalf("unexpected stdout transcripts %#v", printed.Transcripts)
	}
}

func newS3Objects(keys ...string) []s3types.Object {
	objects := make([]s3types.Object, 0, len(keys))
	for _, key := range keys {
		objects = append(objects, s3types.Object{Key: aws.String(key)})
	}
	return objects
}

func mustMarshalTranscriptJSON(t *testing.T, doc map[string]any) string {
	t.Helper()
	body, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	return string(body)
}
