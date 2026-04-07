package main

import (
	"encoding/json"
	"testing"
)

func TestClassifyAPIGatewayV2Event(t *testing.T) {
	payload := json.RawMessage(`{
		"version": "2.0",
		"routeKey": "$default",
		"rawPath": "/daily",
		"requestContext": {
			"http": {
				"method": "GET",
				"path": "/daily"
			}
		}
	}`)

	if got := classifyEvent(payload); got != eventTypeAPIGateway {
		t.Fatalf("expected eventTypeAPIGateway, got %d", got)
	}
}

func TestClassifyScheduledEvent(t *testing.T) {
	payload := json.RawMessage(`{
		"version": "0",
		"id": "12345678-1234-1234-1234-123456789abc",
		"detail-type": "Scheduled Event",
		"source": "aws.events",
		"account": "123456789012",
		"time": "2026-04-07T14:00:00Z",
		"region": "us-east-1",
		"resources": ["arn:aws:events:us-east-1:123456789012:rule/my-rule"],
		"detail": {}
	}`)

	if got := classifyEvent(payload); got != eventTypeScheduled {
		t.Fatalf("expected eventTypeScheduled, got %d", got)
	}
}

func TestClassifyDirectInvocation(t *testing.T) {
	payload := json.RawMessage(`{"action": "health-check"}`)

	if got := classifyEvent(payload); got != eventTypeDirect {
		t.Fatalf("expected eventTypeDirect, got %d", got)
	}
}

func TestClassifyEmptyPayload(t *testing.T) {
	payload := json.RawMessage(`{}`)

	if got := classifyEvent(payload); got != eventTypeDirect {
		t.Fatalf("expected eventTypeDirect for empty object, got %d", got)
	}
}

func TestClassifyMalformedJSON(t *testing.T) {
	payload := json.RawMessage(`not json`)

	if got := classifyEvent(payload); got != eventTypeDirect {
		t.Fatalf("expected eventTypeDirect for malformed JSON, got %d", got)
	}
}
