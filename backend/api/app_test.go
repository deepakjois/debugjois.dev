package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func newTestHandler() http.Handler {
	return NewAppHandler(
		func(context.Context, string) (string, error) { return "", nil },
		func() string { return "" },
	)
}

func decodeJSON[T any](t *testing.T, body []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return v
}

func TestAppRootGet(t *testing.T) {
	res := serve(t, newTestHandler(), http.MethodGet, "/")

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	body := decodeJSON[rootResponse](t, res.Body.Bytes())
	if body.Message != helloMessage {
		t.Fatalf("expected message %q, got %q", helloMessage, body.Message)
	}
}

func TestAppRootHead(t *testing.T) {
	res := serve(t, newTestHandler(), http.MethodHead, "/")

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestAppNotFound(t *testing.T) {
	res := serve(t, newTestHandler(), http.MethodGet, "/missing")

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, res.Code)
	}

	if !strings.Contains(res.Body.String(), "404 page not found") {
		t.Fatalf("expected default not found body, got %q", res.Body.String())
	}
}

func TestAppMethodNotAllowed(t *testing.T) {
	res := serve(t, newTestHandler(), http.MethodPost, "/")

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}

	allow := res.Header().Get("Allow")
	if !strings.Contains(allow, http.MethodGet) || !strings.Contains(allow, http.MethodHead) {
		t.Fatalf("expected Allow header to mention GET and HEAD, got %q", allow)
	}
}

func TestAppDailyGet(t *testing.T) {
	h := NewAppHandler(
		func(_ context.Context, date string) (string, error) {
			if date != "2026-03-12" {
				t.Fatalf("expected date %q, got %q", "2026-03-12", date)
			}
			return "hello from markdown", nil
		},
		func() string { return "2026-03-12" },
	)

	res := serve(t, h, http.MethodGet, "/daily")

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	body := decodeJSON[dailyResponse](t, res.Body.Bytes())

	if body.Title != "2026-03-12.md" {
		t.Fatalf("expected title %q, got %q", "2026-03-12.md", body.Title)
	}

	if body.Contents != base64.StdEncoding.EncodeToString([]byte("hello from markdown")) {
		t.Fatalf("expected base64 of %q, got %q", "hello from markdown", body.Contents)
	}
}

func TestAppDailyGetMissingNote(t *testing.T) {
	h := NewAppHandler(
		func(_ context.Context, date string) (string, error) {
			if date != "2026-03-13" {
				t.Fatalf("expected date %q, got %q", "2026-03-13", date)
			}
			return "", nil
		},
		func() string { return "2026-03-13" },
	)

	res := serve(t, h, http.MethodGet, "/daily")

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	body := decodeJSON[dailyResponse](t, res.Body.Bytes())

	if body.Title != "2026-03-13.md" {
		t.Fatalf("expected title %q, got %q", "2026-03-13.md", body.Title)
	}

	if body.Contents != "" {
		t.Fatalf("expected empty contents, got %q", body.Contents)
	}
}

func TestAppDailyGetLoaderError(t *testing.T) {
	h := NewAppHandler(
		func(_ context.Context, _ string) (string, error) {
			return "", errors.New("boom")
		},
		func() string { return "2026-03-12" },
	)

	res := serve(t, h, http.MethodGet, "/daily")

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, res.Code)
	}

	body := decodeJSON[errorResponse](t, res.Body.Bytes())

	if body.Error != "failed to load daily note" {
		t.Fatalf("expected error %q, got %q", "failed to load daily note", body.Error)
	}
}

func TestAppDailyMethodNotAllowed(t *testing.T) {
	res := serve(t, newTestHandler(), http.MethodPost, "/daily")

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}

	allow := res.Header().Get("Allow")
	if !strings.Contains(allow, http.MethodGet) || !strings.Contains(allow, http.MethodHead) {
		t.Fatalf("expected Allow header to mention GET and HEAD, got %q", allow)
	}
}
