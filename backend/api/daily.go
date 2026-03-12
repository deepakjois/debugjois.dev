package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/matthalp/go-meridian/cet"
)

const (
	githubOwner          = "deepakjois"
	githubRepo           = "debugjois.dev"
	dailyNotesPathPrefix = "site/content/daily-notes"
)

type dailyResponse struct {
	Title    string `json:"title"`
	Contents string `json:"contents"`
}

func todayStringInCET() string {
	return cet.Now().Format("2006-01-02")
}

func currentTimestampInCET() string {
	return cet.Now().Format("2006-01-02 15:04:05")
}

func newGitHubClient() (*github.Client, error) {
	token := strings.TrimSpace(os.Getenv(githubTokenEnvVar))
	if token == "" {
		return nil, fmt.Errorf("%s is not set", githubTokenEnvVar)
	}

	return github.NewClient(&http.Client{Timeout: 10 * time.Second}).WithAuthToken(token), nil
}

func loadDailyNoteContentFromGitHub(ctx context.Context, client *github.Client, date string) (string, error) {
	path := fmt.Sprintf("%s/%s.md", dailyNotesPathPrefix, date)

	fileContent, _, _, err := client.Repositories.GetContents(ctx, githubOwner, githubRepo, path, nil)
	if err != nil {
		var githubError *github.ErrorResponse
		if errors.As(err, &githubError) && githubError.Response != nil && githubError.Response.StatusCode == http.StatusNotFound {
			return "", nil
		}

		return "", fmt.Errorf("get GitHub contents for %q: %w", path, err)
	}

	if fileContent == nil {
		return "", fmt.Errorf("GitHub returned no file content for %q", path)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("decode GitHub contents for %q: %w", path, err)
	}

	return content, nil
}

func validateDailyTitle(title, currentDate string) error {
	if title != fmt.Sprintf("%s.md", currentDate) {
		return fmt.Errorf("title must match current date %s.md", currentDate)
	}

	return nil
}

func saveDailyNoteContentToGitHub(ctx context.Context, client *github.Client, title, contents, commitMessage string) error {
	path := fmt.Sprintf("%s/%s", dailyNotesPathPrefix, title)
	opts := &github.RepositoryContentFileOptions{
		Message: new(commitMessage),
		Content: []byte(contents),
	}

	fileContent, _, _, err := client.Repositories.GetContents(ctx, githubOwner, githubRepo, path, nil)
	if err != nil {
		var githubError *github.ErrorResponse
		if !errors.As(err, &githubError) || githubError.Response == nil || githubError.Response.StatusCode != http.StatusNotFound {
			return fmt.Errorf("get GitHub contents for %q: %w", path, err)
		}
	}

	if fileContent == nil {
		_, _, err = client.Repositories.CreateFile(ctx, githubOwner, githubRepo, path, opts)
		if err != nil {
			return fmt.Errorf("create GitHub contents for %q: %w", path, err)
		}

		return nil
	}

	opts.SHA = fileContent.SHA
	_, _, err = client.Repositories.UpdateFile(ctx, githubOwner, githubRepo, path, opts)
	if err != nil {
		return fmt.Errorf("update GitHub contents for %q: %w", path, err)
	}

	return nil
}

func encodeDailyContents(contents string) string {
	return base64.StdEncoding.EncodeToString([]byte(contents))
}
