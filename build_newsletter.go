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

// NewsletterWeek holds the calculated week information for a newsletter.
type NewsletterWeek struct {
	Start   AppTimezone // Sunday (start of the newsletter period)
	End     AppTimezone // Saturday (end of the newsletter period)
	Year    int         // ISO year from the Monday of the week
	WeekNum int         // ISO week number from the Monday of the week
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

type ButtondownResponse struct {
	ID string `json:"id"`
}

// calculateNewsletterWeek computes the newsletter week information for a given time.
// The newsletter covers Sunday to Saturday of the previous week.
// The week number is based on the Monday of that week (ISO week standard).
func calculateNewsletterWeek(t AppTimezone) NewsletterWeek {
	sat := lastSaturday(t).Truncate(24 * time.Hour)
	sun := sat.AddDate(0, 0, -6)
	// Use Monday (sun + 1 day) to get the ISO week number
	// This ensures the week number is consistent with the week the newsletter covers
	mon := sun.AddDate(0, 0, 1)
	year, weekNum := mon.ISOWeek()

	return NewsletterWeek{
		Start:   sun,
		End:     sat,
		Year:    year,
		WeekNum: weekNum,
	}
}

// lastSaturday returns the most recent Saturday before the given time t.
// If t is Saturday, it returns the Saturday one week ago.
func lastSaturday(t AppTimezone) AppTimezone {
	diff := 0
	if t.Weekday() == time.Saturday {
		diff = -7
	} else {
		diff = -int(t.Weekday()) - 1
	}
	return t.AddDate(0, 0, diff)
}

func (cmd *BuildNewsletterCmd) Run() error {
	week := calculateNewsletterWeek(Now())

	files, err := collectFiles(week.Start, week.End)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	content, err := processFiles(files)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	if cmd.Post {
		fmt.Fprintf(os.Stderr, "posting weekly digest for Week %d, %d (%s to %s) to Buttondown\n", week.WeekNum, week.Year, week.Start.Format("2006-01-02"), week.End.Format("2006-01-02"))
		draftURL, err := postToButtondown(content, week.Year, week.WeekNum)
		if err != nil {
			return fmt.Errorf("failed to post to ButtonDown: %w", err)
		}
		fmt.Fprintf(os.Stderr, "draft created: %s\n", draftURL)

		if cmd.Notify {
			if err := sendNotificationEmail(week.Year, week.WeekNum, draftURL); err != nil {
				return fmt.Errorf("failed to send notification email: %w", err)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "weekly digest for week %d %d (%s to %s)\n", week.WeekNum, week.Year, week.Start.Format("2006-01-02"), week.End.Format("2006-01-02"))
		fmt.Println(content)
	}
	return nil
}

func postToButtondown(content string, year, weekNum int) (string, error) {
	payload := ButtondownPayload{
		Subject: fmt.Sprintf("Daily Log Digest – Week %d, %d", weekNum, year),
		Body:    "<!-- buttondown-editor-mode: plaintext -->\n" + content, // See: https://github.com/buttondown/discussions/discussions/59#discussioncomment-12251332
		Status:  "draft",
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return "", fmt.Errorf("failed to encode JSON payload: %w", err)
	}

	apiKey := os.Getenv("BUTTONDOWN_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("BUTTONDOWN_API_KEY environment variable not set")
	}

	req, err := http.NewRequest("POST", "https://api.buttondown.com/v1/emails", &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, body)
	}

	var response ButtondownResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	draftURL := fmt.Sprintf("https://buttondown.com/emails/%s", response.ID)
	return draftURL, nil
}

func collectFiles(start, end AppTimezone) ([]string, error) {
	var files []string
	err := filepath.Walk("content/daily-notes", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(info.Name(), ".md") {
			// Parse using the app timezone helper
			dateStr := strings.TrimSuffix(info.Name(), ".md")
			date, err := ParseDate(dateStr)
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
func sendNotificationEmail(year, weekNum int, draftURL string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY environment variable not set")
	}

	client := resend.NewClient(apiKey)

	params := &resend.SendEmailRequest{
		From:    "debugjois.dev NewsletterBot <hi@notifications.debugjois.dev>",
		To:      []string{"deepak.jois@gmail.com"},
		Subject: fmt.Sprintf("Newsletter posted - Week %d, %d", weekNum, year),
		Html:    fmt.Sprintf("Your weekly newsletter for Week %d, %d has been posted to Buttondown.<br><br>Edit draft: <a href=\"%s\">%s</a>", weekNum, year, draftURL, draftURL),
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
