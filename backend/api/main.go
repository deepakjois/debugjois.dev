package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultPort  = "8000"
	helloMessage = "Hello from debugjois.dev Lambda!"
)

type lambdaEvent struct {
	RawPath         string            `json:"rawPath,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	Body            string            `json:"body,omitempty"`
	IsBase64Encoded bool              `json:"isBase64Encoded,omitempty"`
	RequestContext  struct {
		HTTP struct {
			Method string `json:"method,omitempty"`
			Path   string `json:"path,omitempty"`
		} `json:"http,omitempty"`
		Authorizer struct {
			JWT struct {
				Claims map[string]string `json:"claims,omitempty"`
			} `json:"jwt,omitempty"`
		} `json:"authorizer,omitempty"`
	} `json:"requestContext,omitempty"`
}

type lambdaResponse struct {
	StatusCode      int               `json:"statusCode"`
	Headers         map[string]string `json:"headers,omitempty"`
	Body            string            `json:"body,omitempty"`
	IsBase64Encoded bool              `json:"isBase64Encoded"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type rootResponse struct {
	Message string `json:"message"`
}

type healthResponse struct {
	Status string  `json:"status"`
	Email  *string `json:"email"`
}

type lambdaErrorResponse struct {
	ErrorMessage string `json:"errorMessage"`
}

func main() {
	runtimeAPI := strings.TrimSpace(os.Getenv("AWS_LAMBDA_RUNTIME_API"))
	if runtimeAPI != "" {
		if err := runLambda(runtimeAPI); err != nil {
			log.Fatal(err)
		}
		return
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           http.HandlerFunc(handleLocalRequest),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Starting HTTP server on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func handleLocalRequest(w http.ResponseWriter, r *http.Request) {
	applyCORSHeaders(w.Header())

	status, payload := routeRequest(r.Method, r.URL.Path, nil)
	writeHTTPResponse(w, status, payload)
}

func routeRequest(method string, path string, email *string) (int, any) {
	if method == http.MethodOptions {
		return http.StatusNoContent, nil
	}

	switch path {
	case "/":
		if !isReadMethod(method) {
			return http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"}
		}
		if method == http.MethodHead {
			return http.StatusOK, nil
		}
		return http.StatusOK, rootResponse{Message: helloMessage}
	case "/health":
		if !isReadMethod(method) {
			return http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"}
		}
		if method == http.MethodHead {
			return http.StatusOK, nil
		}
		return http.StatusOK, healthResponse{Status: "ok", Email: email}
	default:
		return http.StatusNotFound, errorResponse{Error: "not found"}
	}
}

func isReadMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
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

func runLambda(runtimeAPI string) error {
	client := &http.Client{}
	baseURL := fmt.Sprintf("http://%s/2018-06-01/runtime", runtimeAPI)

	for {
		event, requestID, err := nextInvocation(client, baseURL)
		if err != nil {
			return err
		}

		response, err := handleLambdaInvocation(event)
		if err != nil {
			if postInvocationError(client, baseURL, requestID, err) != nil {
				return err
			}
			continue
		}

		if err := postInvocationResponse(client, baseURL, requestID, response); err != nil {
			return err
		}
	}
}

func nextInvocation(client *http.Client, baseURL string) (lambdaEvent, string, error) {
	request, err := http.NewRequest(http.MethodGet, baseURL+"/invocation/next", nil)
	if err != nil {
		return lambdaEvent{}, "", err
	}

	response, err := client.Do(request)
	if err != nil {
		return lambdaEvent{}, "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return lambdaEvent{}, "", fmt.Errorf("lambda runtime returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	requestID := strings.TrimSpace(response.Header.Get("Lambda-Runtime-Aws-Request-Id"))
	if requestID == "" {
		return lambdaEvent{}, "", fmt.Errorf("lambda runtime response missing request id")
	}

	var event lambdaEvent
	if err := json.NewDecoder(response.Body).Decode(&event); err != nil {
		return lambdaEvent{}, "", err
	}

	return event, requestID, nil
}

func handleLambdaInvocation(event lambdaEvent) (lambdaResponse, error) {
	path := strings.TrimSpace(event.RawPath)
	if path == "" {
		path = strings.TrimSpace(event.RequestContext.HTTP.Path)
	}
	if path == "" {
		path = "/"
	}

	status, payload := routeRequest(strings.TrimSpace(event.RequestContext.HTTP.Method), path, getEmailFromRequest(&event))
	response := lambdaResponse{
		StatusCode:      status,
		Headers:         corsHeaders(),
		IsBase64Encoded: false,
	}

	if payload == nil {
		return response, nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return lambdaResponse{}, err
	}

	response.Body = string(body)
	return response, nil
}

func postInvocationResponse(client *http.Client, baseURL string, requestID string, response lambdaResponse) error {
	payload, err := json.Marshal(response)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, baseURL+"/invocation/"+requestID+"/response", bytes.NewReader(payload))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	postResponse, err := client.Do(request)
	if err != nil {
		return err
	}
	defer postResponse.Body.Close()

	if postResponse.StatusCode != http.StatusAccepted && postResponse.StatusCode != http.StatusOK && postResponse.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(postResponse.Body)
		return fmt.Errorf("lambda runtime rejected response with %s: %s", postResponse.Status, strings.TrimSpace(string(body)))
	}

	return nil
}

func postInvocationError(client *http.Client, baseURL string, requestID string, invocationErr error) error {
	payload, err := json.Marshal(lambdaErrorResponse{ErrorMessage: invocationErr.Error()})
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, baseURL+"/invocation/"+requestID+"/error", bytes.NewReader(payload))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Lambda-Runtime-Function-Error-Type", "Runtime.Error")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("lambda runtime rejected error with %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	return nil
}

func getEmailFromRequest(event *lambdaEvent) *string {
	if event == nil {
		return nil
	}

	email := strings.TrimSpace(event.RequestContext.Authorizer.JWT.Claims["email"])
	if email == "" {
		return nil
	}

	return &email
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
