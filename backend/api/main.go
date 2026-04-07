package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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
	lambdaRuntime := isLambdaRuntime()

	if !lambdaRuntime && len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(1)
	}

	var err error
	if lambdaRuntime {
		err = loadLambdaSecrets(ctx)
	} else {
		err = loadLocalEnvFile()
	}
	if err != nil {
		log.Fatal(err)
	}

	httpHandler := buildAppHandler()

	if lambdaRuntime {
		lambda.Start(newLambdaBackendHandler(httpHandler).HandleLambdaEvent)
		return
	}

	switch os.Args[1] {
	case "serve":
		runServe(httpHandler)
	case "invoke":
		if err := runInvoke(ctx, os.Args[2:], newLocalBackendHandler(), os.Stdin, os.Stdout); err != nil {
			log.Fatal(err)
		}
	default:
		printUsage(os.Stderr)
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "Usage: %s <command> [args]\n\n", os.Args[0])
	fmt.Fprintf(w, "Commands:\n")
	fmt.Fprintf(w, "  serve    Start the local HTTP server\n")
	fmt.Fprintf(w, "  invoke   Process a JSON payload (from stdin or --payload file)\n")
}

func runServe(httpHandler http.Handler) {
	handler := withLocalCORS(httpHandler)

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Starting HTTP server on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func runInvoke(ctx context.Context, args []string, handler BackendHandler, stdin io.Reader, stdout io.Writer) error {
	invokeFlags := flag.NewFlagSet("invoke", flag.ContinueOnError)
	invokeFlags.SetOutput(io.Discard)
	payloadFile := invokeFlags.String("payload", "", "Path to JSON payload file (reads from stdin if not set)")
	if err := invokeFlags.Parse(args); err != nil {
		return fmt.Errorf("parse invoke flags: %w", err)
	}
	if invokeFlags.NArg() != 0 {
		return fmt.Errorf("invoke does not accept positional arguments")
	}

	payload, err := readInvokePayload(*payloadFile, stdin)
	if err != nil {
		return err
	}

	result, err := handler.HandleLambdaEvent(ctx, payload)
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	if _, err := fmt.Fprintln(stdout, string(result)); err != nil {
		return fmt.Errorf("write invoke response: %w", err)
	}

	return nil
}

func readInvokePayload(payloadFile string, stdin io.Reader) (json.RawMessage, error) {
	var (
		payload []byte
		err     error
	)

	if strings.TrimSpace(payloadFile) != "" {
		payload, err = os.ReadFile(payloadFile)
	} else {
		payload, err = io.ReadAll(stdin)
	}
	if err != nil {
		return nil, fmt.Errorf("read invoke payload: %w", err)
	}

	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return nil, errors.New("invoke payload is empty")
	}

	return json.RawMessage(payload), nil
}

func isLambdaRuntime() bool {
	return strings.TrimSpace(os.Getenv("AWS_LAMBDA_RUNTIME_API")) != ""
}

func withLocalCORS(next http.Handler) http.Handler {
	headers := corsHeaders()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mirror the Lambda response headers on the local server so browser-based development behaves the same way.
		for key, value := range headers {
			w.Header().Set(key, value)
		}

		// Preflight requests do not need to reach the app handler.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
