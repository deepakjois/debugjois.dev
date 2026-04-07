package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type errorResponse struct {
	Error string `json:"error"`
}

type rootResponse struct {
	Message string `json:"message"`
}

type app struct {
	loadDailyNote      func(ctx context.Context, date string) (string, error)
	saveDailyNote      func(ctx context.Context, title, contents, commitMessage string) error
	currentDailyDate   func() string
	currentTimestamp   func() string
	linkPreviewAPIKey  string
	linkPreviewBaseURL string
}

func NewAppHandler(
	loadDailyNote func(ctx context.Context, date string) (string, error),
	saveDailyNote func(ctx context.Context, title, contents, commitMessage string) error,
	currentDailyDate func() string,
	currentTimestamp func() string,
	linkPreviewAPIKey string,
	linkPreviewBaseURL string,
) http.Handler {
	a := &app{
		loadDailyNote:      loadDailyNote,
		saveDailyNote:      saveDailyNote,
		currentDailyDate:   currentDailyDate,
		currentTimestamp:   currentTimestamp,
		linkPreviewAPIKey:  linkPreviewAPIKey,
		linkPreviewBaseURL: linkPreviewBaseURL,
	}
	return newHTTPHandler(a)
}

func buildAppHandler() http.Handler {
	return NewAppHandler(
		loadDailyNoteContentFromDrive,
		saveDailyNoteContentToDrive,
		todayStringInCET,
		currentTimestampInCET,
		os.Getenv(linkPreviewAPIKeyEnvVar),
		linkPreviewBaseURL,
	)
}

func newHTTPHandler(a *app) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleRootGet)
	mux.HandleFunc("GET /daily", a.handleDailyGet)
	mux.HandleFunc("POST /daily", a.handleDailyPost)
	mux.HandleFunc("GET /linkpreview", newLinkPreviewHandler(a.linkPreviewAPIKey, a.linkPreviewBaseURL))
	return mux
}

func handleRootGet(w http.ResponseWriter, _ *http.Request) {
	writeHTTPResponse(w, http.StatusOK, rootResponse{Message: helloMessage})
}

func (a *app) handleDailyGet(w http.ResponseWriter, r *http.Request) {
	date := a.currentDailyDate()
	content, err := a.loadDailyNote(r.Context(), date)
	if err != nil {
		log.Printf("loadDailyNote error: %v", err)
		writeHTTPResponse(w, http.StatusInternalServerError, errorResponse{Error: "failed to load daily note"})
		return
	}

	writeHTTPResponse(w, http.StatusOK, dailyResponse{
		Title:    fmt.Sprintf("%s.md", date),
		Contents: encodeDailyContents(content),
	})
}

func (a *app) handleDailyPost(w http.ResponseWriter, r *http.Request) {
	var payload dailyResponse

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeHTTPResponse(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeHTTPResponse(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	payload.Title = strings.TrimSpace(payload.Title)

	if err := validateDailyTitle(payload.Title, a.currentDailyDate()); err != nil {
		writeHTTPResponse(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	contents, err := base64.StdEncoding.DecodeString(payload.Contents)
	if err != nil {
		writeHTTPResponse(w, http.StatusBadRequest, errorResponse{Error: "contents must be valid base64"})
		return
	}

	commitMessage := fmt.Sprintf("Web Editor Update %s", a.currentTimestamp())
	if err := a.saveDailyNote(r.Context(), payload.Title, string(contents), commitMessage); err != nil {
		log.Printf("saveDailyNote error: %v", err)
		writeHTTPResponse(w, http.StatusInternalServerError, errorResponse{Error: "failed to save daily note"})
		return
	}

	writeHTTPResponse(w, http.StatusOK, payload)
}

func writeHTTPResponse(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
