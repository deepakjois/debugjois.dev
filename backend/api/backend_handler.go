package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type BackendHandler interface {
	HandleLambdaEvent(ctx context.Context, payload json.RawMessage) (json.RawMessage, error)
}

type lambdaBackendHandler struct {
	httpHandler http.Handler
}

type localBackendHandler struct{}

func newLambdaBackendHandler(httpHandler http.Handler) BackendHandler {
	return lambdaBackendHandler{httpHandler: httpHandler}
}

func newLocalBackendHandler() BackendHandler {
	return localBackendHandler{}
}

func (h lambdaBackendHandler) HandleLambdaEvent(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	return dispatchBackendEvent(ctx, payload, h.httpHandler)
}

func (h localBackendHandler) HandleLambdaEvent(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	if classifyEvent(payload) == eventTypeAPIGateway {
		log.Printf("Ignoring API Gateway event for local invoke; start the local server with `serve` and send the request over HTTP instead")
		return nil, nil
	}

	return dispatchBackendEvent(ctx, payload, nil)
}
