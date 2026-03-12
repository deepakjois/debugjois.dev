package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	return serveBody(t, h, method, path, nil)
}

func serveBody(t *testing.T, h http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func newTestHandler() http.Handler {
	return NewAppHandler(
		func(context.Context, string) (string, error) { return "", nil },
		func(context.Context, string, string, string) error { return nil },
		func() string { return "" },
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
		nil,
		func() string { return "2026-03-12" },
		nil,
	)

	res := serve(t, h, http.MethodGet, "/daily")

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	body := decodeJSON[dailyResponse](t, res.Body.Bytes())

	if body.Title != "2026-03-12.md" {
		t.Fatalf("expected title %q, got %q", "2026-03-12.md", body.Title)
	}

	if body.Contents != encodeDailyContents("hello from markdown") {
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
		nil,
		func() string { return "2026-03-13" },
		nil,
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
		nil,
		func() string { return "2026-03-12" },
		nil,
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
	res := serve(t, newTestHandler(), http.MethodPut, "/daily")

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}

	allow := res.Header().Get("Allow")
	if !strings.Contains(allow, http.MethodGet) || !strings.Contains(allow, http.MethodHead) || !strings.Contains(allow, http.MethodPost) {
		t.Fatalf("expected Allow header to mention GET, HEAD, and POST, got %q", allow)
	}
}

func TestAppDailyPost(t *testing.T) {
	var gotTitle string
	var gotContents string
	var gotCommitMessage string

	h := NewAppHandler(
		func(context.Context, string) (string, error) { return "", nil },
		func(_ context.Context, title, contents, commitMessage string) error {
			gotTitle = title
			gotContents = contents
			gotCommitMessage = commitMessage
			return nil
		},
		func() string { return "2026-03-12" },
		func() string { return "2026-03-12 15:44:16" },
	)

	payload := []byte(`{"title":"2026-03-12.md","contents":"` + encodeDailyContents("hello from markdown\nadditional line\n") + `"}`)
	res := serveBody(t, h, http.MethodPost, "/daily", payload)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	body := decodeJSON[dailyResponse](t, res.Body.Bytes())
	if body.Title != "2026-03-12.md" {
		t.Fatalf("expected title %q, got %q", "2026-03-12.md", body.Title)
	}
	if body.Contents != encodeDailyContents("hello from markdown\nadditional line\n") {
		t.Fatalf("expected contents to round-trip, got %q", body.Contents)
	}
	if gotTitle != "2026-03-12.md" {
		t.Fatalf("expected saved title %q, got %q", "2026-03-12.md", gotTitle)
	}
	if gotContents != "hello from markdown\nadditional line\n" {
		t.Fatalf("expected saved contents to decode base64, got %q", gotContents)
	}
	if gotCommitMessage != "Web Editor Update 2026-03-12 15:44:16" {
		t.Fatalf("expected commit message %q, got %q", "Web Editor Update 2026-03-12 15:44:16", gotCommitMessage)
	}
}

func TestAppDailyPostInvalidJSON(t *testing.T) {
	h := NewAppHandler(nil, nil, func() string { return "2026-03-12" }, nil)

	res := serveBody(t, h, http.MethodPost, "/daily", []byte(`{"title":`))

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}

	body := decodeJSON[errorResponse](t, res.Body.Bytes())
	if body.Error != "invalid request body" {
		t.Fatalf("expected error %q, got %q", "invalid request body", body.Error)
	}
}

func TestAppDailyPostInvalidTitle(t *testing.T) {
	h := NewAppHandler(nil, nil, func() string { return "2026-03-12" }, nil)

	res := serveBody(t, h, http.MethodPost, "/daily", []byte(`{"title":"2026-03-12","contents":"`+encodeDailyContents("hello")+`"}`))

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}

	body := decodeJSON[errorResponse](t, res.Body.Bytes())
	if body.Error != "title must match current date 2026-03-12.md" {
		t.Fatalf("expected error %q, got %q", "title must match current date 2026-03-12.md", body.Error)
	}
}

func TestAppDailyPostWrongDate(t *testing.T) {
	h := NewAppHandler(nil, nil, func() string { return "2026-03-12" }, nil)

	res := serveBody(t, h, http.MethodPost, "/daily", []byte(`{"title":"2026-03-11.md","contents":"`+encodeDailyContents("hello")+`"}`))

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}

	body := decodeJSON[errorResponse](t, res.Body.Bytes())
	if body.Error != "title must match current date 2026-03-12.md" {
		t.Fatalf("expected error %q, got %q", "title must match current date 2026-03-12.md", body.Error)
	}
}

func TestAppDailyPostInvalidBase64(t *testing.T) {
	h := NewAppHandler(nil, nil, func() string { return "2026-03-12" }, nil)

	res := serveBody(t, h, http.MethodPost, "/daily", []byte(`{"title":"2026-03-12.md","contents":"%%%"}`))

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}

	body := decodeJSON[errorResponse](t, res.Body.Bytes())
	if body.Error != "contents must be valid base64" {
		t.Fatalf("expected error %q, got %q", "contents must be valid base64", body.Error)
	}
}

func TestAppDailyPostSaveError(t *testing.T) {
	h := NewAppHandler(
		nil,
		func(context.Context, string, string, string) error { return errors.New("boom") },
		func() string { return "2026-03-12" },
		func() string { return "2026-03-12 15:44:16" },
	)

	res := serveBody(t, h, http.MethodPost, "/daily", []byte(`{"title":"2026-03-12.md","contents":"`+encodeDailyContents("hello")+`"}`))

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, res.Code)
	}

	body := decodeJSON[errorResponse](t, res.Body.Bytes())
	if body.Error != "failed to save daily note" {
		t.Fatalf("expected error %q, got %q", "failed to save daily note", body.Error)
	}
}
