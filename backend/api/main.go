package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const (
	defaultPort  = "8000"
	helloMessage = "Hello from debugjois.dev Lambda!"
)

type contextKey string

const emailContextKey contextKey = "email"

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

func main() {
	if isLambdaRuntime() {
		lambda.Start(handleLambdaInvocation)
		return
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           newHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Starting HTTP server on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func isLambdaRuntime() bool {
	return strings.TrimSpace(os.Getenv("AWS_LAMBDA_RUNTIME_API")) != ""
}

func newHandler() http.Handler {
	return withCORS(http.HandlerFunc(routeRequest))
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

func routeRequest(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		handleRoot(w, r)
	case "/health":
		handleHealth(w, r)
	default:
		writeHTTPResponse(w, http.StatusNotFound, errorResponse{Error: "not found"})
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if !isReadMethod(r.Method) {
		writeHTTPResponse(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	if r.Method == http.MethodHead {
		writeHTTPResponse(w, http.StatusOK, nil)
		return
	}

	writeHTTPResponse(w, http.StatusOK, rootResponse{Message: helloMessage})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if !isReadMethod(r.Method) {
		writeHTTPResponse(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	if r.Method == http.MethodHead {
		writeHTTPResponse(w, http.StatusOK, nil)
		return
	}

	writeHTTPResponse(w, http.StatusOK, healthResponse{Status: "ok", Email: getEmailFromRequest(r)})
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

func handleLambdaInvocation(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	request, err := httpRequestFromLambdaEvent(ctx, event)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	responseRecorder := httptest.NewRecorder()
	newHandler().ServeHTTP(responseRecorder, request)
	return lambdaResponseFromRecorder(responseRecorder), nil
}

func httpRequestFromLambdaEvent(ctx context.Context, event events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	method := strings.TrimSpace(event.RequestContext.HTTP.Method)
	if method == "" {
		method = http.MethodGet
	}

	path := strings.TrimSpace(event.RawPath)
	if path == "" {
		path = strings.TrimSpace(event.RequestContext.HTTP.Path)
	}
	if path == "" {
		path = "/"
	}

	target := path
	if rawQuery := strings.TrimSpace(event.RawQueryString); rawQuery != "" {
		target = fmt.Sprintf("%s?%s", path, rawQuery)
	}

	body, err := lambdaRequestBody(event)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	for key, value := range event.Headers {
		request.Header.Set(key, value)
	}
	if len(event.Cookies) > 0 && request.Header.Get("Cookie") == "" {
		request.Header.Set("Cookie", strings.Join(event.Cookies, "; "))
	}

	if email := getEmailFromLambdaEvent(event); email != nil {
		request = request.WithContext(context.WithValue(request.Context(), emailContextKey, *email))
	}

	return request, nil
}

func lambdaRequestBody(event events.APIGatewayV2HTTPRequest) ([]byte, error) {
	if event.Body == "" {
		return nil, nil
	}

	if !event.IsBase64Encoded {
		return []byte(event.Body), nil
	}

	body, err := base64.StdEncoding.DecodeString(event.Body)
	if err != nil {
		return nil, fmt.Errorf("decode base64 request body: %w", err)
	}

	return body, nil
}

func lambdaResponseFromRecorder(responseRecorder *httptest.ResponseRecorder) events.APIGatewayV2HTTPResponse {
	headers := make(map[string]string)
	var cookies []string

	for key, values := range responseRecorder.Header() {
		if strings.EqualFold(key, "Set-Cookie") {
			cookies = append(cookies, values...)
			continue
		}
		if len(values) == 0 {
			continue
		}

		headers[key] = strings.Join(values, ",")
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode:      responseRecorder.Code,
		Headers:         headers,
		Body:            responseRecorder.Body.String(),
		IsBase64Encoded: false,
		Cookies:         cookies,
	}
}

func getEmailFromRequest(request *http.Request) *string {
	if request == nil {
		return nil
	}

	email, _ := request.Context().Value(emailContextKey).(string)
	email = strings.TrimSpace(email)
	if email == "" {
		return nil
	}

	return &email
}

func getEmailFromLambdaEvent(event events.APIGatewayV2HTTPRequest) *string {
	if event.RequestContext.Authorizer == nil || event.RequestContext.Authorizer.JWT == nil {
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
