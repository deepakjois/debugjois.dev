package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/deepakjois/debugjois.dev/backend/api/internal/podcastaddict"
	"github.com/deepakjois/debugjois.dev/backend/api/internal/transcribe"
)

func TestPodcastTranscribePost(t *testing.T) {
	h := newHTTPHandler(&app{
		parsePodcastTranscribe: func(_ context.Context, text string) (podcastaddict.Result, error) {
			if text != "hello world" {
				t.Fatalf("expected text %q, got %q", "hello world", text)
			}

			return podcastaddict.Result{
				Source: podcastaddict.Source{
					Input:      "hello world",
					EpisodeURL: "https://podcastaddict.com/example/episode/123",
				},
				Podcast: podcastaddict.Podcast{
					Title: "Example Podcast",
					URL:   "https://podcastaddict.com/podcast/example/999",
				},
				Episode: podcastaddict.Episode{
					Title:           "Example Episode",
					PublishedAt:     "2026-04-07T05:00:00-07:00",
					PublishedDate:   "2026-04-07",
					Duration:        "52 mins",
					AudioURL:        "https://cdn.example.com/audio.mp3",
					DescriptionHTML: "<p>hello</p>",
				},
			}, nil
		},
		dispatchPodcastTranscribe: func(_ context.Context, podcast podcastaddict.Result) (string, error) {
			if podcast.Podcast.Title != "Example Podcast" {
				t.Fatalf("expected podcast metadata to be forwarded, got %#v", podcast)
			}
			return "local-123", nil
		},
	})

	res := serveForm(t, h, "/podcast-transcribe", url.Values{"text": {"hello world"}})

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	body := decodeJSON[podcastTranscribeAcceptedResponse](t, res.Body.Bytes())
	if body.Podcast.Podcast.Title != "Example Podcast" {
		t.Fatalf("expected podcast title %q, got %q", "Example Podcast", body.Podcast.Podcast.Title)
	}
	if body.Podcast.Episode.DescriptionHTML != "<p>hello</p>" {
		t.Fatalf("expected description HTML %q, got %q", "<p>hello</p>", body.Podcast.Episode.DescriptionHTML)
	}
	if body.TranscriptionLambdaID != "local-123" {
		t.Fatalf("expected transcription id %q, got %q", "local-123", body.TranscriptionLambdaID)
	}
}

func TestPodcastTranscribePostMissingText(t *testing.T) {
	h := newHTTPHandler(&app{})

	res := serveForm(t, h, "/podcast-transcribe", url.Values{})

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}

	body := decodeJSON[errorResponse](t, res.Body.Bytes())
	if body.Error != "text parameter is required" {
		t.Fatalf("expected error %q, got %q", "text parameter is required", body.Error)
	}
}

func TestPodcastTranscribePostInvalidPayload(t *testing.T) {
	h := newHTTPHandler(&app{
		parsePodcastTranscribe: func(_ context.Context, _ string) (podcastaddict.Result, error) {
			return podcastaddict.Result{}, &podcastaddict.Error{
				Kind: podcastaddict.ErrorKindInvalidInput,
				Err:  errors.New("expected Podcast Addict episode URL"),
			}
		},
	})

	res := serveForm(t, h, "/podcast-transcribe", url.Values{"text": {"not a valid payload"}})

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}

	body := decodeJSON[errorResponse](t, res.Body.Bytes())
	if body.Error != "expected Podcast Addict episode URL" {
		t.Fatalf("expected error %q, got %q", "expected Podcast Addict episode URL", body.Error)
	}
}

func TestPodcastTranscribePostUpstreamError(t *testing.T) {
	h := newHTTPHandler(&app{
		parsePodcastTranscribe: func(_ context.Context, _ string) (podcastaddict.Result, error) {
			return podcastaddict.Result{}, &podcastaddict.Error{
				Kind: podcastaddict.ErrorKindUpstream,
				Err:  errors.New("fetch episode page: unexpected status 403"),
			}
		},
	})

	res := serveForm(t, h, "/podcast-transcribe", url.Values{"text": {"https://podcastaddict.com/example/episode/123"}})

	if res.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, res.Code)
	}

	body := decodeJSON[errorResponse](t, res.Body.Bytes())
	if body.Error != "fetch episode page: unexpected status 403" {
		t.Fatalf("expected error %q, got %q", "fetch episode page: unexpected status 403", body.Error)
	}
}

func TestPodcastTranscribePostDispatchError(t *testing.T) {
	h := newHTTPHandler(&app{
		parsePodcastTranscribe: func(_ context.Context, _ string) (podcastaddict.Result, error) {
			return podcastaddict.Result{
				Episode: podcastaddict.Episode{AudioURL: "https://cdn.example.com/audio.mp3"},
			}, nil
		},
		dispatchPodcastTranscribe: func(_ context.Context, _ podcastaddict.Result) (string, error) {
			return "", errors.New("invoke Lambda for transcription: boom")
		},
	})

	res := serveForm(t, h, "/podcast-transcribe", url.Values{"text": {"hello world"}})

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, res.Code)
	}

	body := decodeJSON[errorResponse](t, res.Body.Bytes())
	if body.Error != "invoke Lambda for transcription: boom" {
		t.Fatalf("expected error %q, got %q", "invoke Lambda for transcription: boom", body.Error)
	}
}

func TestPodcastTranscribeDispatcherUsesLambdaInvokeWhenRuntimeSet(t *testing.T) {
	t.Setenv("AWS_LAMBDA_RUNTIME_API", "127.0.0.1")

	original := invokeSelfForPodcastTranscribeFunc
	invokeSelfForPodcastTranscribeFunc = func(_ context.Context, podcast podcastaddict.Result) (string, error) {
		if podcast.Podcast.Title != "Example Podcast" {
			t.Fatalf("expected podcast metadata to be forwarded, got %#v", podcast)
		}
		return "invoke-request-id", nil
	}
	defer func() {
		invokeSelfForPodcastTranscribeFunc = original
	}()

	dispatch := (&app{}).podcastTranscribeDispatcher()
	gotID, err := dispatch(context.Background(), podcastaddict.Result{
		Podcast: podcastaddict.Podcast{Title: "Example Podcast"},
	})
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}
	if gotID != "invoke-request-id" {
		t.Fatalf("expected id %q, got %q", "invoke-request-id", gotID)
	}
}

func TestPodcastTranscribeDispatcherRunsLocalTranscription(t *testing.T) {
	t.Setenv("AWS_LAMBDA_RUNTIME_API", "")

	originalRunLocal := runLocalPodcastTranscriptionFunc
	originalLocalID := newLocalTranscriptionIDFunc
	defer func() {
		runLocalPodcastTranscriptionFunc = originalRunLocal
		newLocalTranscriptionIDFunc = originalLocalID
	}()

	done := make(chan podcastaddict.Result, 1)
	runLocalPodcastTranscriptionFunc = func(_ context.Context, podcast podcastaddict.Result) (transcribe.Result, error) {
		done <- podcast
		return transcribe.Result{}, nil
	}
	newLocalTranscriptionIDFunc = func() string { return "local-test-id" }

	dispatch := (&app{}).podcastTranscribeDispatcher()
	gotID, err := dispatch(context.Background(), podcastaddict.Result{
		Podcast: podcastaddict.Podcast{Title: "Example Podcast"},
	})
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}
	if gotID != "local-test-id" {
		t.Fatalf("expected id %q, got %q", "local-test-id", gotID)
	}

	gotPodcast := <-done
	if gotPodcast.Podcast.Title != "Example Podcast" {
		t.Fatalf("expected local transcription to receive podcast metadata, got %#v", gotPodcast)
	}
}

func serveForm(t *testing.T, h http.Handler, path string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(values.Encode())))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}
