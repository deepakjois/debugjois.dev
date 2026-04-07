package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/config"
	awslambdasdk "github.com/aws/aws-sdk-go-v2/service/lambda"
	awslambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/deepakjois/debugjois.dev/backend/api/internal/podcastaddict"
	"github.com/deepakjois/debugjois.dev/backend/api/internal/transcribe"
)

type podcastTranscribeAcceptedResponse struct {
	Podcast               podcastaddict.Result `json:"podcast"`
	TranscriptionLambdaID string               `json:"transcription_lambda_id"`
}

var (
	invokeSelfForPodcastTranscribeFunc = invokeSelfForPodcastTranscribe
	runLocalPodcastTranscriptionFunc   = func(ctx context.Context, podcast podcastaddict.Result) (transcribe.Result, error) {
		return transcribe.TranscribePodcast(ctx, podcast, nil)
	}
	newLocalTranscriptionIDFunc = func() string {
		return fmt.Sprintf("local-%d", time.Now().UnixNano())
	}
)

func (a *app) handlePodcastTranscribePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeHTTPResponse(w, http.StatusBadRequest, errorResponse{Error: "invalid form body"})
		return
	}

	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		writeHTTPResponse(w, http.StatusBadRequest, errorResponse{Error: "text parameter is required"})
		return
	}

	result, err := a.podcastTranscribeParser()(r.Context(), text)
	if err != nil {
		writeHTTPResponse(w, podcastaddict.HTTPStatus(err), errorResponse{Error: err.Error()})
		return
	}

	transcriptionID, err := a.podcastTranscribeDispatcher()(r.Context(), result)
	if err != nil {
		writeHTTPResponse(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	writeHTTPResponse(w, http.StatusOK, podcastTranscribeAcceptedResponse{
		Podcast:               result,
		TranscriptionLambdaID: transcriptionID,
	})
}

func (a *app) podcastTranscribeParser() func(context.Context, string) (podcastaddict.Result, error) {
	if a != nil && a.parsePodcastTranscribe != nil {
		return a.parsePodcastTranscribe
	}

	return func(ctx context.Context, text string) (podcastaddict.Result, error) {
		return podcastaddict.ParseEpisode(ctx, podcastaddict.NewHTTPClient(), text)
	}
}

func (a *app) podcastTranscribeDispatcher() func(context.Context, podcastaddict.Result) (string, error) {
	if a != nil && a.dispatchPodcastTranscribe != nil {
		return a.dispatchPodcastTranscribe
	}

	if isLambdaRuntime() {
		return func(ctx context.Context, podcast podcastaddict.Result) (string, error) {
			return invokeSelfForPodcastTranscribeFunc(ctx, podcast)
		}
	}

	return func(_ context.Context, podcast podcastaddict.Result) (string, error) {
		id := newLocalTranscriptionIDFunc()
		go func(podcast podcastaddict.Result) {
			if _, err := runLocalPodcastTranscriptionFunc(context.Background(), podcast); err != nil {
				log.Printf("local podcast transcribe failed: %v", err)
			}
		}(podcast)
		return id, nil
	}
}

func invokeSelfForPodcastTranscribe(ctx context.Context, podcast podcastaddict.Result) (string, error) {
	functionName := strings.TrimSpace(os.Getenv("AWS_LAMBDA_FUNCTION_NAME"))
	if functionName == "" {
		return "", fmt.Errorf("AWS_LAMBDA_FUNCTION_NAME must be set in Lambda")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("load AWS config for Lambda invoke: %w", err)
	}

	payload, err := json.Marshal(transcribe.DirectRequest{
		Action:  "transcribe",
		Podcast: podcast,
	})
	if err != nil {
		return "", fmt.Errorf("marshal transcription payload: %w", err)
	}

	client := awslambdasdk.NewFromConfig(cfg)
	output, err := client.Invoke(ctx, &awslambdasdk.InvokeInput{
		FunctionName:   &functionName,
		InvocationType: awslambdatypes.InvocationTypeEvent,
		Payload:        payload,
	})
	if err != nil {
		return "", fmt.Errorf("invoke Lambda for transcription: %w", err)
	}

	requestID, ok := awsmiddleware.GetRequestIDMetadata(output.ResultMetadata)
	if !ok || strings.TrimSpace(requestID) == "" {
		return "", fmt.Errorf("invoke Lambda for transcription: missing request ID")
	}

	return requestID, nil
}
