package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"sort"

	"github.com/bitfield/script"
	"github.com/otiai10/copy"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/hashtag"
)

// Page represents the structure of a web page.
type Page struct {
	Title string
	Body  template.HTML
}

// Note represents a single daily note.
type Note struct {
	Body template.HTML
}

func main() {
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithExtensions(
			&hashtag.Extender{Variant: hashtag.ObsidianVariant},
		),
	)

	if err := generateSite(md); err != nil {
		log.Fatalf("Failed to generate site: %v", err)
	}
}

func generateSite(md goldmark.Markdown) error {
	if err := os.MkdirAll("build", 0755); err != nil {
		return fmt.Errorf("create build directory: %w", err)
	}

	if err := copy.Copy("static", "build"); err != nil {
		return fmt.Errorf("copy static files: %w", err)
	}

	tmpl, err := template.ParseFiles("templates/shell.html")
	if err != nil {
		return fmt.Errorf("parse shell template: %w", err)
	}

	if err := generateIndexPage(tmpl); err != nil {
		return fmt.Errorf("generate index page: %w", err)
	}

	if err := generateDailyNotesPage(md, tmpl); err != nil {
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
		Title: "Deepak Jois",
		Body:  template.HTML(content),
	}

	return renderPage(tmpl, "build/index.html", page)
}

func generateDailyNotesPage(md goldmark.Markdown, tmpl *template.Template) error {
	files, err := script.ListFiles("content/daily-notes/*.md").Slice()
	if err != nil {
		return fmt.Errorf("list daily notes: %w", err)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	var notes []Note
	for _, file := range files {
		var buf bytes.Buffer
		if err := convertMarkdownToHTML(md, file, &buf); err != nil {
			return fmt.Errorf("convert note %s: %w", file, err)
		}
		notes = append(notes, Note{Body: template.HTML(buf.String())})
	}

	ntmpl, err := template.ParseFiles("templates/daily.html")
	if err != nil {
		return fmt.Errorf("parse daily notes template: %w", err)
	}

	var buf bytes.Buffer
	if err := ntmpl.Execute(&buf, struct{ Notes []Note }{Notes: notes}); err != nil {
		return fmt.Errorf("execute daily notes template: %w", err)
	}

	page := Page{
		Title: "Deepak Jois · Daily Notes",
		Body:  template.HTML(buf.String()),
	}

	return renderPage(tmpl, "build/daily", page)
}

func convertMarkdownToHTML(md goldmark.Markdown, filename string, w io.Writer) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read markdown file: %w", err)
	}

	if err := md.Convert(content, w); err != nil {
		return fmt.Errorf("convert markdown to HTML: %w", err)
	}

	return nil
}

func renderPage(tmpl *template.Template, outputPath string, page Page) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, page); err != nil {
		return fmt.Errorf("execute page template: %w", err)
	}

	return nil
}
