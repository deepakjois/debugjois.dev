package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLinkPreviewMissingQ(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("upstream should not be called when q is missing")
	}))
	defer upstream.Close()

	h := NewAppHandler(
		func(context.Context, string) (string, error) { return "", nil },
		func(context.Context, string, string, string) error { return nil },
		func() string { return "" },
		func() string { return "" },
		"test-api-key",
		upstream.URL,
	)

	res := serve(t, h, http.MethodGet, "/linkpreview")

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}

	body := decodeJSON[errorResponse](t, res.Body.Bytes())
	if body.Error != "q parameter is required" {
		t.Fatalf("expected error %q, got %q", "q parameter is required", body.Error)
	}
}

func TestLinkPreviewSuccess(t *testing.T) {
	const responseBody = `{"title":"Example Title","description":"Example description"}`
	var gotAPIKey string
	var gotQuery string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("X-Linkpreview-Api-Key")
		gotQuery = r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, responseBody)
	}))
	defer upstream.Close()

	h := NewAppHandler(
		func(context.Context, string) (string, error) { return "", nil },
		func(context.Context, string, string, string) error { return nil },
		func() string { return "" },
		func() string { return "" },
		"test-api-key",
		upstream.URL,
	)

	res := serve(t, h, http.MethodGet, "/linkpreview?q=https://example.com")

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	if res.Body.String() != responseBody {
		t.Fatalf("expected body %q, got %q", responseBody, res.Body.String())
	}
	if gotAPIKey != "test-api-key" {
		t.Fatalf("expected API key %q, got %q", "test-api-key", gotAPIKey)
	}
	if gotQuery != "https://example.com" {
		t.Fatalf("expected query %q, got %q", "https://example.com", gotQuery)
	}
}

func TestLinkPreviewUpstreamError(t *testing.T) {
	const responseBody = `{"title":"","description":"","error":"Unprocessable Entity"}`

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, responseBody)
	}))
	defer upstream.Close()

	h := NewAppHandler(
		func(context.Context, string) (string, error) { return "", nil },
		func(context.Context, string, string, string) error { return nil },
		func() string { return "" },
		func() string { return "" },
		"test-api-key",
		upstream.URL,
	)

	res := serve(t, h, http.MethodGet, "/linkpreview?q=https://example.com")

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, res.Code)
	}
	if res.Body.String() != responseBody {
		t.Fatalf("expected body %q, got %q", responseBody, res.Body.String())
	}
}
