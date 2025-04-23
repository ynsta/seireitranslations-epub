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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
	// Fetch and parse the HTML from the URL
	doc, err := s.fetchAndParseHTML(pageURL)
	if err != nil {
		return Content{}, err
	}

	// Try each extraction pattern to find content
	contentDoc, err := s.extractContentWithPatterns(doc, pageURL, lineNum)
	if err != nil {
		return Content{}, err
	}

	// Remove empty elements
	s.removeEmptyElements(contentDoc)

	// Remove redundant elements (sharethis buttons, chapter title)
	s.removeRedundantElements(contentDoc)

	// Remove blog URL entries
	s.removeBlogURLEntries(contentDoc)

	// Convert "Part X" paragraphs to h3 subtitles
	s.convertPartHeadings(contentDoc)

	// Process navigation and Patreon elements
	s.processNavigationElements(contentDoc)

	// Convert div elements to p elements
	s.convertDivsToParagraphs(contentDoc)

	// Get the processed HTML
	processedHTML, err := contentDoc.Html()
	if err != nil {
		return Content{}, fmt.Errorf("error generating processed HTML: %v", err)
	}

	// Small delay to be nice to the server
	time.Sleep(500 * time.Millisecond)

	return Content{HTML: processedHTML}, nil
}

// fetchAndParseHTML downloads a webpage and parses the HTML
func (s *Scraper) fetchAndParseHTML(pageURL string) (*goquery.Document, error) {
	// Get the page
	resp, err := http.Get(pageURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching URL: %v", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Warn("Failed to close response body", "url", pageURL, "error", closeErr)
		}
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status code: %d", resp.StatusCode)
	}

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %v", err)
	}

	return doc, nil
}

// extractContentWithPatterns tries each extraction pattern to find content
func (s *Scraper) extractContentWithPatterns(doc *goquery.Document, pageURL string, lineNum int) (*goquery.Document, error) {
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
		return nil, fmt.Errorf("could not find content in the blog post")
	}

	// Create a document from the extracted HTML to process it
	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("error parsing content HTML: %v", err)
	}

	return contentDoc, nil
}

// removeEmptyElements removes elements without text or images
func (s *Scraper) removeEmptyElements(doc *goquery.Document) {
	doc.Find("p, div, span, h1, h2, h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
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
		slog.Debug("Content length after removing empty elements", "length", len(doc.Text()))
	}
}

// removeRedundantElements removes sharethis buttons and first chapter title
func (s *Scraper) removeRedundantElements(doc *goquery.Document) {
	// Remove the sharethis-inline-reaction-buttons div
	doc.Find(".sharethis-inline-reaction-buttons").Remove()

	if logger.Debug {
		slog.Debug("Content length after sharethis removal", "length", len(doc.Text()))
	}

	// Remove only the first chapter title element since it's redundant with the chapter title
	firstChapterTitle := doc.Find(ChapterTitleSelector).First()
	if firstChapterTitle.Length() > 0 {
		firstChapterTitle.Remove()
		if logger.Debug {
			slog.Debug("Removed first chapter title element")
		}
	}

	if logger.Debug {
		slog.Debug("Content length after chapter title removal", "length", len(doc.Text()))
	}
}

// removeBlogURLEntries removes blog URL references
func (s *Scraper) removeBlogURLEntries(doc *goquery.Document) {
	doc.Find("p, div").Each(func(i int, s *goquery.Selection) {
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
		slog.Debug("Content length after blog URL removal", "length", len(doc.Text()))
	}
}

// convertPartHeadings converts "Part X" paragraphs to h3 subtitles
func (s *Scraper) convertPartHeadings(doc *goquery.Document) {
	// Convert "Part X" paragraphs to h3 subtitles
	doc.Find("p span[style*='font-weight: 800'], p span[style*='font-weight:800']").Each(func(i int, s *goquery.Selection) {
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
	doc.Find("div > b").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.HasPrefix(strings.ToLower(text), "part ") {
			// Get the parent div
			parentDiv := s.Parent()
			// Create a new h3 element with the same text
			parentDiv.ReplaceWithHtml(fmt.Sprintf("<h3>%s</h3>", text))
			if logger.Debug {
				slog.Debug("Converted div>b Part X to h3", "text", text)
			}
		}
	})

	// Also convert <p><span><b>Part X</b></span></p> format to h3 subtitles
	doc.Find("p > span > b").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.HasPrefix(strings.ToLower(text), "part ") {
			// Get the parent div
			parentDiv := s.Parent().Parent()
			// Create a new h3 element with the same text
			parentDiv.ReplaceWithHtml(fmt.Sprintf("<h3>%s</h3>", text))
			if logger.Debug {
				slog.Debug("Converted p>span>b Part X to h3", "text", text)
			}
		}
	})

	if logger.Debug {
		slog.Debug("Content length after converting Part X to h3", "length", len(doc.Text()))
	}
}

// processNavigationElements handles centered paragraphs with navigation/Patreon
func (s *Scraper) processNavigationElements(doc *goquery.Document) {
	// Process centered paragraphs:
	// First, remove specific navigation or Patreon-related paragraphs
	doc.Find("p[style*='center'], div[style*='center']").Each(func(i int, s *goquery.Selection) {
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
		slog.Debug("Content length after removing centered paragraphs", "length", len(doc.Text()))
	}

	// Then, remove the last three centered paragraphs if they still exist
	centeredParagraphs := doc.Find("p[style*='text-align: center']")
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
		slog.Debug("Content length after removing the last three centered paragraphs", "length", len(doc.Text()))
	}
}

// convertDivsToParagraphs converts div elements to proper p elements
func (s *Scraper) convertDivsToParagraphs(doc *goquery.Document) {
	doc.Find("div").Each(func(i int, s *goquery.Selection) {
		// Skip divs with specific classes that shouldn't be converted
		if s.HasClass("sharethis-inline-reaction-buttons") ||
			s.HasClass("post-header") ||
			s.HasClass("post-footer") ||
			s.HasClass("post-bottom") {
			return
		}

		// Get the HTML content
		html, err := s.Html()
		if err != nil {
			return
		}

		// Check if it's an empty div with just a <br> tag
		if strings.TrimSpace(html) == "<br>" || strings.TrimSpace(html) == "<br/>" || strings.TrimSpace(html) == "<br />" {
			s.Remove()
			if logger.Debug {
				slog.Debug("Removed empty div with just a br tag")
			}
			return
		}

		// Check if this has meaningful text content (not just whitespace or empty)
		text := strings.TrimSpace(s.Text())
		if text != "" {
			// Check if this div only contains text (no other significant HTML elements)
			// This avoids converting container divs with nested content
			childDivs := s.Find("div").Length()
			childHeaders := s.Find("h1, h2, h3, h4, h5, h6").Length()
			childParagraphs := s.Find("p").Length()

			// If it has no divs, headers, or paragraphs, it's likely just text content
			if childDivs == 0 && childHeaders == 0 && childParagraphs == 0 {
				// Replace the div with a p element
				s.ReplaceWithHtml(fmt.Sprintf("<p>%s</p>", html))
				if logger.Debug {
					slog.Debug("Converted div to p element", "text_preview", text[:min(30, len(text))])
				}
			}
		}
	})

	if logger.Debug {
		slog.Debug("Content length after converting divs to p elements", "length", len(doc.Text()))
	}
}
