package transcribe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/deepakjois/debugjois.dev/backend/api/internal/podcastaddict"
)

type fakeAudioTranscriber struct {
	gotURL string
	body   json.RawMessage
	err    error
}

func (f *fakeAudioTranscriber) TranscribeURL(_ context.Context, audioURL string) (json.RawMessage, error) {
	f.gotURL = audioURL
	if f.err != nil {
		return nil, f.err
	}
	return f.body, nil
}

func TestTranscribePodcast(t *testing.T) {
	fake := &fakeAudioTranscriber{
		body: json.RawMessage(`{"metadata":{"request_id":"dg-1"},"results":{"channels":[]}}`),
	}

	result, err := TranscribePodcast(context.Background(), podcastaddict.Result{
		Podcast: podcastaddict.Podcast{Title: "Example Podcast"},
		Episode: podcastaddict.Episode{
			Title:    "Example Episode",
			AudioURL: "https://cdn.example.com/audio.mp3",
		},
	}, fake)
	if err != nil {
		t.Fatalf("TranscribePodcast returned error: %v", err)
	}

	if fake.gotURL != "https://cdn.example.com/audio.mp3" {
		t.Fatalf("expected audio URL %q, got %q", "https://cdn.example.com/audio.mp3", fake.gotURL)
	}
	if result.Podcast.Episode.Title != "Example Episode" {
		t.Fatalf("expected wrapped podcast metadata, got %#v", result.Podcast)
	}
	if string(result.Deepgram) != `{"metadata":{"request_id":"dg-1"},"results":{"channels":[]}}` {
		t.Fatalf("unexpected Deepgram payload %s", string(result.Deepgram))
	}
}

func TestTranscribePodcastRequiresAudioURL(t *testing.T) {
	_, err := TranscribePodcast(context.Background(), podcastaddict.Result{}, &fakeAudioTranscriber{})
	if err == nil {
		t.Fatal("expected missing audio URL error")
	}
	if HTTPStatus(err) != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400, got %d", HTTPStatus(err))
	}
}

func TestTranscribePodcastPropagatesUpstreamError(t *testing.T) {
	fake := &fakeAudioTranscriber{
		err: &Error{
			Kind: ErrorKindUpstream,
			Err:  errors.New("deepgram transcription failed"),
		},
	}

	_, err := TranscribePodcast(context.Background(), podcastaddict.Result{
		Episode: podcastaddict.Episode{AudioURL: "https://cdn.example.com/audio.mp3"},
	}, fake)
	if err == nil {
		t.Fatal("expected upstream error")
	}
	if HTTPStatus(err) != http.StatusBadGateway {
		t.Fatalf("expected HTTP 502, got %d", HTTPStatus(err))
	}
}

func TestNewPreRecordedOptions(t *testing.T) {
	options := newPreRecordedOptions()

	if options.Model != "nova-3" {
		t.Fatalf("expected model %q, got %q", "nova-3", options.Model)
	}
	if options.Language != "en" {
		t.Fatalf("expected language %q, got %q", "en", options.Language)
	}
	if !options.Diarize || !options.Paragraphs || !options.Punctuate || !options.Numerals || !options.SmartFormat || !options.Utterances {
		t.Fatalf("expected Deepgram options to enable diarize/paragraphs/punctuate/numerals/smart_format/utterances, got %#v", options)
	}
}
