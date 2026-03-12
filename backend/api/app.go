package main

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

type rootResponse struct {
	Message string `json:"message"`
}

func NewAppHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleRootGet)
	return mux
}

func handleRootGet(w http.ResponseWriter, _ *http.Request) {
	writeHTTPResponse(w, http.StatusOK, rootResponse{Message: helloMessage})
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
