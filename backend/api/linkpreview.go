package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func newLinkPreviewHandler(apiKey, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			writeHTTPResponse(w, http.StatusBadRequest, errorResponse{Error: "q parameter is required"})
			return
		}

		upstream := fmt.Sprintf("%s/?q=%s&fields=title,description", baseURL, url.QueryEscape(q))

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstream, nil)
		if err != nil {
			writeHTTPResponse(w, http.StatusBadGateway, errorResponse{Error: "failed to fetch link preview"})
			return
		}
		req.Header.Set("X-Linkpreview-Api-Key", apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			writeHTTPResponse(w, http.StatusBadGateway, errorResponse{Error: "failed to fetch link preview"})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			writeHTTPResponse(w, http.StatusBadGateway, errorResponse{Error: "failed to fetch link preview"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
	}
}
