package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

var allowedEmails = []string{
	"deepak.jois@gmail.com",
}

func authorizeLambdaEvent(event events.APIGatewayV2HTTPRequest) *events.APIGatewayV2HTTPResponse {
	if strings.EqualFold(strings.TrimSpace(event.RequestContext.HTTP.Method), http.MethodOptions) {
		return nil
	}

	email := getEmailFromLambdaEvent(event)
	if email == nil {
		log.Printf("Unauthorized request: no email in JWT claims method=%s path=%s", event.RequestContext.HTTP.Method, event.RequestContext.HTTP.Path)
		response := newLambdaJSONResponse(http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return &response
	}

	if !isAllowedEmail(*email) {
		log.Printf("Forbidden request: email=%s method=%s path=%s", *email, event.RequestContext.HTTP.Method, event.RequestContext.HTTP.Path)
		response := newLambdaJSONResponse(http.StatusForbidden, errorResponse{Error: "forbidden"})
		return &response
	}

	return nil
}

func isAllowedEmail(email string) bool {
	normalizedEmail := strings.TrimSpace(email)
	for _, allowedEmail := range allowedEmails {
		if strings.EqualFold(normalizedEmail, strings.TrimSpace(allowedEmail)) {
			return true
		}
	}

	return false
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

func newLambdaJSONResponse(status int, payload any) events.APIGatewayV2HTTPResponse {
	headers := corsHeaders()
	if payload == nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode:      status,
			Headers:         headers,
			IsBase64Encoded: false,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode:      http.StatusInternalServerError,
			Headers:         headers,
			Body:            http.StatusText(http.StatusInternalServerError),
			IsBase64Encoded: false,
		}
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode:      status,
		Headers:         headers,
		Body:            string(body),
		IsBase64Encoded: false,
	}
}
