package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bitfield/script"
	"github.com/gorilla/feeds"
	"github.com/otiai10/copy"
	lo "github.com/samber/lo"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/anchor"
	"go.abhg.dev/goldmark/hashtag"
)

type BuildCmd struct {
	Dev     bool `help:"Run in dev mode, which compiles scratch file and drafts" default:"false"`
	Rebuild bool `help:"Rebuild the entire archive" default:"false"`
	md      goldmark.Markdown
}

func (b *BuildCmd) Run() error {
	b.md = goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithExtensions(
			&hashtag.Extender{Variant: hashtag.ObsidianVariant},
			&ObsidianImageExtender{ImagePath: "/images/"},
			&ObsidianEmbedExtender{},
			extension.NewLinkify(),
			&anchor.Extender{},
		),
	)
	return b.generateSite()
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

// GroupedNotes represents a list of notes grouped by month
type GroupedNotes struct {
	Month string
	Notes []*Note
}

func (b *BuildCmd) generateSite() error {
	if err := os.MkdirAll("build/images", 0755); err != nil {
		return fmt.Errorf("create build directory: %w", err)
	}

	if err := copy.Copy("static", "build"); err != nil {
		return fmt.Errorf("copy static files: %w", err)
	}

	if err := copy.Copy("content/daily-notes/attachments", "build/images"); err != nil {
		return fmt.Errorf("copy attachments: %w", err)
	}

	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return fmt.Errorf("parse shell template: %w", err)
	}

	if err := generateIndexPage(tmpl); err != nil {
		return fmt.Errorf("generate index page: %w", err)
	}

	notes, err := getAllNotes(b.md)
	if err != nil {
		return fmt.Errorf("get all notes: %w", err)
	}

	if err := generateDailyNotesPage(tmpl, lo.Slice(notes, 0, 31)); err != nil {
		return fmt.Errorf("generate daily notes page: %w", err)
	}

	grouped := groupAllNotes(notes)

	if err := generateDailyNotesArchive(tmpl, grouped, b.Rebuild); err != nil {
		return fmt.Errorf("generate daily notes archive page: %w", err)
	}

	if err := generateDailyNotesArchiveIndex(tmpl, grouped); err != nil {
		return fmt.Errorf("generate daily notes archive page: %w", err)
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

func getAllNotes(md goldmark.Markdown) (notes []*Note, err error) {
	files, err := script.ListFiles("content/daily-notes/*.md").Slice()
	if err != nil {
		return nil, fmt.Errorf("list daily notes: %w", err)
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	for _, file := range files {
		// Avoid files created by Google Drive sync conflicts
		// See: https://github.com/deepakjois/debugjois.dev/issues/13
		if strings.Contains(file, "conflict") {
			fmt.Printf("skipping conflict file: %s\n", file)
			continue
		}

		var buf bytes.Buffer
		if err := convertMarkdownToHTML(md, file, &buf); err != nil {
			return nil, fmt.Errorf("convert note %s: %w", file, err)
		}
		date := strings.TrimSuffix(filepath.Base(file), ".md")

		notes = append(notes, &Note{Body: template.HTML(buf.String()), Date: date})
	}
	return notes, nil
}

// Group and sort notes by YYYY-MM
func groupAllNotes(notes []*Note) []GroupedNotes {
	groups := lo.GroupBy(notes, func(n *Note) string {
		return n.Date[0:7]
	})

	months := lo.MapToSlice(groups, func(key string, value []*Note) GroupedNotes {
		return GroupedNotes{
			Month: key,
			Notes: value,
		}
	})

	sort.Slice(months, func(i, j int) bool {
		return months[i].Month > months[j].Month
	})

	return months
}

func generateDailyNotesPage(tmpl *template.Template, notes []*Note) error {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "daily.html", struct{ Notes []*Note }{Notes: lo.Slice(notes, 0, 31)}); err != nil {
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

func generateDailyNotesArchive(tmpl *template.Template, grouped []GroupedNotes, rebuild bool) error {
	// prune to last two months unless rebuild is explicitly requested
	if !rebuild {
		grouped = lo.Slice(grouped, 0, 2)
	}

	for _, month := range grouped {
		// Format month
		t, _ := time.Parse("2006-01", month.Month)
		s := t.Format("Jan 2006")

		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "daily-archive.html", GroupedNotes{
			Notes: month.Notes,
			Month: s,
		}); err != nil {
			return fmt.Errorf("execute daily notes archive template: %w", err)
		}

		page := Page{
			Title: "Deepak Jois · Daily Notes " + s,
			Body:  template.HTML(buf.String()),
		}

		if err := renderPage(tmpl, fmt.Sprintf("build/daily-archive-%s", month.Month), page); err != nil {
			return err
		}
	}

	return nil
}

func generateDailyNotesArchiveIndex(tmpl *template.Template, grouped []GroupedNotes) error {
	type CalEntry struct {
		Link bool
		Day  string
	}

	type Month struct {
		Entries     []CalEntry
		DisplayName string
		Slug        string
	}

	var months []Month

	for _, m := range grouped {
		t, _ := time.Parse("2006-01", m.Month)
		grid := make([]CalEntry, 42)
		idx := t.Weekday()
		currDay := t
		for {
			entry := CalEntry{Day: fmt.Sprintf("%02d", currDay.Day())}
			// check if there is an entry for the day
			entry.Link = lo.ContainsBy(m.Notes, func(n *Note) bool {
				return n.Date == fmt.Sprintf("%s-%02d", m.Month, currDay.Day())
			})
			grid[idx] = entry

			idx = idx + 1
			currDay = currDay.AddDate(0, 0, 1)
			if currDay.Year() > t.Year() || currDay.Month() > t.Month() {
				break
			}
		}

		months = append(months, Month{
			Entries:     grid,
			DisplayName: t.Format("Jan 2006"),
			Slug:        m.Month,
		})
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "daily-archive-index.html", struct{ Months []Month }{months}); err != nil {
		return fmt.Errorf("execute daily notes archive index template: %w", err)
	}

	page := Page{
		Title: "Deepak Jois · Daily Notes Archive Index",
		Body:  template.HTML(buf.String()),
	}

	return renderPage(tmpl, "build/daily-archive-index", page)
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

func renderDailyNotesFeed(notes []*Note) error {
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

	if err := tmpl.ExecuteTemplate(f, "shell.html", page); err != nil {
		return fmt.Errorf("execute page template: %w", err)
	}

	return nil
}
