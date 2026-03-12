package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

const (
	defaultPort  = "8000"
	helloMessage = "Hello from debugjois.dev Lambda!"
)

func main() {
	ctx := context.Background()
	isLambda := isLambdaRuntime()

	if isLambda {
		if err := loadLambdaGitHubToken(ctx); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := loadLocalEnvFile(); err != nil {
			log.Fatal(err)
		}
	}

	client, err := newGitHubClient()
	if err != nil {
		log.Fatal(err)
	}

	app := NewAppHandler(
		func(ctx context.Context, date string) (string, error) {
			return loadDailyNoteContentFromGitHub(ctx, client, date)
		},
		todayStringInCET,
	)

	if isLambda {
		lambda.Start(newLambdaHandler(app))
		return
	}

	// Local dev serves the API directly, so add CORS here.
	app = withLocalCORS(app)

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           app,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Starting HTTP server on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func isLambdaRuntime() bool {
	return strings.TrimSpace(os.Getenv("AWS_LAMBDA_RUNTIME_API")) != ""
}

func withLocalCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applyCORSHeaders(w.Header())
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
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
