package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
}

func TestAppNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	res := httptest.NewRecorder()

	NewAppHandler().ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, res.Code)
	}

	if !strings.Contains(res.Body.String(), "404 page not found") {
		t.Fatalf("expected default not found body, got %q", res.Body.String())
	}
}

func TestAppMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	res := httptest.NewRecorder()

	NewAppHandler().ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}

	allow := res.Header().Get("Allow")
	if !strings.Contains(allow, http.MethodGet) || !strings.Contains(allow, http.MethodHead) {
		t.Fatalf("expected Allow header to mention GET and HEAD, got %q", allow)
	}
}
