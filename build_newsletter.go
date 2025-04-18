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

	"github.com/resend/resend-go/v2"
)

type BuildNewsletterCmd struct {
	Post   bool `help:"Post to Buttondown API (BUTTONDOWN_API_KEY must be set)"`
	Notify bool `help:"Send notification email after posting (RESEND_API_KEY must be set)"`
}

// Validate ensures that --notify is only used with --post
func (cmd *BuildNewsletterCmd) Validate() error {
	if cmd.Notify && !cmd.Post {
		return fmt.Errorf("--notify can only be used with --post")
	}
	return nil
}

type ButtondownPayload struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Status  string `json:"status"`
}

// lastSaturday returns the most recent Saturday before the given time t.
// If t is Saturday, it returns the Saturday one week ago.
func lastSaturday(t time.Time) time.Time {
	diff := 0
	if t.Weekday() == time.Saturday {
		diff = -7
	} else {
		diff = -int(t.Weekday()) - 1
	}
	return t.AddDate(0, 0, diff)
}

func (cmd *BuildNewsletterCmd) Run() error {
	// Load the Asia/Kolkata time zone (IST)
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return err
	}
	now := time.Now().In(ist)
	sat := lastSaturday(now)
	sun := sat.AddDate(0, 0, -6).Truncate(24 * time.Hour)

	files, err := collectFiles(sun, sat)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	content, err := processFiles(files)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	year, weekNum := sun.ISOWeek()
	if cmd.Post {
		fmt.Fprintf(os.Stderr, "posting weekly digest for Week %d, %d to Buttondown\n", weekNum, year)
		if err := postToButtondown(content, year, weekNum); err != nil {
			return fmt.Errorf("failed to post to ButtonDown: %w", err)
		}

		if cmd.Notify {
			if err := sendNotificationEmail(year, weekNum); err != nil {
				return fmt.Errorf("failed to send notification email: %w", err)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "weekly digest for week %d, %d\n", weekNum, year)
		fmt.Println(content)
	}
	return nil
}

func postToButtondown(content string, year, weekNum int) error {
	payload := ButtondownPayload{
		Subject: fmt.Sprintf("Daily Log Digest – Week %d, %d", weekNum, year),
		Body:    "<!-- buttondown-editor-mode: plaintext -->\n" + content, // See: https://github.com/buttondown/discussions/discussions/59#discussioncomment-12251332
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

// sendNotificationEmail sends a notification email using Resend API after the newsletter has been posted
func sendNotificationEmail(year, weekNum int) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY environment variable not set")
	}

	client := resend.NewClient(apiKey)

	params := &resend.SendEmailRequest{
		From:    "hi@notifications.debugjois.dev",
		To:      []string{"deepak.jois@gmail.com"},
		Subject: fmt.Sprintf("Newsletter posted - Week %d, %d", weekNum, year),
		Html:    fmt.Sprintf("Your weekly newsletter for Week %d, %d has been posted to Buttondown.", weekNum, year),
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
