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

type healthResponse struct {
	Status string `json:"status"`
}

type appRouter struct {
	routes map[string]map[string]http.HandlerFunc
}

func NewAppHandler() http.Handler {
	router := &appRouter{
		routes: map[string]map[string]http.HandlerFunc{
			"/": {
				http.MethodGet:  handleRootGet,
				http.MethodHead: handleNoBody,
			},
			"/health": {
				http.MethodGet:  handleHealthGet,
				http.MethodHead: handleNoBody,
			},
		},
	}

	return withCORS(router)
}

func (router *appRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	methodHandlers, ok := router.routes[r.URL.Path]
	if !ok {
		writeHTTPResponse(w, http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}

	handler, ok := methodHandlers[r.Method]
	if !ok {
		writeHTTPResponse(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	handler(w, r)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applyCORSHeaders(w.Header())
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleRootGet(w http.ResponseWriter, _ *http.Request) {
	writeHTTPResponse(w, http.StatusOK, rootResponse{Message: helloMessage})
}

func handleHealthGet(w http.ResponseWriter, _ *http.Request) {
	writeHTTPResponse(w, http.StatusOK, healthResponse{Status: "ok"})
}

func handleNoBody(w http.ResponseWriter, _ *http.Request) {
	writeHTTPResponse(w, http.StatusOK, nil)
}

func writeHTTPResponse(w http.ResponseWriter, status int, payload any) {
	if payload == nil {
		w.WriteHeader(status)
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func corsHeaders() map[string]string {
	return map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Headers": "Content-Type, X-Amz-Date, Authorization, X-Api-Key, X-Amz-Security-Token",
		"Access-Control-Allow-Methods": "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS",
		"Content-Type":                 "application/json",
	}
}

func applyCORSHeaders(headers http.Header) {
	for key, value := range corsHeaders() {
		headers.Set(key, value)
	}
}
