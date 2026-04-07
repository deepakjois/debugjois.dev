package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/deepakjois/debugjois.dev/backend/api/internal/podcastaddict"
	"github.com/deepakjois/debugjois.dev/backend/api/internal/transcribe"
)

func TestHandleDirectLambdaEventTranscribe(t *testing.T) {
	original := transcribePodcastFunc
	transcribePodcastFunc = func(_ context.Context, request transcribe.DirectRequest) (transcribe.Result, error) {
		if request.Action != "transcribe" {
			t.Fatalf("expected action %q, got %q", "transcribe", request.Action)
		}
		if request.Podcast.Podcast.Title != "Example Podcast" {
			t.Fatalf("expected podcast metadata to be forwarded, got %#v", request.Podcast)
		}

		return transcribe.Result{
			Podcast:  request.Podcast,
			Deepgram: json.RawMessage(`{"metadata":{"request_id":"dg-123"}}`),
		}, nil
	}
	defer func() {
		transcribePodcastFunc = original
	}()

	payload := json.RawMessage(`{
		"action":"transcribe",
		"podcast":{
			"source":{"input":"hello","episode_url":"https://podcastaddict.com/example/episode/123"},
			"podcast":{"title":"Example Podcast"},
			"episode":{"title":"Example Episode","audio_url":"https://cdn.example.com/audio.mp3","description_html":"<p>hello</p>"}
		}
	}`)

	result, err := handleDirectLambdaEvent(context.Background(), payload)
	if err != nil {
		t.Fatalf("handleDirectLambdaEvent returned error: %v", err)
	}

	var got transcribe.Result
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	if got.Podcast.Podcast.Title != "Example Podcast" {
		t.Fatalf("expected podcast title %q, got %q", "Example Podcast", got.Podcast.Podcast.Title)
	}
	if string(got.Deepgram) != `{"metadata":{"request_id":"dg-123"}}` {
		t.Fatalf("unexpected Deepgram payload %s", string(got.Deepgram))
	}
}

func TestHandleDirectLambdaEventHealthCheck(t *testing.T) {
	result, err := handleDirectLambdaEvent(context.Background(), json.RawMessage(`{"action":"health-check"}`))
	if err != nil {
		t.Fatalf("handleDirectLambdaEvent returned error: %v", err)
	}

	if string(result) != `{"ok":true}` {
		t.Fatalf("expected ok response, got %q", string(result))
	}
}

func TestHandleDirectLambdaEventUnknownAction(t *testing.T) {
	_, err := handleDirectLambdaEvent(context.Background(), json.RawMessage(`{"action":"unknown"}`))
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
	if transcribe.HTTPStatus(err) != 400 {
		t.Fatalf("expected HTTP 400, got %d", transcribe.HTTPStatus(err))
	}
}

func TestDirectRequestJSONShape(t *testing.T) {
	payload, err := json.Marshal(transcribe.DirectRequest{
		Action: "transcribe",
		Podcast: podcastaddict.Result{
			Podcast: podcastaddict.Podcast{Title: "Example Podcast"},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	if string(payload) == "" {
		t.Fatal("expected marshaled payload")
	}
}
