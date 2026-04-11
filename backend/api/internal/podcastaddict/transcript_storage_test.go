package podcastaddict

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/deepakjois/debugjois.dev/backend/api/internal/transcripts"
)

type fakeAPIError struct {
	code string
	msg  string
}

func (e *fakeAPIError) Error() string {
	return e.msg
}

func (e *fakeAPIError) ErrorCode() string {
	return e.code
}

func (e *fakeAPIError) ErrorMessage() string {
	return e.msg
}

func (e *fakeAPIError) ErrorFault() smithy.ErrorFault {
	return smithy.FaultClient
}

type fakeTranscriptS3Client struct {
	putInputs           []*s3.PutObjectInput
	listOutputs         []*s3.ListObjectsV2Output
	listCalls           int
	objects             map[string]fakeTranscriptObject
	failIndexWriteCount int
	lastIndexPutIfMatch string
	lastIndexPutIfNone  string
}

type fakeTranscriptObject struct {
	body string
	etag string
}

func (f *fakeTranscriptS3Client) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	body, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	input.Body = bytes.NewReader(body)
	f.putInputs = append(f.putInputs, input)

	key := aws.ToString(input.Key)
	if key == transcripts.IndexObjectKey {
		f.lastIndexPutIfMatch = aws.ToString(input.IfMatch)
		f.lastIndexPutIfNone = aws.ToString(input.IfNoneMatch)
		if f.failIndexWriteCount > 0 {
			f.failIndexWriteCount--
			return nil, &fakeAPIError{code: "ConditionalRequestConflict", msg: "conflict"}
		}
	}

	if f.objects == nil {
		f.objects = make(map[string]fakeTranscriptObject)
	}
	f.objects[key] = fakeTranscriptObject{body: string(body), etag: fmt.Sprintf("etag-%d", len(f.putInputs))}
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeTranscriptS3Client) ListObjectsV2(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if f.listCalls >= len(f.listOutputs) {
		return &s3.ListObjectsV2Output{}, nil
	}
	output := f.listOutputs[f.listCalls]
	f.listCalls++
	return output, nil
}

func (f *fakeTranscriptS3Client) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	object, ok := f.objects[aws.ToString(input.Key)]
	if !ok {
		return nil, &fakeAPIError{code: "NoSuchKey", msg: "missing"}
	}

	return &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader(object.body)),
		ETag: aws.String(object.etag),
	}, nil
}

func TestTranscriptObjectKeyIsStable(t *testing.T) {
	podcast := Result{
		Podcast: Podcast{Title: "Example Podcast"},
		Episode: Episode{Title: "Example Episode"},
	}

	first, err := transcriptObjectKey("transcribe", podcast)
	if err != nil {
		t.Fatalf("transcriptObjectKey returned error: %v", err)
	}
	second, err := transcriptObjectKey("transcribe", podcast)
	if err != nil {
		t.Fatalf("transcriptObjectKey returned error: %v", err)
	}

	if first != second {
		t.Fatalf("expected stable key, got %q and %q", first, second)
	}
	if !strings.HasPrefix(first, "transcripts/example-podcast--example-episode--") {
		t.Fatalf("expected readable transcript key prefix, got %q", first)
	}
	if !strings.HasSuffix(first, ".json") {
		t.Fatalf("expected .json suffix, got %q", first)
	}
}

func TestTranscriptReadableSlugFallsBackToSourceMetadata(t *testing.T) {
	got := transcriptReadableSlug(Result{
		Source: Source{
			ShareTitle: "[Better Offline] The Reality of AI Economics",
		},
	})

	if got != "better-offline-the-reality-of-ai-economics" {
		t.Fatalf("unexpected fallback slug %q", got)
	}
}

func TestPersistTranscriptWritesExpectedS3Object(t *testing.T) {
	originalClientFactory := newTranscriptS3ClientFunc
	defer func() {
		newTranscriptS3ClientFunc = originalClientFactory
	}()

	client := &fakeTranscriptS3Client{
		listOutputs: []*s3.ListObjectsV2Output{{
			Contents: []s3types.Object{
				{Key: aws.String("transcripts/example-podcast--example-episode.json")},
			},
		}},
		objects: map[string]fakeTranscriptObject{
			"transcripts/example-podcast--example-episode.json": {
				body: `{"podcast":{"podcast":{"title":"Example Podcast"},"episode":{"title":"Example Episode","published_date":"2026-04-06"}},"deepgram":{"metadata":{"created":"2026-04-08T18:30:10.753Z"}}}`,
				etag: "existing-etag",
			},
		},
	}
	newTranscriptS3ClientFunc = func(context.Context, string) (transcriptS3Client, error) {
		return client, nil
	}

	podcast := Result{
		Podcast: Podcast{Title: "Example Podcast"},
		Episode: Episode{Title: "Example Episode"},
	}
	body := []byte(`{"podcast":{"podcast":{"title":"Example Podcast"}},"deepgram":{"metadata":{"request_id":"dg-123"}}}`)

	if err := PersistTranscript(context.Background(), "transcribe", podcast, body); err != nil {
		t.Fatalf("PersistTranscript returned error: %v", err)
	}

	if len(client.putInputs) != 2 {
		t.Fatalf("expected 2 PutObject calls, got %d", len(client.putInputs))
	}
	if got := *client.putInputs[0].Bucket; got != "debugjois-dev-site" {
		t.Fatalf("expected bucket %q, got %q", "debugjois-dev-site", got)
	}
	if got := *client.putInputs[0].ContentType; got != "application/json" {
		t.Fatalf("expected content type %q, got %q", "application/json", got)
	}

	gotBody, err := io.ReadAll(client.putInputs[0].Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if !bytes.Equal(gotBody, body) {
		t.Fatalf("unexpected S3 body %s", string(gotBody))
	}
	if got := aws.ToString(client.putInputs[1].Key); got != transcripts.IndexObjectKey {
		t.Fatalf("expected index key %q, got %q", transcripts.IndexObjectKey, got)
	}
	if client.lastIndexPutIfNone != "*" {
		t.Fatalf("expected initial index write to use If-None-Match, got %q", client.lastIndexPutIfNone)
	}
}

func TestPersistTranscriptRetriesTranscriptIndexUpdateOnce(t *testing.T) {
	originalClientFactory := newTranscriptS3ClientFunc
	defer func() {
		newTranscriptS3ClientFunc = originalClientFactory
	}()

	client := &fakeTranscriptS3Client{
		failIndexWriteCount: 1,
		listOutputs: []*s3.ListObjectsV2Output{{
			Contents: []s3types.Object{{Key: aws.String("transcripts/example-podcast--example-episode.json")}},
		}, {
			Contents: []s3types.Object{{Key: aws.String("transcripts/example-podcast--example-episode.json")}},
		}},
		objects: map[string]fakeTranscriptObject{
			transcripts.IndexObjectKey: {
				body: `{"transcripts":[]}`,
				etag: `"etag-1"`,
			},
			"transcripts/example-podcast--example-episode.json": {
				body: `{"podcast":{"podcast":{"title":"Example Podcast"},"episode":{"title":"Example Episode","published_date":"2026-04-06"}},"deepgram":{"metadata":{"created":"2026-04-08T18:30:10.753Z"}}}`,
				etag: "existing-etag",
			},
		},
	}
	newTranscriptS3ClientFunc = func(context.Context, string) (transcriptS3Client, error) {
		return client, nil
	}

	podcast := Result{Podcast: Podcast{Title: "Example Podcast"}, Episode: Episode{Title: "Example Episode"}}
	body := []byte(`{"podcast":{"podcast":{"title":"Example Podcast"}},"deepgram":{"metadata":{"request_id":"dg-123"}}}`)

	if err := PersistTranscript(context.Background(), "transcribe", podcast, body); err != nil {
		t.Fatalf("PersistTranscript returned error: %v", err)
	}

	if len(client.putInputs) != 3 {
		t.Fatalf("expected 3 PutObject calls, got %d", len(client.putInputs))
	}
	if client.lastIndexPutIfMatch == "" {
		t.Fatal("expected retry write to use If-Match")
	}
}
