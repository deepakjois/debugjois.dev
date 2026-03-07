package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoot(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	handleLocalRequest(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var body rootResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Message != helloMessage {
		t.Fatalf("expected message %q, got %q", helloMessage, body.Message)
	}
}

func TestHealthNoJWTContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	handleLocalRequest(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var body healthResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected status %q, got %q", "ok", body.Status)
	}

	if body.Email != nil {
		t.Fatalf("expected nil email, got %q", *body.Email)
	}
}

func TestGetEmailWithoutEvent(t *testing.T) {
	if email := getEmailFromRequest(nil); email != nil {
		t.Fatalf("expected nil email, got %q", *email)
	}
}

func TestGetEmailWithJWTClaims(t *testing.T) {
	event := lambdaEvent{}
	event.RequestContext.Authorizer.JWT.Claims = map[string]string{"email": "test@example.com"}

	email := getEmailFromRequest(&event)
	if email == nil {
		t.Fatal("expected email, got nil")
	}

	if *email != "test@example.com" {
		t.Fatalf("expected email %q, got %q", "test@example.com", *email)
	}
}

func TestLambdaHealthIncludesJWTEmail(t *testing.T) {
	event := lambdaEvent{RawPath: "/health"}
	event.RequestContext.HTTP.Method = http.MethodGet
	event.RequestContext.Authorizer.JWT.Claims = map[string]string{"email": "test@example.com"}

	response, err := handleLambdaInvocation(event)
	if err != nil {
		t.Fatalf("handle lambda invocation: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}

	var body healthResponse
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Email == nil || *body.Email != "test@example.com" {
		t.Fatalf("expected lambda email %q, got %#v", "test@example.com", body.Email)
	}
}
