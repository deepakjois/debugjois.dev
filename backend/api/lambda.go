package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

func newLambdaHandler(app http.Handler) func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return func(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return handleLambdaInvocation(ctx, event, app)
	}
}

func handleLambdaInvocation(ctx context.Context, event events.APIGatewayV2HTTPRequest, app http.Handler) (events.APIGatewayV2HTTPResponse, error) {
	if authResponse := authorizeLambdaEvent(event); authResponse != nil {
		return *authResponse, nil
	}

	request, err := httpRequestFromLambdaEvent(ctx, event)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, err
	}

	responseRecorder := httptest.NewRecorder()
	app.ServeHTTP(responseRecorder, request)
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
