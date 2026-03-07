package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestRoot(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	newHandler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var body rootResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Message != helloMessage {
		t.Fatalf("expected message %q, got %q", helloMessage, body.Message)
	}
}

func TestHealthNoJWTContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	newHandler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var body healthResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Status != "ok" {
		t.Fatalf("expected status %q, got %q", "ok", body.Status)
	}

	if body.Email != nil {
		t.Fatalf("expected nil email, got %q", *body.Email)
	}
}

func TestGetEmailWithoutRequest(t *testing.T) {
	if email := getEmailFromRequest(nil); email != nil {
		t.Fatalf("expected nil email, got %q", *email)
	}
}

func TestGetEmailWithRequestContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req = req.WithContext(context.WithValue(req.Context(), emailContextKey, "test@example.com"))

	email := getEmailFromRequest(req)
	if email == nil {
		t.Fatal("expected email, got nil")
	}

	if *email != "test@example.com" {
		t.Fatalf("expected email %q, got %q", "test@example.com", *email)
	}
}

func TestLambdaHealthIncludesJWTEmail(t *testing.T) {
	event := events.APIGatewayV2HTTPRequest{
		RawPath: "/health",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
					Claims: map[string]string{"email": "test@example.com"},
				},
			},
		},
	}

	response, err := handleLambdaInvocation(context.Background(), event)
	if err != nil {
		t.Fatalf("handle lambda invocation: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}

	var body healthResponse
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Email == nil || *body.Email != "test@example.com" {
		t.Fatalf("expected lambda email %q, got %#v", "test@example.com", body.Email)
	}
}

func TestLambdaRequestWithInvalidBase64Body(t *testing.T) {
	event := events.APIGatewayV2HTTPRequest{
		Body:            "%%%not-base64%%%",
		IsBase64Encoded: true,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodPost,
				Path:   "/",
			},
		},
	}

	_, err := handleLambdaInvocation(context.Background(), event)
	if err == nil {
		t.Fatal("expected invalid base64 error, got nil")
	}
}
