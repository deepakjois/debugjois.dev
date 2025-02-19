package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type BuildNewsletterCmd struct {
	Post bool `help:"Post to Buttondown API (BUTTONDOWN_API_KEY must be set)"`
}

type ButtondownPayload struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Status  string `json:"status"`
}

func (cmd *BuildNewsletterCmd) Run() error {
	// Load the Asia/Kolkata time zone (IST)
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return err
	}
	now := time.Now().In(ist)
	lastSaturday := time.Date(now.Year(), now.Month(), now.Day()-int(now.Weekday())-1, 23, 59, 59, 0, ist)
	lastSunday := lastSaturday.AddDate(0, 0, -6).Truncate(24 * time.Hour)

	files, err := collectFiles(lastSunday, lastSaturday)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	content, err := processFiles(files)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	if cmd.Post {
		year, weekNum := lastSunday.ISOWeek()
		if err := postToButtondown(content, year, weekNum); err != nil {
			return fmt.Errorf("failed to post to ButtonDown: %w", err)
		}
	} else {
		fmt.Println(content)
	}
	return nil
}

func postToButtondown(content string, year, weekNum int) error {
	payload := ButtondownPayload{
		Subject: fmt.Sprintf("Daily Log Digest – Week %d, %d", weekNum, year),
		Body:    content,
		Status:  "draft",
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	apiKey := os.Getenv("BUTTONDOWN_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("BUTTONDOWN_API_KEY environment variable not set")
	}

	req, err := http.NewRequest("POST", "https://api.buttondown.com/v1/emails", &buf)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, body)
	}

	return nil
}

func collectFiles(start, end time.Time) ([]string, error) {
	var files []string
	err := filepath.Walk("content/daily-notes", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".md") {
			date, err := time.Parse("2006-01-02.md", info.Name())
			if err != nil {
				return nil // Skip files that don't match the expected format
			}
			if (date.After(start) || date.Equal(start)) && (date.Before(end) || date.Equal(end)) {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func processFiles(files []string) (string, error) {
	var content strings.Builder
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", file, err)
		}

		processed := strings.TrimRight(transformMarkdown(string(data)), "\r\n")
		content.WriteString(processed)
		content.WriteString("\n\n")
	}
	return content.String(), nil
}

func transformMarkdown(content string) string {
	// Strip image syntax from embeds
	embedRegex := regexp.MustCompile(`!\[]\((.*)\)`)
	content = embedRegex.ReplaceAllString(content, "$1")

	// Replace Obsidian image embeds
	obsidianImageRegex := regexp.MustCompile(`!\[\[(.+)\]\]`)
	content = obsidianImageRegex.ReplaceAllString(content, "![](https://debugjois.dev/images/$1)")

	return content
}
