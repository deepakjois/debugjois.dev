package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppRootGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	NewAppHandler().ServeHTTP(res, req)

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

func TestAppRootHead(t *testing.T) {
	req := httptest.NewRequest(http.MethodHead, "/", nil)
	res := httptest.NewRecorder()

	NewAppHandler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	if res.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", res.Body.String())
	}
}

func TestAppHealthGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	NewAppHandler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status %q, got %#v", "ok", body["status"])
	}

	if _, ok := body["email"]; ok {
		t.Fatalf("expected email field to be omitted, got %#v", body["email"])
	}
}

func TestAppHealthHead(t *testing.T) {
	req := httptest.NewRequest(http.MethodHead, "/health", nil)
	res := httptest.NewRecorder()

	NewAppHandler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	if res.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", res.Body.String())
	}
}

func TestAppNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	res := httptest.NewRecorder()

	NewAppHandler().ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, res.Code)
	}

	var body errorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Error != "not found" {
		t.Fatalf("expected error %q, got %q", "not found", body.Error)
	}
}

func TestAppMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	res := httptest.NewRecorder()

	NewAppHandler().ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}

	var body errorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Error != "method not allowed" {
		t.Fatalf("expected error %q, got %q", "method not allowed", body.Error)
	}
}

func TestAppOptionsReturnsCORSHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	res := httptest.NewRecorder()

	NewAppHandler().ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, res.Code)
	}

	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected allow origin %q, got %q", "*", got)
	}
}
