package podcastaddict

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestParseInputRawURL(t *testing.T) {
	source, err := parseInput("https://podcastaddict.com/better-offline/episode/221030037")
	if err != nil {
		t.Fatalf("parseInput returned error: %v", err)
	}

	if source.ShareTitle != "" {
		t.Fatalf("expected empty share title, got %q", source.ShareTitle)
	}
	if source.EpisodeURL != "https://podcastaddict.com/better-offline/episode/221030037" {
		t.Fatalf("unexpected episode URL %q", source.EpisodeURL)
	}
}

func TestParseInputMarkdownURL(t *testing.T) {
	source, err := parseInput("[Gastropod] Protein, Pyramids, and Politics: The Forgotten Stories and Controversial Science Behind Government Dietary Advice - [Gastropod - Protein, Pyramids, and Politics: The Forgotten Stories and Controversial Science Behind Government Dietary Advice](https://podcastaddict.com/gastropod/episode/221081444) via")
	if err != nil {
		t.Fatalf("parseInput returned error: %v", err)
	}

	if source.ShareTitle != "" {
		t.Fatalf("expected empty share title, got %q", source.ShareTitle)
	}
	if source.EpisodeURL != "https://podcastaddict.com/gastropod/episode/221081444" {
		t.Fatalf("unexpected episode URL %q", source.EpisodeURL)
	}
	if source.Input == "" {
		t.Fatal("expected input to be preserved")
	}
}

func TestParseInputShareFixtures(t *testing.T) {
	testCases := []struct {
		name        string
		fixture     string
		wantTitle   string
		wantEpisode string
	}{
		{
			name:        "overthink",
			fixture:     "overthink-share.txt",
			wantTitle:   "[Overthink] Closer Look: Levinas, On Escape",
			wantEpisode: "https://podcastaddict.com/overthink/episode/221058424",
		},
		{
			name:        "fat science",
			fixture:     "fat-science-share.txt",
			wantTitle:   "[Fat Science] Navigating the GLP-1 Wild West: A Conversation With Dr. Vin Gupta",
			wantEpisode: "https://podcastaddict.com/fat-science/episode/220958646",
		},
		{
			name:        "the indicator",
			fixture:     "the-indicator-share.txt",
			wantTitle:   "[The Indicator from Planet Money] How are drivers riding out the gas crisis?",
			wantEpisode: "https://podcastaddict.com/the-indicator-from-planet-money/episode/221125712",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source, err := parseInput(readFixture(t, tc.fixture))
			if err != nil {
				t.Fatalf("parseInput returned error: %v", err)
			}

			if source.ShareTitle != tc.wantTitle {
				t.Fatalf("expected share title %q, got %q", tc.wantTitle, source.ShareTitle)
			}
			if source.EpisodeURL != tc.wantEpisode {
				t.Fatalf("expected episode URL %q, got %q", tc.wantEpisode, source.EpisodeURL)
			}
		})
	}
}

func TestParseInputRejectsInvalidPayload(t *testing.T) {
	_, err := parseInput("[Show] Episode Title")
	if err == nil {
		t.Fatal("expected parseInput to reject missing URL")
	}
	if !strings.Contains(err.Error(), "does not contain a Podcast Addict URL") {
		t.Fatalf("unexpected error: %v", err)
	}
	if HTTPStatus(err) != http.StatusBadRequest {
		t.Fatalf("expected bad request status, got %d", HTTPStatus(err))
	}
}

func TestFetchDocumentSetsMobileUserAgent(t *testing.T) {
	var gotUserAgent string
	client := testHTTPClient(func(reqURL string, headers map[string]string) (int, string) {
		gotUserAgent = headers["User-Agent"]
		if reqURL != "https://podcastaddict.com/better-offline/episode/221030037" {
			t.Fatalf("unexpected request URL %q", reqURL)
		}
		return http.StatusOK, "<html><body><div id=\"episode_body\"><p>ok</p></div></body></html>"
	})

	if _, err := fetchDocument(context.Background(), client, "https://podcastaddict.com/better-offline/episode/221030037"); err != nil {
		t.Fatalf("fetchDocument returned error: %v", err)
	}
	if gotUserAgent != UserAgent {
		t.Fatalf("expected user agent %q, got %q", UserAgent, gotUserAgent)
	}
}

func TestParseEpisodeDocumentBetterOfflineFixture(t *testing.T) {
	result := parseFixtureDocument(t, "better-offline.html", Source{
		Input:      "https://podcastaddict.com/better-offline/episode/221030037",
		EpisodeURL: "https://podcastaddict.com/better-offline/episode/221030037",
	})

	if result.Podcast.Title != "Better Offline" {
		t.Fatalf("expected podcast title %q, got %q", "Better Offline", result.Podcast.Title)
	}
	if result.Podcast.URL != "https://podcastaddict.com/podcast/better-offline/4855593" {
		t.Fatalf("unexpected podcast URL %q", result.Podcast.URL)
	}
	if result.Episode.Title != "Better Offline - The Reality of AI Economics With Paul Kedrosky" {
		t.Fatalf("unexpected episode title %q", result.Episode.Title)
	}
	if result.Episode.PublishedAt != "2026-04-06T21:00:00-07:00" {
		t.Fatalf("unexpected published_at %q", result.Episode.PublishedAt)
	}
	if result.Episode.PublishedDate != "2026-04-06" {
		t.Fatalf("unexpected published_date %q", result.Episode.PublishedDate)
	}
	if result.Episode.Duration != "51 mins" {
		t.Fatalf("unexpected duration %q", result.Episode.Duration)
	}
	if !strings.Contains(result.Episode.DescriptionHTML, "paulkedrosky.com") {
		t.Fatalf("expected description HTML to include link, got %q", result.Episode.DescriptionHTML)
	}
	if !strings.Contains(result.Episode.DescriptionHTML, "omnystudio.com/listener") {
		t.Fatalf("expected description HTML to include privacy note, got %q", result.Episode.DescriptionHTML)
	}
}

func TestParseEpisodeDocumentInvincibleFixture(t *testing.T) {
	result := parseFixtureDocument(t, "we-was-watching.html", Source{
		Input:      "https://podcastaddict.com/we-was-watching-an-invincible-podcast/episode/221018105",
		EpisodeURL: "https://podcastaddict.com/we-was-watching-an-invincible-podcast/episode/221018105",
	})

	if result.Podcast.Title != "We Was Watching: An Invincible Podcast" {
		t.Fatalf("unexpected podcast title %q", result.Podcast.Title)
	}
	if result.Episode.Duration != "70 mins" {
		t.Fatalf("unexpected duration %q", result.Episode.Duration)
	}
	if !strings.Contains(result.Episode.DescriptionHTML, "Conquest punches a hole through Mark’s body") {
		t.Fatalf("expected dense paragraph description HTML, got %q", result.Episode.DescriptionHTML)
	}
	if !strings.Contains(result.Episode.DescriptionHTML, "#Invincible") {
		t.Fatalf("expected description HTML to preserve hashtag links, got %q", result.Episode.DescriptionHTML)
	}
}

func TestParseEpisodeDocumentErrors(t *testing.T) {
	testCases := []struct {
		name       string
		html       string
		wantErr    string
		wantStatus int
	}{
		{
			name:       "missing json ld",
			html:       `<html><body><div id="episode_body"><p>hello</p></div></body></html>`,
			wantErr:    "missing PodcastEpisode JSON-LD",
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "missing episode body",
			html:       `<html><head><script type="application/ld+json">[{"@type":"PodcastEpisode","name":"Example"}]</script></head><body></body></html>`,
			wantErr:    "missing episode description",
			wantStatus: http.StatusBadGateway,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("NewDocumentFromReader returned error: %v", err)
			}

			_, err = parseEpisodeDocument(doc, Source{EpisodeURL: "https://podcastaddict.com/example/episode/1"})
			if err == nil {
				t.Fatal("expected parseEpisodeDocument to return an error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
			if HTTPStatus(err) != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, HTTPStatus(err))
			}
		})
	}
}

func parseFixtureDocument(t *testing.T, fixture string, source Source) Result {
	t.Helper()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(readFixture(t, fixture)))
	if err != nil {
		t.Fatalf("NewDocumentFromReader returned error: %v", err)
	}

	result, err := parseEpisodeDocument(doc, source)
	if err != nil {
		t.Fatalf("parseEpisodeDocument returned error: %v", err)
	}

	return result
}

func readFixture(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned error: %v", path, err)
	}

	return string(data)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func testHTTPClient(handler func(reqURL string, headers map[string]string) (statusCode int, body string)) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			headers := make(map[string]string)
			for key, values := range req.Header {
				if len(values) > 0 {
					headers[key] = values[0]
				}
			}

			statusCode, body := handler(req.URL.String(), headers)
			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}
}
