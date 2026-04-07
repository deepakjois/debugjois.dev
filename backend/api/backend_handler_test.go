package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"testing"
)

type backendHandlerFunc func(context.Context, json.RawMessage) (json.RawMessage, error)

func (f backendHandlerFunc) HandleLambdaEvent(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	return f(ctx, payload)
}

func TestRunInvokeReadsPayloadFromStdin(t *testing.T) {
	var gotPayload json.RawMessage

	handler := backendHandlerFunc(func(_ context.Context, payload json.RawMessage) (json.RawMessage, error) {
		gotPayload = append(json.RawMessage(nil), payload...)
		return json.RawMessage(`{"ok":true}`), nil
	})

	var stdout bytes.Buffer
	err := runInvoke(
		context.Background(),
		nil,
		handler,
		strings.NewReader("\n{\"action\":\"health-check\"}\n"),
		&stdout,
	)
	if err != nil {
		t.Fatalf("run invoke: %v", err)
	}

	if got := string(gotPayload); got != `{"action":"health-check"}` {
		t.Fatalf("expected stdin payload to be forwarded, got %q", got)
	}

	if got := stdout.String(); got != "{\"ok\":true}\n" {
		t.Fatalf("expected invoke output, got %q", got)
	}
}

func TestLocalBackendHandlerIgnoresAPIGatewayEvent(t *testing.T) {
	handler := newLocalBackendHandler()
	var logOutput bytes.Buffer

	originalWriter := log.Writer()
	log.SetOutput(&logOutput)
	defer log.SetOutput(originalWriter)

	result, err := handler.HandleLambdaEvent(context.Background(), json.RawMessage(`{
		"version": "2.0",
		"rawPath": "/",
		"requestContext": {
			"http": {
				"method": "GET",
				"path": "/"
			}
		}
	}`))
	if err != nil {
		t.Fatalf("expected API Gateway invoke to be ignored, got %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for ignored API Gateway event, got %q", string(result))
	}
	if !strings.Contains(logOutput.String(), "`serve`") {
		t.Fatalf("expected log to mention serve command, got %q", logOutput.String())
	}
}

func TestRunInvokeSkipsOutputWhenHandlerReturnsNilPayload(t *testing.T) {
	handler := backendHandlerFunc(func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
		return nil, nil
	})

	var stdout bytes.Buffer
	err := runInvoke(
		context.Background(),
		nil,
		handler,
		strings.NewReader(`{"action":"health-check"}`),
		&stdout,
	)
	if err != nil {
		t.Fatalf("run invoke: %v", err)
	}

	if got := stdout.String(); got != "" {
		t.Fatalf("expected no output for nil payload, got %q", got)
	}
}

func TestLambdaBackendHandlerHandlesScheduledEvent(t *testing.T) {
	handler := newLambdaBackendHandler(NewAppHandler(nil, nil, nil, nil, "", ""))

	result, err := handler.HandleLambdaEvent(context.Background(), json.RawMessage(`{
		"version": "0",
		"id": "12345678-1234-1234-1234-123456789abc",
		"detail-type": "Scheduled Event",
		"source": "aws.events",
		"detail": {}
	}`))
	if err != nil {
		t.Fatalf("handle scheduled event: %v", err)
	}

	if got := string(result); got != `{"ok":true}` {
		t.Fatalf("expected ok response, got %q", got)
	}
}
