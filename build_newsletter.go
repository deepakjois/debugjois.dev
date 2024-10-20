package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type BuildNewsletterCmd struct{}

func (cmd *BuildNewsletterCmd) Run() error {
	now := time.Now()
	lastSaturday := now.AddDate(0, 0, -int(now.Weekday())-1)
	lastSunday := lastSaturday.AddDate(0, 0, -6)

	files, err := collectFiles(lastSunday, lastSaturday)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	content, err := processFiles(files)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	fmt.Println(content)
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

		processed := transformMarkdown(string(data))
		content.WriteString(processed)
		content.WriteString("\n\n")
	}
	return content.String(), nil
}

func transformMarkdown(content string) string {
	// Replace Obsidian image embeds
	obsidianImageRegex := regexp.MustCompile(`\[\[(.+)\]\]`)
	content = obsidianImageRegex.ReplaceAllString(content, "![](https://debugjois.dev/images/$1)")

	// Strip image syntax from embeds
	twitterEmbedRegex := regexp.MustCompile(`!\[]\((.*)\)`)
	content = twitterEmbedRegex.ReplaceAllString(content, "$1")

	return content
}
