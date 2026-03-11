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
	if isLambdaRuntime() {
		if err := loadLambdaGitHubToken(ctx); err != nil {
			log.Fatal(err)
		}

		app := NewAppHandler()
		lambda.Start(newLambdaHandler(app))
		return
	}

	if err := loadLocalEnvFile(); err != nil {
		log.Fatal(err)
	}

	app := NewAppHandler()
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
