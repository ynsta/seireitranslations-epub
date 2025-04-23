package scraper

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Content represents the extracted content from a web page
type Content struct {
	HTML string
}

// Scraper handles web scraping functionality
type Scraper struct {
	debug    bool
	patterns []ExtractionPattern
}

// New creates a new Scraper instance
func New(debug bool) *Scraper {
	return &Scraper{
		debug:    debug,
		patterns: DefaultPatterns(),
	}
}

// ExtractContent downloads a page and extracts content
func (s *Scraper) ExtractContent(pageURL string) (Content, error) {
	// Get the page
	resp, err := http.Get(pageURL)
	if err != nil {
		return Content{}, fmt.Errorf("error fetching URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return Content{}, fmt.Errorf("HTTP status code: %d", resp.StatusCode)
	}

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return Content{}, fmt.Errorf("error parsing HTML: %v", err)
	}

	// Try each extraction pattern
	var content string
	var found bool

	for _, pattern := range s.patterns {
		content, found = pattern.Extract(doc, pattern.Selector)
		if found {
			break
		}
	}

	if !found {
		return Content{}, fmt.Errorf("could not find content in the blog post")
	}

	// Create a document from the extracted HTML to process it
	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return Content{}, fmt.Errorf("error parsing content HTML: %v", err)
	}

	// Remove the h4 element since it's redundant with the chapter title
	contentDoc.Find("h4").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	// Remove blog URL entries (seireitranslations.blogspot.com)
	contentDoc.Find("p, div").Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		htmlLower := strings.ToLower(html)

		if strings.Contains(htmlLower, "seireitranslations.blogspot.com") {
			s.Remove()
			fmt.Printf("Removed blog URL element\n")
		}
	})

	// Convert "Part X" paragraphs to h3 subtitles
	contentDoc.Find("p span[style*='font-weight: 800'], p span[style*='font-weight:800']").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.HasPrefix(strings.ToLower(text), "part ") {
			// Get the parent paragraph
			parentP := s.Parent()
			if parentP.Is("p") {
				// Create a new h3 element with the same text
				parentP.ReplaceWithHtml(fmt.Sprintf("<h3>%s</h3>", text))
			}
		}
	})

	// Find and remove specific navigation or Patreon-related paragraphs
	contentDoc.Find("p[style*='text-align: center']").Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		htmlLower := strings.ToLower(html)

		// Remove paragraphs containing Patreon links
		if strings.Contains(htmlLower, "patreon") {
			s.Remove()
			fmt.Printf("Removed Patreon link paragraph\n")
			return
		}

		// Remove navigation paragraphs (Previous | Table of Contents | Next)
		if (strings.Contains(htmlLower, "previous") && strings.Contains(htmlLower, "next")) ||
			(strings.Contains(htmlLower, "previous") && strings.Contains(htmlLower, "table of contents")) ||
			(strings.Contains(htmlLower, "next") && strings.Contains(htmlLower, "table of contents")) {
			s.Remove()
			fmt.Printf("Removed navigation paragraph\n")
			return
		}
	})

	// Get the processed HTML
	processedHTML, err := contentDoc.Html()
	if err != nil {
		return Content{}, fmt.Errorf("error generating processed HTML: %v", err)
	}

	// Small delay to be nice to the server
	time.Sleep(500 * time.Millisecond)

	return Content{HTML: processedHTML}, nil
}
