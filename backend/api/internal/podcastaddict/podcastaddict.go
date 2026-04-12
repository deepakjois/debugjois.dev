package podcastaddict

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const UserAgent = "Mozilla/5.0 (Android 14; Mobile; rv:124.0) Gecko/124.0 Firefox/124.0"

var (
	podcastAddictMarkdownURLPattern = regexp.MustCompile(`\((https://(?:www\.)?podcastaddict\.com/[^)\s]+)\)`)
	podcastAddictURLPattern         = regexp.MustCompile(`https://(?:www\.)?podcastaddict\.com/[^\s)]+`)
	podcastEpisodePathPattern       = regexp.MustCompile(`(^|/)episode/\d+/?$`)
)

type ErrorKind int

const (
	ErrorKindInvalidInput ErrorKind = iota + 1
	ErrorKindUpstream
	ErrorKindInternal
)

type Error struct {
	Kind ErrorKind
	Err  error
}

func (e *Error) Error() string {
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func HTTPStatus(err error) int {
	var target *Error
	if errors.As(err, &target) {
		switch target.Kind {
		case ErrorKindInvalidInput:
			return http.StatusBadRequest
		case ErrorKindUpstream:
			return http.StatusBadGateway
		default:
			return http.StatusInternalServerError
		}
	}

	return http.StatusInternalServerError
}

type Result struct {
	Source  Source  `json:"source"`
	Podcast Podcast `json:"podcast"`
	Episode Episode `json:"episode"`
}

type Source struct {
	Input      string `json:"input"`
	ShareTitle string `json:"share_title,omitempty"`
	EpisodeURL string `json:"episode_url"`
}

type Podcast struct {
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
}

type Episode struct {
	Title           string `json:"title"`
	PublishedAt     string `json:"published_at,omitempty"`
	PublishedDate   string `json:"published_date,omitempty"`
	Duration        string `json:"duration,omitempty"`
	AudioURL        string `json:"audio_url,omitempty"`
	DescriptionHTML string `json:"description_html"`
}

type podcastEpisodeJSONLD struct {
	Type            string `json:"@type"`
	URL             string `json:"url"`
	Name            string `json:"name"`
	DatePublished   string `json:"datePublished"`
	Description     string `json:"description"`
	AssociatedMedia struct {
		ContentURL string `json:"contentUrl"`
	} `json:"associatedMedia"`
	PartOfSeries struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"partOfSeries"`
}

func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

func ParseEpisode(ctx context.Context, client *http.Client, rawInput string) (Result, error) {
	source, err := parseInput(rawInput)
	if err != nil {
		return Result{}, err
	}

	doc, err := fetchDocument(ctx, client, source.EpisodeURL)
	if err != nil {
		return Result{}, err
	}

	return parseEpisodeDocument(doc, source)
}

func parseInput(raw string) (Source, error) {
	input := strings.TrimSpace(raw)
	if input == "" {
		return Source{}, invalidInputError("input is empty")
	}

	matchedURL := extractEpisodeURL(input)
	if matchedURL == "" {
		return Source{}, invalidInputError("input does not contain a Podcast Addict URL")
	}

	episodeURL, err := normalizeEpisodeURL(matchedURL)
	if err != nil {
		return Source{}, err
	}

	source := Source{
		Input:      input,
		EpisodeURL: episodeURL,
	}

	lines := strings.Split(input, "\n")
	if len(lines) > 1 {
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if strings.Contains(trimmed, episodeURL) || podcastAddictURLPattern.MatchString(trimmed) {
				break
			}
			source.ShareTitle = trimmed
			break
		}
	}

	return source, nil
}

func extractEpisodeURL(input string) string {
	match := podcastAddictMarkdownURLPattern.FindStringSubmatch(input)
	if len(match) == 2 {
		return match[1]
	}

	return podcastAddictURLPattern.FindString(input)
}

func normalizeEpisodeURL(raw string) (string, error) {
	parsed, err := neturl.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", invalidInputError("parse episode URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return "", invalidInputError("expected Podcast Addict HTTPS URL")
	}
	if !isPodcastAddictHost(parsed.Host) {
		return "", invalidInputError("expected Podcast Addict URL")
	}
	if !podcastEpisodePathPattern.MatchString(parsed.EscapedPath()) {
		return "", invalidInputError("expected Podcast Addict episode URL")
	}

	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""

	return parsed.String(), nil
}

func isPodcastAddictHost(host string) bool {
	switch strings.ToLower(host) {
	case "podcastaddict.com", "www.podcastaddict.com":
		return true
	default:
		return false
	}
}

func fetchDocument(ctx context.Context, client *http.Client, episodeURL string) (*goquery.Document, error) {
	if client == nil {
		client = NewHTTPClient()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, episodeURL, nil)
	if err != nil {
		return nil, internalError("build request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, upstreamError("fetch episode page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, upstreamError("fetch episode page: unexpected status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, upstreamError("parse episode HTML: %w", err)
	}

	return doc, nil
}

func parseEpisodeDocument(doc *goquery.Document, source Source) (Result, error) {
	episodeJSONLD, err := extractPodcastEpisodeJSONLD(doc)
	if err != nil {
		return Result{}, err
	}

	descriptionHTML, _, err := extractEpisodeDescription(doc)
	if err != nil {
		return Result{}, err
	}

	_, duration := extractVisibleMetadata(doc)

	publishedAt := decodeHTMLString(episodeJSONLD.DatePublished)
	publishedDate := ""
	if publishedAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, publishedAt)
		if err == nil {
			publishedDate = parsedTime.Format("2006-01-02")
		}
	}

	return Result{
		Source: source,
		Podcast: Podcast{
			Title: decodeHTMLString(episodeJSONLD.PartOfSeries.Name),
			URL:   decodeHTMLString(episodeJSONLD.PartOfSeries.URL),
		},
		Episode: Episode{
			Title:           decodeHTMLString(episodeJSONLD.Name),
			PublishedAt:     publishedAt,
			PublishedDate:   publishedDate,
			Duration:        duration,
			AudioURL:        decodeHTMLString(episodeJSONLD.AssociatedMedia.ContentURL),
			DescriptionHTML: descriptionHTML,
		},
	}, nil
}

func extractPodcastEpisodeJSONLD(doc *goquery.Document) (podcastEpisodeJSONLD, error) {
	var (
		found   bool
		episode podcastEpisodeJSONLD
	)

	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		candidate, ok := decodePodcastEpisodeScript(strings.TrimSpace(selection.Text()))
		if !ok {
			return true
		}

		episode = candidate
		found = true
		return false
	})

	if !found {
		return podcastEpisodeJSONLD{}, upstreamError("missing PodcastEpisode JSON-LD")
	}

	return episode, nil
}

func decodePodcastEpisodeScript(script string) (podcastEpisodeJSONLD, bool) {
	if script == "" {
		return podcastEpisodeJSONLD{}, false
	}

	var array []json.RawMessage
	if err := json.Unmarshal([]byte(script), &array); err == nil {
		for _, item := range array {
			if episode, ok := decodePodcastEpisodeObject(item); ok {
				return episode, true
			}
		}
		return podcastEpisodeJSONLD{}, false
	}

	return decodePodcastEpisodeObject([]byte(script))
}

func decodePodcastEpisodeObject(data []byte) (podcastEpisodeJSONLD, bool) {
	var probe struct {
		Type string `json:"@type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return podcastEpisodeJSONLD{}, false
	}
	if probe.Type != "PodcastEpisode" {
		return podcastEpisodeJSONLD{}, false
	}

	var episode podcastEpisodeJSONLD
	if err := json.Unmarshal(data, &episode); err != nil {
		return podcastEpisodeJSONLD{}, false
	}

	return episode, true
}

func extractEpisodeDescription(doc *goquery.Document) (string, string, error) {
	selection := doc.Find("div#episode_body").First()
	if selection.Length() == 0 {
		return "", "", upstreamError("missing episode description")
	}

	descriptionHTML, err := selection.Html()
	if err != nil {
		return "", "", upstreamError("extract episode description HTML: %w", err)
	}

	descriptionText := selectionText(selection)
	if descriptionText == "" {
		return "", "", upstreamError("episode description is empty")
	}

	return strings.TrimSpace(descriptionHTML), descriptionText, nil
}

func extractVisibleMetadata(doc *goquery.Document) (string, string) {
	spans := doc.Find("div.titlestack h5 span")
	if spans.Length() == 0 {
		return "", ""
	}

	displayDate := cleanText(spans.First().Text())
	duration := ""
	if spans.Length() > 1 {
		duration = cleanText(spans.Eq(1).Text())
	}

	return displayDate, duration
}

func selectionText(selection *goquery.Selection) string {
	paragraphs := selection.Find("p")
	if paragraphs.Length() == 0 {
		return cleanText(selection.Text())
	}

	parts := make([]string, 0, paragraphs.Length())
	paragraphs.Each(func(_ int, paragraph *goquery.Selection) {
		text := cleanText(paragraph.Text())
		if text != "" {
			parts = append(parts, text)
		}
	})

	if len(parts) == 0 {
		return cleanText(selection.Text())
	}

	return strings.Join(parts, "\n\n")
}

func cleanText(value string) string {
	value = strings.ReplaceAll(value, "\u00a0", " ")
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func decodeHTMLString(value string) string {
	decoded := value
	for range 3 {
		next := html.UnescapeString(decoded)
		if next == decoded {
			break
		}
		decoded = next
	}
	return decoded
}

func invalidInputError(format string, args ...any) error {
	return &Error{
		Kind: ErrorKindInvalidInput,
		Err:  fmt.Errorf(format, args...),
	}
}

func upstreamError(format string, args ...any) error {
	return &Error{
		Kind: ErrorKindUpstream,
		Err:  fmt.Errorf(format, args...),
	}
}

func internalError(format string, args ...any) error {
	return &Error{
		Kind: ErrorKindInternal,
		Err:  fmt.Errorf(format, args...),
	}
}
