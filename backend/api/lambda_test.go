package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestLambdaRootAuthorized(t *testing.T) {
	response, err := handleLambdaInvocation(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/",
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
	}, NewAppHandler(nil, nil, nil, nil, "", ""))
	if err != nil {
		t.Fatalf("handle lambda invocation: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if _, ok := body["message"]; !ok {
		t.Fatalf("expected message field in response, got %#v", body)
	}
}

func TestLambdaUnauthorizedRejectedBeforeRouter(t *testing.T) {
	response, err := handleLambdaInvocation(context.Background(), events.APIGatewayV2HTTPRequest{
		RawPath: "/",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodGet,
			},
		},
	}, NewAppHandler(nil, nil, nil, nil, "", ""))
	if err != nil {
		t.Fatalf("handle lambda invocation: %v", err)
	}

	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.StatusCode)
	}
}

func TestLambdaRequestWithInvalidBase64Body(t *testing.T) {
	_, err := handleLambdaInvocation(context.Background(), events.APIGatewayV2HTTPRequest{
		Body:            "%%%not-base64%%%",
		IsBase64Encoded: true,
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: http.MethodPost,
				Path:   "/",
			},
			Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
				JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
					Claims: map[string]string{"email": allowedEmails[0]},
				},
			},
		},
	}, NewAppHandler(nil, nil, nil, nil, "", ""))
	if err == nil {
		t.Fatal("expected invalid base64 error, got nil")
	}
}
