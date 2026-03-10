package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

var primaryAllowedEmail = allowedEmails[0]

func TestRoot(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	newHandler().ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}

	var body errorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Error != "unauthorized" {
		t.Fatalf("expected error %q, got %q", "unauthorized", body.Error)
	}

}

func TestRootAllowedEmail(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), emailContextKey, primaryAllowedEmail))
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

func TestRootDisallowedEmail(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), emailContextKey, "other@example.com"))
	res := httptest.NewRecorder()

	newHandler().ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, res.Code)
	}

	var body errorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Error != "forbidden" {
		t.Fatalf("expected error %q, got %q", "forbidden", body.Error)
	}
}

func TestHealthNoJWTContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	newHandler().ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, res.Code)
	}

	var body errorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Error != "unauthorized" {
		t.Fatalf("expected error %q, got %q", "unauthorized", body.Error)
	}

}

func TestHealthAllowedEmail(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req = req.WithContext(context.WithValue(req.Context(), emailContextKey, primaryAllowedEmail))
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

	if body.Email == nil || *body.Email != primaryAllowedEmail {
		t.Fatalf("expected email %q, got %#v", primaryAllowedEmail, body.Email)
	}
}

func TestHealthAllowedEmailCaseInsensitive(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req = req.WithContext(context.WithValue(req.Context(), emailContextKey, strings.ToUpper(primaryAllowedEmail)))
	res := httptest.NewRecorder()

	newHandler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
}

func TestHealthDisallowedEmail(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req = req.WithContext(context.WithValue(req.Context(), emailContextKey, "other@example.com"))
	res := httptest.NewRecorder()

	newHandler().ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, res.Code)
	}

	var body errorResponse
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Error != "forbidden" {
		t.Fatalf("expected error %q, got %q", "forbidden", body.Error)
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
					Claims: map[string]string{"email": primaryAllowedEmail},
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

	if body.Email == nil || *body.Email != primaryAllowedEmail {
		t.Fatalf("expected lambda email %q, got %#v", primaryAllowedEmail, body.Email)
	}
}

func TestLambdaHealthRejectsDisallowedEmail(t *testing.T) {
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

	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.StatusCode)
	}

	var body errorResponse
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Error != "forbidden" {
		t.Fatalf("expected error %q, got %q", "forbidden", body.Error)
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
