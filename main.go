package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"slices"

	"github.com/bitfield/script"
	cp "github.com/otiai10/copy"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/hashtag"
)

// Page represents the structure of a web page.
type Page struct {
	Title string
	Body  template.HTML
}

func main() {
	if err := generateSite(); err != nil {
		log.Fatalf("Failed to generate site: %v", err)
	}
}

func generateSite() error {
	if err := os.MkdirAll("build/", 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := cp.Copy("static", "build"); err != nil {
		return fmt.Errorf("copy static files: %w", err)
	}

	tmpl, err := template.ParseFiles("templates/shell.html")
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	if err := generateIndexPage(tmpl); err != nil {
		return fmt.Errorf("generate index page: %w", err)
	}

	if err := generateDailyNotesPage(tmpl); err != nil {
		return fmt.Errorf("generate daily notes page: %w", err)
	}

	return nil
}

func generateIndexPage(tmpl *template.Template) error {
	content, err := os.ReadFile("content/index.html")
	if err != nil {
		return fmt.Errorf("read index content: %w", err)
	}

	page := Page{
		Title: "Coming Soon",
		Body:  template.HTML(content),
	}

	return renderPage(tmpl, "build/index.html", page)
}

func generateDailyNotesPage(tmpl *template.Template) error {
	notes, err := script.ListFiles("content/daily-notes/*.md").Slice()
	if err != nil {
		return fmt.Errorf("list daily notes: %w", err)
	}

	var buf bytes.Buffer
	for _, note := range slices.Backward(notes) {
		if err := convertMarkdownToHTML(note, &buf); err != nil {
			return fmt.Errorf("convert note %s: %w", note, err)
		}
	}

	page := Page{
		Title: "Daily Notes",
		Body:  template.HTML(buf.String()),
	}

	return renderPage(tmpl, "build/daily", page)
}

func convertMarkdownToHTML(filename string, w io.Writer) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithExtensions(
			&hashtag.Extender{Variant: hashtag.ObsidianVariant},
		),
	)

	if err := md.Convert(content, w); err != nil {
		return fmt.Errorf("convert markdown: %w", err)
	}

	return nil
}

func renderPage(tmpl *template.Template, outputPath string, page Page) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, page); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}
