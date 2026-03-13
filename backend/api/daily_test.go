package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v84/github"
)

func TestValidateDailyTitle(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		currentDate string
		wantErr     string
	}{
		{name: "valid", title: "2026-03-12.md", currentDate: "2026-03-12"},
		{name: "invalid format", title: "2026-03-12", currentDate: "2026-03-12", wantErr: "title must match current date 2026-03-12.md"},
		{name: "wrong date", title: "2026-03-11.md", currentDate: "2026-03-12", wantErr: "title must match current date 2026-03-12.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDailyTitle(tt.title, tt.currentDate)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}

func TestLoadDailyNoteContentFromGitHubMissingFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	}))
	defer server.Close()

	client := github.NewClient(server.Client())
	client.BaseURL = mustParseURL(t, server.URL+"/")

	content, err := loadDailyNoteContentFromGitHub(context.Background(), client, "2026-03-13")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if content != "### 2026-03-13\n" {
		t.Fatalf("expected default heading for missing note, got %q", content)
	}
}

func TestSaveDailyNoteContentToGitHubCreatesFileWhenMissing(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotMessage string
	var gotContent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			gotMethod = r.Method
			gotPath = r.URL.Path
			http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
		case http.MethodPut:
			gotMethod = r.Method
			gotPath = r.URL.Path

			var body struct {
				Message string `json:"message"`
				Content string `json:"content"`
				SHA     string `json:"sha"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}

			gotMessage = body.Message
			gotContent = body.Content
			if body.SHA != "" {
				t.Fatalf("expected empty sha for create, got %q", body.SHA)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"content":{"sha":"created-sha"}}`)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	client := github.NewClient(server.Client())
	client.BaseURL = mustParseURL(t, server.URL+"/")

	err := saveDailyNoteContentToGitHub(context.Background(), client, "2026-03-12.md", "hello from markdown\n", "Web Editor Update 2026-03-12 15:44:16")
	if err != nil {
		t.Fatalf("save daily note: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Fatalf("expected final write method %q, got %q", http.MethodPut, gotMethod)
	}
	if gotPath != "/repos/deepakjois/debugjois.dev/contents/site/content/daily-notes/2026-03-12.md" {
		t.Fatalf("unexpected path %q", gotPath)
	}
	if gotMessage != "Web Editor Update 2026-03-12 15:44:16" {
		t.Fatalf("unexpected commit message %q", gotMessage)
	}
	if gotContent != base64.StdEncoding.EncodeToString([]byte("hello from markdown\n")) {
		t.Fatalf("unexpected encoded content %q", gotContent)
	}
}

func TestSaveDailyNoteContentToGitHubUpdatesExistingFile(t *testing.T) {
	requestCount := 0
	var gotMessage string
	var gotContent string
	var gotSHA string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/repos/deepakjois/debugjois.dev/contents/site/content/daily-notes/2026-03-12.md" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}

		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"name":"2026-03-12.md","path":"site/content/daily-notes/2026-03-12.md","sha":"existing-sha","type":"file","content":"aGVsbG8=","encoding":"base64"}`)
		case http.MethodPut:
			var body struct {
				Message string `json:"message"`
				Content string `json:"content"`
				SHA     string `json:"sha"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}

			gotMessage = body.Message
			gotContent = body.Content
			gotSHA = body.SHA

			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"content":{"sha":"updated-sha"}}`)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	client := github.NewClient(server.Client())
	client.BaseURL = mustParseURL(t, server.URL+"/")

	err := saveDailyNoteContentToGitHub(context.Background(), client, "2026-03-12.md", "updated content\n", "Web Editor Update 2026-03-12 15:44:16")
	if err != nil {
		t.Fatalf("save daily note: %v", err)
	}

	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
	if gotMessage != "Web Editor Update 2026-03-12 15:44:16" {
		t.Fatalf("unexpected commit message %q", gotMessage)
	}
	if gotContent != base64.StdEncoding.EncodeToString([]byte("updated content\n")) {
		t.Fatalf("unexpected encoded content %q", gotContent)
	}
	if gotSHA != "existing-sha" {
		t.Fatalf("expected sha %q, got %q", "existing-sha", gotSHA)
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url %q: %v", raw, err)
	}

	return parsed
}
