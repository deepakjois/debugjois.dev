package transcripts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
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

type fakeClient struct {
	listOutputs         []*s3.ListObjectsV2Output
	listCalls           int
	objects             map[string]fakeObject
	conditionalFailures int
	indexPutCalls       int
	lastIfMatch         string
	lastIfNoneMatch     string
}

type fakeObject struct {
	body string
	etag string
}

func (f *fakeClient) ListObjectsV2(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if f.listCalls >= len(f.listOutputs) {
		return &s3.ListObjectsV2Output{}, nil
	}
	output := f.listOutputs[f.listCalls]
	f.listCalls++
	return output, nil
}

func (f *fakeClient) GetObject(_ context.Context, input *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	object, ok := f.objects[aws.ToString(input.Key)]
	if !ok {
		return nil, &fakeAPIError{code: "NoSuchKey", msg: "missing"}
	}
	return &s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader(object.body)),
		ETag: aws.String(object.etag),
	}, nil
}

func (f *fakeClient) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	body, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}

	key := aws.ToString(input.Key)
	if key == IndexObjectKey {
		f.indexPutCalls++
		f.lastIfMatch = aws.ToString(input.IfMatch)
		f.lastIfNoneMatch = aws.ToString(input.IfNoneMatch)
		if f.conditionalFailures > 0 {
			f.conditionalFailures--
			if current, ok := f.objects[IndexObjectKey]; ok {
				current.etag = fmt.Sprintf("etag-retry-%d", f.indexPutCalls)
				f.objects[IndexObjectKey] = current
			}
			return nil, &fakeAPIError{code: "ConditionalRequestConflict", msg: "conflict"}
		}
	}

	f.objects[key] = fakeObject{body: string(body), etag: fmt.Sprintf("etag-%d", f.indexPutCalls)}
	return &s3.PutObjectOutput{}, nil
}

func TestBuildIndexPrefersEpisodeTitleAndPublishedDate(t *testing.T) {
	client := &fakeClient{
		listOutputs: []*s3.ListObjectsV2Output{{
			Contents: []s3types.Object{{Key: aws.String("transcripts/example-episode.json")}, {Key: aws.String(IndexObjectKey)}},
		}},
		objects: map[string]fakeObject{
			"transcripts/example-episode.json": {
				body: mustMarshalDoc(t, map[string]any{
					"podcast": map[string]any{
						"source":  map[string]any{"share_title": "[Example Podcast] Shared title"},
						"podcast": map[string]any{"title": "Example Podcast"},
						"episode": map[string]any{"title": "Example Episode", "published_date": "2026-04-06"},
					},
					"deepgram": map[string]any{"metadata": map[string]any{"created": "2026-04-08T19:18:58.618Z"}},
				}),
			},
		},
	}

	index, err := BuildIndex(context.Background(), client, "debugjois-dev-site")
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	if len(index.Transcripts) != 1 {
		t.Fatalf("expected 1 transcript, got %d", len(index.Transcripts))
	}

	got := index.Transcripts[0]
	if got.Title != "Example Episode" {
		t.Fatalf("unexpected title %q", got.Title)
	}
	if got.Location != "https://www.debugjois.dev/transcripts/example-episode.json" {
		t.Fatalf("unexpected location %q", got.Location)
	}
	if got.Date != "2026-04-06" {
		t.Fatalf("unexpected date %q", got.Date)
	}
}

func TestBuildIndexFallsBackToDeepgramCreatedDate(t *testing.T) {
	client := &fakeClient{
		listOutputs: []*s3.ListObjectsV2Output{{
			Contents: []s3types.Object{{Key: aws.String("transcripts/sample.json")}},
		}},
		objects: map[string]fakeObject{
			"transcripts/sample.json": {
				body: mustMarshalDoc(t, map[string]any{
					"podcast": map[string]any{
						"podcast": map[string]any{"title": "SampleLib Demo Podcast"},
						"episode": map[string]any{"title": "3 Second Audio Sample"},
					},
					"deepgram": map[string]any{"metadata": map[string]any{"created": "2026-04-08T18:30:10.753Z"}},
				}),
			},
		},
	}

	index, err := BuildIndex(context.Background(), client, "debugjois-dev-site")
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	if got := index.Transcripts[0].Date; got != "2026-04-08" {
		t.Fatalf("unexpected fallback date %q", got)
	}
}

func TestBuildIndexSortsByDateStringDescending(t *testing.T) {
	client := &fakeClient{
		listOutputs: []*s3.ListObjectsV2Output{{
			Contents: []s3types.Object{
				{Key: aws.String("transcripts/older.json")},
				{Key: aws.String("transcripts/newer.json")},
				{Key: aws.String("transcripts/same-date.json")},
			},
		}},
		objects: map[string]fakeObject{
			"transcripts/older.json": {
				body: mustMarshalDoc(t, map[string]any{
					"podcast":  map[string]any{"episode": map[string]any{"title": "Older Episode", "published_date": "2026-04-06"}},
					"deepgram": map[string]any{"metadata": map[string]any{"created": "2026-04-08T18:30:10.753Z"}},
				}),
			},
			"transcripts/newer.json": {
				body: mustMarshalDoc(t, map[string]any{
					"podcast":  map[string]any{"episode": map[string]any{"title": "Newer Episode", "published_date": "2026-04-08"}},
					"deepgram": map[string]any{"metadata": map[string]any{"created": "2026-04-09T18:30:10.753Z"}},
				}),
			},
			"transcripts/same-date.json": {
				body: mustMarshalDoc(t, map[string]any{
					"podcast":  map[string]any{"episode": map[string]any{"title": "Alphabetical Tie", "published_date": "2026-04-08"}},
					"deepgram": map[string]any{"metadata": map[string]any{"created": "2026-04-09T19:30:10.753Z"}},
				}),
			},
		},
	}

	index, err := BuildIndex(context.Background(), client, BucketName)
	if err != nil {
		t.Fatalf("BuildIndex returned error: %v", err)
	}

	if len(index.Transcripts) != 3 {
		t.Fatalf("expected 3 transcripts, got %d", len(index.Transcripts))
	}

	gotTitles := []string{
		index.Transcripts[0].Title,
		index.Transcripts[1].Title,
		index.Transcripts[2].Title,
	}
	wantTitles := []string{"Alphabetical Tie", "Newer Episode", "Older Episode"}
	for i := range wantTitles {
		if gotTitles[i] != wantTitles[i] {
			t.Fatalf("unexpected title order %v", gotTitles)
		}
	}
}

func TestRefreshIndexUsesIfNoneMatchWhenIndexMissing(t *testing.T) {
	client := &fakeClient{
		listOutputs: []*s3.ListObjectsV2Output{{
			Contents: []s3types.Object{{Key: aws.String("transcripts/example.json")}},
		}},
		objects: map[string]fakeObject{
			"transcripts/example.json": {
				body: mustMarshalDoc(t, map[string]any{
					"podcast":  map[string]any{"episode": map[string]any{"title": "Example", "published_date": "2026-04-06"}},
					"deepgram": map[string]any{"metadata": map[string]any{"created": "2026-04-08T18:30:10.753Z"}},
				}),
			},
		},
	}

	if err := RefreshIndex(context.Background(), client, "debugjois-dev-site"); err != nil {
		t.Fatalf("RefreshIndex returned error: %v", err)
	}

	if client.lastIfNoneMatch != "*" {
		t.Fatalf("expected If-None-Match, got %q", client.lastIfNoneMatch)
	}
	if client.lastIfMatch != "" {
		t.Fatalf("expected no If-Match, got %q", client.lastIfMatch)
	}
}

func TestRefreshIndexRetriesConditionalConflictOnce(t *testing.T) {
	client := &fakeClient{
		listOutputs: []*s3.ListObjectsV2Output{{
			Contents: []s3types.Object{{Key: aws.String("transcripts/example.json")}},
		}, {
			Contents: []s3types.Object{{Key: aws.String("transcripts/example.json")}},
		}},
		objects: map[string]fakeObject{
			IndexObjectKey: {
				body: `{"transcripts":[]}`,
				etag: `"etag-1"`,
			},
			"transcripts/example.json": {
				body: mustMarshalDoc(t, map[string]any{
					"podcast":  map[string]any{"episode": map[string]any{"title": "Example", "published_date": "2026-04-06"}},
					"deepgram": map[string]any{"metadata": map[string]any{"created": "2026-04-08T18:30:10.753Z"}},
				}),
			},
		},
		conditionalFailures: 1,
	}

	if err := RefreshIndex(context.Background(), client, "debugjois-dev-site"); err != nil {
		t.Fatalf("RefreshIndex returned error: %v", err)
	}

	if client.indexPutCalls != 2 {
		t.Fatalf("expected 2 index writes, got %d", client.indexPutCalls)
	}
	if client.lastIfMatch != "etag-retry-1" {
		t.Fatalf("expected retry to use refreshed ETag, got %q", client.lastIfMatch)
	}
}

func TestRefreshIndexFailsAfterSecondConditionalConflict(t *testing.T) {
	client := &fakeClient{
		listOutputs: []*s3.ListObjectsV2Output{{
			Contents: []s3types.Object{{Key: aws.String("transcripts/example.json")}},
		}, {
			Contents: []s3types.Object{{Key: aws.String("transcripts/example.json")}},
		}},
		objects: map[string]fakeObject{
			IndexObjectKey: {
				body: `{"transcripts":[]}`,
				etag: `"etag-1"`,
			},
			"transcripts/example.json": {
				body: mustMarshalDoc(t, map[string]any{
					"podcast":  map[string]any{"episode": map[string]any{"title": "Example", "published_date": "2026-04-06"}},
					"deepgram": map[string]any{"metadata": map[string]any{"created": "2026-04-08T18:30:10.753Z"}},
				}),
			},
		},
		conditionalFailures: 2,
	}

	err := RefreshIndex(context.Background(), client, "debugjois-dev-site")
	if err == nil {
		t.Fatal("expected RefreshIndex to fail")
	}
	if !strings.Contains(err.Error(), "after retry") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustMarshalDoc(t *testing.T, doc map[string]any) string {
	t.Helper()
	body, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	return string(body)
}
