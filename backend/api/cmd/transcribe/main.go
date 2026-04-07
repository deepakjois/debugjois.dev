package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/deepakjois/debugjois.dev/backend/api/internal/podcastaddict"
	"github.com/deepakjois/debugjois.dev/backend/api/internal/transcribe"
	"github.com/joho/godotenv"
)

var transcribePodcastFunc = transcribe.TranscribePodcast

const cliTimeout = 10 * time.Minute

func main() {
	if err := loadCLIEnv(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cliTimeout)
	defer cancel()

	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout, newHTTPClient()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, client *http.Client) error {
	input, err := readInput(args, stdin)
	if err != nil {
		return err
	}

	result, err := podcastaddict.ParseEpisode(ctx, client, input)
	if err != nil {
		return err
	}

	transcriptionResult, err := transcribePodcastFunc(ctx, result, nil)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(transcriptionResult); err != nil {
		return fmt.Errorf("encode result: %w", err)
	}

	return nil
}

func readInput(args []string, stdin io.Reader) (string, error) {
	if len(args) > 1 {
		return "", fmt.Errorf("expected at most one positional argument")
	}
	if len(args) == 1 {
		input := strings.TrimSpace(args[0])
		if input == "" {
			return "", fmt.Errorf("input is empty")
		}
		return input, nil
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}

	input := strings.TrimSpace(string(data))
	if input == "" {
		return "", fmt.Errorf("input is empty")
	}

	return input, nil
}

func newHTTPClient() *http.Client {
	return podcastaddict.NewHTTPClient()
}

func loadCLIEnv() error {
	if err := godotenv.Overload(".env"); err != nil {
		return fmt.Errorf("load local env file %q: %w", ".env", err)
	}

	if strings.TrimSpace(os.Getenv(transcribe.DeepgramAPIKeyEnvVar)) == "" {
		return fmt.Errorf("%s must be set in %s for local CLI use", transcribe.DeepgramAPIKeyEnvVar, ".env")
	}

	return nil
}
