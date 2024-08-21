package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bitfield/script"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/feeds"
	"github.com/otiai10/copy"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/hashtag"
)

type BuildCmd struct {
	Watch bool `help:"Watch for file changes and rebuild" default:"false"`
	Dev   bool `help:"Run in dev mode, which compiles scratch file and drafts" default:"false"`
	md    goldmark.Markdown
}

func (b *BuildCmd) Run() error {
	b.md = goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithExtensions(
			&hashtag.Extender{Variant: hashtag.ObsidianVariant},
		),
	)
	if !b.Watch {
		return b.generateSite()
	}

	if watcher, err := fsnotify.NewWatcher(); err != nil {
		return err
	} else {
		defer watcher.Close()

		done := make(chan bool)
		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op&fsnotify.Write == fsnotify.Write {
						fmt.Println("Modified file:", event.Name)
						err := b.generateSite()
						if err != nil {
							fmt.Printf("Build failed: %v\n", err)
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Println("Error:", err)
				}
			}
		}()

		for _, path := range []string{"content", "static", "templates"} {
			err := watcher.Add(path)
			if err != nil {
				return fmt.Errorf("error adding directory to watch: %v", err)
			}

		}
		fmt.Println("Watching for file changes. Press Ctrl+C to stop.")
		<-done
	}

	return nil
}

// Page represents the structure of a web page.
type Page struct {
	Title string
	Body  template.HTML
}

// Note represents a single daily note.
type Note struct {
	Body template.HTML
	Date string
}

func (b *BuildCmd) generateSite() error {
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

	if err := generateDailyNotesPage(b.md, tmpl); err != nil {
		return fmt.Errorf("generate daily notes page: %w", err)
	}

	if b.Dev {
		fmt.Println("generating scratch page")
		if err := generateScratchPage(b.md, tmpl); err != nil {
			return fmt.Errorf("generate scratch page: %w", err)
		}
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
		date := strings.TrimSuffix(filepath.Base(file), ".md")
		notes = append(notes, Note{Body: template.HTML(buf.String()), Date: date})
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

	if err := renderPage(tmpl, "build/daily", page); err != nil {
		return err
	}

	return renderDailyNotesFeed(notes)
}

func generateScratchPage(md goldmark.Markdown, tmpl *template.Template) error {
	var buf bytes.Buffer
	if err := convertMarkdownToHTML(md, "content/scratch.md", &buf); err != nil {
		return fmt.Errorf("convert scratch: %w", err)
	}

	page := Page{
		Title: "Scratch",
		Body:  template.HTML(buf.String()),
	}

	return renderPage(tmpl, "build/scratch", page)
}

func renderDailyNotesFeed(notes []Note) error {
	feed := &feeds.AtomFeed{
		Title:    "Deepak Jois · Daily Log",
		Subtitle: "Running log of links, code snippets and other miscellany.",
		Link:     &feeds.AtomLink{Href: "https://www.debugjois.dev/daily"},
		Author:   &feeds.AtomAuthor{AtomPerson: feeds.AtomPerson{Name: "Deepak Jois", Email: "deepak.jois@gmail.com"}},
		Logo:     "https://www.debugjois.dev/android-chrome-512x512.png",
		Icon:     "https://www.debugjois.dev/favicon.ico",
		Id:       "https://www.debugjois.dev/daily",
		Updated:  time.Now().Format(time.RFC3339),
	}

	// Load the Asia/Kolkata time zone (IST)
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		return err
	}

	for _, note := range notes {
		if note.Date == time.Now().In(ist).Format("2006-01-02") { // skip today
			continue
		}
		var updated time.Time
		if date, err := time.ParseInLocation("2006-01-02", note.Date, ist); err != nil {
			return err
		} else {
			date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, ist)
			updated = date.Add(24 * time.Hour)
		}

		entry := feeds.AtomEntry{
			Title: note.Date,
			Links: []feeds.AtomLink{
				{Href: fmt.Sprintf("https://www.debugjois.dev/daily#%s", note.Date)},
			},
			Updated: updated.Format(time.RFC3339),
			Id:      note.Date,
			Author:  &feeds.AtomAuthor{AtomPerson: feeds.AtomPerson{Name: "Deepak Jois"}},
			Content: &feeds.AtomContent{Content: string(note.Body), Type: "html"},
		}
		feed.Entries = append(feed.Entries, &entry)
	}

	f, err := os.Create("build/daily.xml")
	if err != nil {
		return fmt.Errorf("create atom file: %w", err)
	}
	defer f.Close()

	return feeds.WriteXML(feed, f)

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
