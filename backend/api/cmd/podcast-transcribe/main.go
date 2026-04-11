package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
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

var (
	transcribePodcastFunc      = transcribe.TranscribePodcast
	persistTranscriptStoreFunc = podcastaddict.PersistTranscript
	defaultWriteFlag           = flag.Bool("write", false, "Write transcript JSON and update transcripts.json in S3")
)

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
	options, input, err := parseArgs(args, stdin)
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

	body, err := json.Marshal(transcriptionResult)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	if options.write {
		if err := persistTranscriptStoreFunc(ctx, "transcribe", result, body); err != nil {
			return err
		}
	}

	if err := writeIndentedJSON(stdout, body); err != nil {
		return fmt.Errorf("encode result: %w", err)
	}

	return nil
}

type cliOptions struct {
	write bool
}

func parseArgs(args []string, stdin io.Reader) (cliOptions, string, error) {
	flags := flag.NewFlagSet("podcast-transcribe", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	write := flags.Bool("write", *defaultWriteFlag, "Write transcript JSON and update transcripts.json in S3")
	if err := flags.Parse(args); err != nil {
		return cliOptions{}, "", fmt.Errorf("parse flags: %w", err)
	}

	if flags.NArg() > 1 {
		return cliOptions{}, "", fmt.Errorf("expected at most one positional argument")
	}
	if flags.NArg() == 1 {
		input := strings.TrimSpace(flags.Arg(0))
		if input == "" {
			return cliOptions{}, "", fmt.Errorf("input is empty")
		}
		return cliOptions{write: *write}, input, nil
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return cliOptions{}, "", fmt.Errorf("read stdin: %w", err)
	}

	input := strings.TrimSpace(string(data))
	if input == "" {
		return cliOptions{}, "", fmt.Errorf("input is empty")
	}

	return cliOptions{write: *write}, input, nil
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

func writeIndentedJSON(w io.Writer, body []byte) error {
	var indented bytes.Buffer
	if err := json.Indent(&indented, body, "", "  "); err != nil {
		return err
	}
	indented.WriteByte('\n')

	_, err := w.Write(indented.Bytes())
	return err
}
