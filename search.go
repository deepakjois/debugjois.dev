package main

import (
	"fmt"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/highlight/highlighter/ansi"
)

type SearchCmd struct {
	Query string `arg:"" help:"Search query string"`
}

func (s *SearchCmd) Run() error {
	// Open existing index
	index, err := bleve.Open("debugjois-dev.bleve")
	if err != nil {
		return fmt.Errorf("failed to open index: %w", err)
	}
	defer index.Close()

	// Create search query
	query := bleve.NewQueryStringQuery(s.Query)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"Date", "Content"}
	searchRequest.Highlight = bleve.NewHighlightWithStyle(ansi.Name)
	searchRequest.Highlight.Fields = []string{"Content"}

	// Execute search
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Print results
	fmt.Printf("Found %d matches\n\n", searchResult.Total)
	for _, hit := range searchResult.Hits {
		date := hit.Fields["Date"].(string)
		fmt.Printf("\n📅 %s\n", date)

		// Print highlighted snippets
		if fragments, exists := hit.Fragments["Content"]; exists && len(fragments) > 0 {
			for _, fragment := range fragments {
				fmt.Printf("   %s\n", fragment)
			}
		}
		fmt.Println()
	}

	return nil
}
