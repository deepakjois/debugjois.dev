package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

type rootResponse struct {
	Message string `json:"message"`
}

type app struct {
	loadDailyNote    func(ctx context.Context, date string) (string, error)
	currentDailyDate func() string
}

func NewAppHandler(
	loadDailyNote func(ctx context.Context, date string) (string, error),
	currentDailyDate func() string,
) http.Handler {
	a := &app{
		loadDailyNote:    loadDailyNote,
		currentDailyDate: currentDailyDate,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleRootGet)
	mux.HandleFunc("GET /daily", a.handleDailyGet)
	return mux
}

func handleRootGet(w http.ResponseWriter, _ *http.Request) {
	writeHTTPResponse(w, http.StatusOK, rootResponse{Message: helloMessage})
}

func (a *app) handleDailyGet(w http.ResponseWriter, r *http.Request) {
	date := a.currentDailyDate()
	content, err := a.loadDailyNote(r.Context(), date)
	if err != nil {
		writeHTTPResponse(w, http.StatusInternalServerError, errorResponse{Error: "failed to load daily note"})
		return
	}

	writeHTTPResponse(w, http.StatusOK, dailyResponse{
		Title:    fmt.Sprintf("%s.md", date),
		Contents: base64.StdEncoding.EncodeToString([]byte(content)),
	})
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
