package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/web"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
)

type IndexCmd struct {
	// No flags needed for now
}

type DailyNote struct {
	Date    string
	Content string
}

type plainTextRenderer struct{}

func (r *plainTextRenderer) Render(w io.Writer, source []byte, n ast.Node) error {
	var text bytes.Buffer
	ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == ast.KindText {
			text.Write(n.Text(source))
			text.WriteByte(' ')
		}
		return ast.WalkContinue, nil
	})

	_, err := w.Write(text.Bytes())
	return err
}

func (r *plainTextRenderer) AddOptions(...renderer.Option) {}

func (i *IndexCmd) Run() error {
	// Create a new index or open existing
	mapping := bleve.NewIndexMapping()
	mapping.DefaultAnalyzer = web.Name
	index, err := bleve.New("debugjois-dev.bleve", mapping)
	if err != nil {
		if err == bleve.ErrorIndexPathExists {
			// If index exists, open it
			index, err = bleve.Open("debugjois-dev.bleve")
			if err != nil {
				return fmt.Errorf("failed to open existing index: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	defer index.Close()

	// Setup markdown to text converter
	md := goldmark.New(
		goldmark.WithRenderer(&plainTextRenderer{}),
	)

	// Walk through all markdown files
	err = filepath.Walk("content/daily-notes", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}
		// Convert markdown to plain text
		var buf bytes.Buffer
		if err := md.Convert(content, &buf); err != nil {
			return fmt.Errorf("failed to convert markdown to text for %s: %w", path, err)
		}

		// Extract date from filename
		date := strings.TrimSuffix(info.Name(), ".md")

		// Validate date format
		_, err = time.Parse("2006-01-02", date)
		if err != nil {
			return fmt.Errorf("invalid date format in filename %s: %w", info.Name(), err)
		}

		// Create document
		doc := DailyNote{
			Date:    date,
			Content: buf.String(),
		}

		// Index document
		if err := index.Index(date, doc); err != nil {
			return fmt.Errorf("failed to index document %s: %w", date, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	return nil
}
