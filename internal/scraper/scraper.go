package scraper

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ynsta/seireitranslations-epub/internal/logger"
)

// Content represents the extracted content from a web page
type Content struct {
	HTML string
}

// Scraper handles web scraping functionality
type Scraper struct {
	debug    bool
	tempDir  string
	patterns []ExtractionPattern
}

// New creates a new Scraper instance
func New(tempDir string, debug bool) *Scraper {
	// Set the debug configuration
	SetDebug(debug)
	SetTempDir(tempDir)

	return &Scraper{
		debug:    debug,
		tempDir:  tempDir,
		patterns: DefaultPatterns(),
	}
}

// ExtractContent downloads a page and extracts content
func (s *Scraper) ExtractContent(pageURL string, lineNum int) (Content, error) {
	// Get the page
	resp, err := http.Get(pageURL)
	if err != nil {
		return Content{}, fmt.Errorf("error fetching URL: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Warn("Failed to close response body", "url", pageURL, "error", closeErr)
		}
	}()

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
		content, found = pattern.Extract(doc, pattern.Selector, pageURL, lineNum)
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

	// Remove empty p, div, span, and heading elements that have no text or image content
	contentDoc.Find("p, div, span, h1, h2, h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
		// Check if element has text content (trimming whitespace)
		text := strings.TrimSpace(s.Text())

		// Check if element has any img children
		hasImage := s.Find("img").Length() > 0

		// If element has no text and no images, remove it
		if text == "" && !hasImage {
			s.Remove()
			if logger.Debug {
				slog.Debug("Removed empty element", "tag", s.Get(0).Data)
			}
		}
	})

	if logger.Debug {
		slog.Debug("Content length after removing empty elements", "length", len(contentDoc.Text()))
	}

	// Remove the sharethis-inline-reaction-buttons div
	contentDoc.Find(".sharethis-inline-reaction-buttons").Remove()

	if logger.Debug {
		slog.Debug("Content length after sharethis removal", "length", len(contentDoc.Text()))
	}

	// Remove only the first chapter title element since it's redundant with the chapter title
	firstChapterTitle := contentDoc.Find(ChapterTitleSelector).First()
	if firstChapterTitle.Length() > 0 {
		firstChapterTitle.Remove()
		if logger.Debug {
			slog.Debug("Removed first chapter title element")
		}
	}

	if logger.Debug {
		slog.Debug("Content length after chapter title removal", "length", len(contentDoc.Text()))
	}

	// Remove blog URL entries (seireitranslations.blogspot.com)
	contentDoc.Find("p, div").Each(func(i int, s *goquery.Selection) {
		// Get the text content of the element
		text := strings.TrimSpace(s.Text())

		// Remove common decorative elements like dashes, arrows, etc.
		text = strings.ReplaceAll(text, "—", "")
		text = strings.ReplaceAll(text, "-", "")
		text = strings.ReplaceAll(text, ">", "")
		text = strings.ReplaceAll(text, "<", "")
		text = strings.ReplaceAll(text, "|", "")
		text = strings.ReplaceAll(text, "*", "")

		// Trim spaces again after removing decorative elements
		text = strings.TrimSpace(text)

		// Check if the cleaned text exactly matches the blog URL
		if strings.EqualFold(text, "seireitranslations.blogspot.com") {
			s.Remove()
			if logger.Debug {
				slog.Debug("Removed blog URL element", "tag", s.Get(0).Data, "original_text", s.Text())
			}
		}
	})

	if logger.Debug {
		slog.Debug("Content length after blog URL removal", "length", len(contentDoc.Text()))
	}

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

	// Also convert <div><b>Part X</b></div> format to h3 subtitles
	contentDoc.Find("div > b").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.HasPrefix(strings.ToLower(text), "part ") {
			// Get the parent div
			parentDiv := s.Parent()
			if parentDiv.Is("div") {
				// Create a new h3 element with the same text
				parentDiv.ReplaceWithHtml(fmt.Sprintf("<h3>%s</h3>", text))
				if logger.Debug {
					slog.Debug("Converted div>b Part X to h3", "text", text)
				}
			}
		}
	})

	if logger.Debug {
		slog.Debug("Content length after converting Part X to h3", "length", len(contentDoc.Text()))
	}

	// Process centered paragraphs:
	// First, remove specific navigation or Patreon-related paragraphs
	contentDoc.Find("p[style*='text-align: center']").Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		htmlLower := strings.ToLower(html)

		// Remove paragraphs containing Patreon links
		if strings.Contains(htmlLower, "patreon") {
			s.Remove()
			if logger.Debug {
				slog.Debug("Removed Patreon link paragraph")
			}
			return
		}

		// Remove navigation paragraphs (Previous | Table of Contents | Next)
		if (strings.Contains(htmlLower, "previous") && strings.Contains(htmlLower, "next")) ||
			(strings.Contains(htmlLower, "previous") && strings.Contains(htmlLower, "table of contents")) ||
			(strings.Contains(htmlLower, "next") && strings.Contains(htmlLower, "table of contents")) {
			s.Remove()
			if logger.Debug {
				slog.Debug("Removed navigation paragraph")
			}
			return
		}
	})

	if logger.Debug {
		slog.Debug("Content length after removing centered paragraphs", "length", len(contentDoc.Text()))
	}

	// Then, remove the last three centered paragraphs if they still exist
	centeredParagraphs := contentDoc.Find("p[style*='text-align: center']")
	if centeredParagraphs.Length() >= 3 {
		// Convert to slice for easier manipulation
		paragraphsToRemove := centeredParagraphs.Slice(centeredParagraphs.Length()-3, centeredParagraphs.Length())
		paragraphsToRemove.Each(func(i int, s *goquery.Selection) {
			s.Remove()
			if logger.Debug {
				slog.Debug("Removed one of the last three centered paragraphs")
			}
		})
	}

	if logger.Debug {
		slog.Debug("Content length after removing the last three centered paragraphs", "length", len(contentDoc.Text()))
	}

	// Get the processed HTML
	processedHTML, err := contentDoc.Html()
	if err != nil {
		return Content{}, fmt.Errorf("error generating processed HTML: %v", err)
	}

	// Small delay to be nice to the server
	time.Sleep(500 * time.Millisecond)

	return Content{HTML: processedHTML}, nil
}
