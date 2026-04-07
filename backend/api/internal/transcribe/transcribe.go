package transcribe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	deepgramapi "github.com/deepgram/deepgram-go-sdk/v3/pkg/api/listen/v1/rest"
	deepgraminterfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	deepgramclient "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/listen"

	"github.com/deepakjois/debugjois.dev/backend/api/internal/podcastaddict"
)

const (
	DeepgramAPIKeyEnvVar = "DEEPGRAM_API_KEY"
	deepgramModel        = "nova-3"
)

var deepgramInitOnce sync.Once

type ErrorKind int

const (
	ErrorKindInvalidInput ErrorKind = iota + 1
	ErrorKindUpstream
	ErrorKindInternal
)

type Error struct {
	Kind ErrorKind
	Err  error
}

func (e *Error) Error() string {
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

type Result struct {
	Podcast  podcastaddict.Result `json:"podcast"`
	Deepgram json.RawMessage      `json:"deepgram"`
}

type DirectRequest struct {
	Action  string               `json:"action"`
	Podcast podcastaddict.Result `json:"podcast"`
}

type AudioTranscriber interface {
	TranscribeURL(ctx context.Context, audioURL string) (json.RawMessage, error)
}

type DeepgramClient struct{}

func HTTPStatus(err error) int {
	var target *Error
	if errors.As(err, &target) {
		switch target.Kind {
		case ErrorKindInvalidInput:
			return 400
		case ErrorKindUpstream:
			return 502
		default:
			return 500
		}
	}

	return 500
}

func NewDeepgramClientFromEnv() (*DeepgramClient, error) {
	if strings.TrimSpace(os.Getenv(DeepgramAPIKeyEnvVar)) == "" {
		return nil, invalidInputError("%s must be set", DeepgramAPIKeyEnvVar)
	}

	deepgramInitOnce.Do(func() {
		deepgramclient.InitWithDefault()
	})

	return &DeepgramClient{}, nil
}

func TranscribePodcast(ctx context.Context, podcast podcastaddict.Result, client AudioTranscriber) (Result, error) {
	if strings.TrimSpace(podcast.Episode.AudioURL) == "" {
		return Result{}, invalidInputError("podcast episode audio URL is missing")
	}

	if client == nil {
		defaultClient, err := NewDeepgramClientFromEnv()
		if err != nil {
			return Result{}, err
		}
		client = defaultClient
	}

	deepgramResponse, err := client.TranscribeURL(ctx, podcast.Episode.AudioURL)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Podcast:  podcast,
		Deepgram: deepgramResponse,
	}, nil
}

func (c *DeepgramClient) TranscribeURL(ctx context.Context, audioURL string) (json.RawMessage, error) {
	restClient := deepgramclient.NewRESTWithDefaults()
	deepgram := deepgramapi.New(restClient)

	response, err := deepgram.FromURL(ctx, audioURL, newPreRecordedOptions())
	if err != nil {
		var statusErr *deepgraminterfaces.StatusError
		if errors.As(err, &statusErr) {
			return nil, upstreamError("deepgram transcription failed: %s", statusErr.DeepgramError.ErrMsg)
		}
		return nil, upstreamError("deepgram transcription failed: %w", err)
	}

	body, err := json.Marshal(response)
	if err != nil {
		return nil, internalError("marshal Deepgram response: %w", err)
	}

	return json.RawMessage(body), nil
}

func newPreRecordedOptions() *deepgraminterfaces.PreRecordedTranscriptionOptions {
	return &deepgraminterfaces.PreRecordedTranscriptionOptions{
		Model:       deepgramModel,
		Language:    "en",
		Diarize:     true,
		Paragraphs:  true,
		Punctuate:   true,
		Numerals:    true,
		SmartFormat: true,
		Utterances:  true,
	}
}

func invalidInputError(format string, args ...any) error {
	return &Error{
		Kind: ErrorKindInvalidInput,
		Err:  fmt.Errorf(format, args...),
	}
}

func upstreamError(format string, args ...any) error {
	return &Error{
		Kind: ErrorKindUpstream,
		Err:  fmt.Errorf(format, args...),
	}
}

func internalError(format string, args ...any) error {
	return &Error{
		Kind: ErrorKindInternal,
		Err:  fmt.Errorf(format, args...),
	}
}
