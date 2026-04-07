package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func requireLambdaResponse(t *testing.T, response *events.APIGatewayV2HTTPResponse, message string) events.APIGatewayV2HTTPResponse {
	t.Helper()

	if response != nil {
		return *response
	}

	t.Fatal(message)
	return events.APIGatewayV2HTTPResponse{}
}

func TestAuthorizeLambdaEventMissingEmail(t *testing.T) {
	response := authorizeLambdaEvent(events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
	})
	gotResponse := requireLambdaResponse(t, response, "expected unauthorized response, got nil")

	if gotResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, gotResponse.StatusCode)
	}

	var body errorResponse
	if err := json.Unmarshal([]byte(gotResponse.Body), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Error != "unauthorized" {
		t.Fatalf("expected error %q, got %q", "unauthorized", body.Error)
	}
}

func TestAuthorizeLambdaEventDisallowedEmail(t *testing.T) {
	response := authorizeLambdaEvent(events.APIGatewayV2HTTPRequest{
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
	})
	gotResponse := requireLambdaResponse(t, response, "expected forbidden response, got nil")

	if gotResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, gotResponse.StatusCode)
	}

	var body errorResponse
	if err := json.Unmarshal([]byte(gotResponse.Body), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body.Error != "forbidden" {
		t.Fatalf("expected error %q, got %q", "forbidden", body.Error)
	}
}

func TestAuthorizeLambdaEventAllowedEmail(t *testing.T) {
	response := authorizeLambdaEvent(events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
					Claims: map[string]string{"email": allowedEmails[0]},
				},
			},
		},
	})
	if response != nil {
		t.Fatalf("expected nil response, got %#v", response)
	}
}

func TestAuthorizeLambdaEventAllowedEmailCaseInsensitive(t *testing.T) {
	response := authorizeLambdaEvent(events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
					Claims: map[string]string{"email": strings.ToUpper(allowedEmails[0])},
				},
			},
		},
	})
	if response != nil {
		t.Fatalf("expected nil response, got %#v", response)
	}
}

func TestAuthorizeLambdaEventOptionsBypassesAuth(t *testing.T) {
	response := authorizeLambdaEvent(events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodOptions,
			},
		},
	})
	if response != nil {
		t.Fatalf("expected nil response, got %#v", response)
	}
}
