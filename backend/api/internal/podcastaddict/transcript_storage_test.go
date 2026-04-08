package podcastaddict

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type fakeTranscriptS3Client struct {
	input *s3.PutObjectInput
}

func (f *fakeTranscriptS3Client) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.input = input
	return &s3.PutObjectOutput{}, nil
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

	client := &fakeTranscriptS3Client{}
	newTranscriptS3ClientFunc = func(context.Context, string) (transcriptS3Client, error) {
		return client, nil
	}

	podcast := Result{
		Podcast: Podcast{Title: "Example Podcast"},
		Episode: Episode{Title: "Example Episode"},
	}
	body := []byte(`{"podcast":{"podcast":{"title":"Example Podcast"}},"deepgram":{"metadata":{"request_id":"dg-123"}}}`)

	if err := PersistTranscript(context.Background(), "arn:aws:s3:::debugjois-dev-site", "transcribe", podcast, body); err != nil {
		t.Fatalf("PersistTranscript returned error: %v", err)
	}

	if client.input == nil {
		t.Fatal("expected PutObject to be called")
	}
	if got := *client.input.Bucket; got != "debugjois-dev-site" {
		t.Fatalf("expected bucket %q, got %q", "debugjois-dev-site", got)
	}
	if got := *client.input.ContentType; got != "application/json" {
		t.Fatalf("expected content type %q, got %q", "application/json", got)
	}

	gotBody, err := io.ReadAll(client.input.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if !bytes.Equal(gotBody, body) {
		t.Fatalf("unexpected S3 body %s", string(gotBody))
	}
}
