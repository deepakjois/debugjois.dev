package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/deepakjois/debugjois.dev/backend/api/internal/podcastaddict"
	"github.com/deepakjois/debugjois.dev/backend/api/internal/transcribe"
)

func TestRunPrintsJSONToStdout(t *testing.T) {
	originalTranscribe := transcribePodcastFunc
	originalPersist := persistTranscriptStoreFunc
	transcribePodcastFunc = func(_ context.Context, podcast podcastaddict.Result, _ transcribe.AudioTranscriber) (transcribe.Result, error) {
		return transcribe.Result{
			Podcast:  podcast,
			Deepgram: json.RawMessage(`{"metadata":{"request_id":"dg-1"}}`),
		}, nil
	}
	defer func() {
		transcribePodcastFunc = originalTranscribe
		persistTranscriptStoreFunc = originalPersist
	}()
	persistTranscriptStoreFunc = func(context.Context, string, podcastaddict.Result, []byte) error {
		t.Fatal("expected run without --write to skip persistence")
		return nil
	}

	var gotUserAgent string
	client := testHTTPClient(func(reqURL string, headers map[string]string) (int, string) {
		gotUserAgent = headers["User-Agent"]
		if reqURL != "https://podcastaddict.com/better-offline/episode/221030037" {
			t.Fatalf("unexpected request URL %q", reqURL)
		}
		return http.StatusOK, readFixture(t, "better-offline.html")
	})

	stdin := strings.NewReader("[Better Offline] The Reality of AI Economics With Paul Kedrosky\nhttps://podcastaddict.com/better-offline/episode/221030037 via @PodcastAddict\n")
	var stdout bytes.Buffer

	if err := run(context.Background(), nil, stdin, &stdout, client); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var got transcribe.Result
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	if got.Podcast.Source.ShareTitle != "[Better Offline] The Reality of AI Economics With Paul Kedrosky" {
		t.Fatalf("unexpected share title %q", got.Podcast.Source.ShareTitle)
	}
	if got.Podcast.Source.EpisodeURL != "https://podcastaddict.com/better-offline/episode/221030037" {
		t.Fatalf("unexpected source episode URL %q", got.Podcast.Source.EpisodeURL)
	}
	if got.Podcast.Episode.Title != "Better Offline - The Reality of AI Economics With Paul Kedrosky" {
		t.Fatalf("unexpected episode title %q", got.Podcast.Episode.Title)
	}
	var deepgramBody map[string]any
	if err := json.Unmarshal(got.Deepgram, &deepgramBody); err != nil {
		t.Fatalf("json.Unmarshal deepgram payload: %v", err)
	}
	metadata, ok := deepgramBody["metadata"].(map[string]any)
	if !ok || metadata["request_id"] != "dg-1" {
		t.Fatalf("unexpected deepgram payload %#v", deepgramBody)
	}
	if gotUserAgent != podcastaddict.UserAgent {
		t.Fatalf("expected user agent %q, got %q", podcastaddict.UserAgent, gotUserAgent)
	}
}

func TestRunRejectsExtraArguments(t *testing.T) {
	err := run(context.Background(), []string{"one", "two"}, strings.NewReader(""), io.Discard, newHTTPClient())
	if err == nil {
		t.Fatal("expected run to reject extra arguments")
	}
	if !strings.Contains(err.Error(), "at most one positional argument") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStoresTranscriptWhenFlagProvided(t *testing.T) {
	originalTranscribe := transcribePodcastFunc
	originalPersist := persistTranscriptStoreFunc
	defer func() {
		transcribePodcastFunc = originalTranscribe
		persistTranscriptStoreFunc = originalPersist
	}()

	transcribePodcastFunc = func(_ context.Context, podcast podcastaddict.Result, _ transcribe.AudioTranscriber) (transcribe.Result, error) {
		return transcribe.Result{
			Podcast:  podcast,
			Deepgram: json.RawMessage(`{"metadata":{"request_id":"dg-store"}}`),
		}, nil
	}

	var gotAction string
	var gotPodcast podcastaddict.Result
	var gotBody []byte
	persistTranscriptStoreFunc = func(_ context.Context, action string, podcast podcastaddict.Result, body []byte) error {
		gotAction = action
		gotPodcast = podcast
		gotBody = append([]byte(nil), body...)
		return nil
	}

	client := testHTTPClient(func(reqURL string, headers map[string]string) (int, string) {
		return http.StatusOK, readFixture(t, "better-offline.html")
	})

	stdin := strings.NewReader("[Better Offline] The Reality of AI Economics With Paul Kedrosky\nhttps://podcastaddict.com/better-offline/episode/221030037 via @PodcastAddict\n")
	var stdout bytes.Buffer

	err := run(context.Background(), []string{"--write"}, stdin, &stdout, client)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if gotAction != "transcribe" {
		t.Fatalf("unexpected action %q", gotAction)
	}
	if gotPodcast.Podcast.Title != "Better Offline" {
		t.Fatalf("unexpected podcast %#v", gotPodcast)
	}
	if string(gotBody) == "" {
		t.Fatal("expected stored transcript body")
	}
}
