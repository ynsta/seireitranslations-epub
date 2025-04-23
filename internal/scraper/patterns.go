package scraper

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ExtractionPattern defines a pattern for extracting content from a web page
type ExtractionPattern struct {
	Name        string
	Description string
	Selector    string
	Extract     func(doc *goquery.Document, selector string) (string, bool)
}

// DefaultPatterns returns the default set of extraction patterns
func DefaultPatterns() []ExtractionPattern {
	return []ExtractionPattern{
		{
			Name:        "Pattern1",
			Description: ".post-body > div > span > div > h4",
			Selector:    ".post-body > div > span > div > h4",
			Extract: func(doc *goquery.Document, selector string) (string, bool) {
				var content string
				found := false

				doc.Find(selector).Each(func(i int, s *goquery.Selection) {
					if found {
						return
					}

					// Get the parent div container
					parentDiv := s.Closest("div.post-body > div")
					if parentDiv.Length() == 0 {
						return
					}

					// Clone the content
					parentHTML, err := parentDiv.Html()
					if err != nil {
						return
					}

					// We found our content
					content = parentHTML
					found = true
					fmt.Printf("Found content using pattern: %s\n", selector)
				})

				return content, found
			},
		},
		{
			Name:        "Pattern2",
			Description: ".post-body > div > span > h4",
			Selector:    ".post-body > div > span > h4",
			Extract: func(doc *goquery.Document, selector string) (string, bool) {
				var content string
				found := false

				doc.Find(selector).Each(func(i int, s *goquery.Selection) {
					if found {
						return
					}

					// Get the parent div container
					parentDiv := s.Closest("div.post-body > div")
					if parentDiv.Length() == 0 {
						return
					}

					// Clone the content
					parentHTML, err := parentDiv.Html()
					if err != nil {
						return
					}

					// We found our content
					content = parentHTML
					found = true
					fmt.Printf("Found content using pattern: %s\n", selector)
				})

				return content, found
			},
		},
		{
			Name:        "Pattern3",
			Description: "div.separator > div > span > h4",
			Selector:    "div.separator > div > span > h4",
			Extract: func(doc *goquery.Document, selector string) (string, bool) {
				var content string
				found := false

				doc.Find(selector).Each(func(i int, s *goquery.Selection) {
					if found {
						return
					}

					// For this pattern, the content is in a sibling div after the h4
					// Get the parent span
					parentSpan := s.Parent()
					if parentSpan.Length() == 0 {
						return
					}

					// Find the content div (third child of the span, after the h4)
					contentDiv := parentSpan.Find("div:nth-child(3)")
					if contentDiv.Length() == 0 {
						// If exact child not found, try any div after the h4
						contentDiv = parentSpan.Find("div")
					}

					// If we found the content div
					if contentDiv.Length() > 0 {
						// Get the h4 text to use as a reference
						h4Text := s.Text()

						// Get the content HTML
						contentHTML, err := contentDiv.Html()
						if err != nil {
							return
						}

						// Combine the h4 and content
						combinedContent := fmt.Sprintf("<h4>%s</h4>%s", h4Text, contentHTML)

						// Set as our found content
						content = combinedContent
						found = true
						fmt.Printf("Found content using pattern: %s with separate content div\n", selector)
						return
					}

					// If we couldn't find the specific content div, fall back to the parent container
					parentDiv := s.Closest("div.separator")
					if parentDiv.Length() == 0 {
						return
					}

					// Clone the content
					parentHTML, err := parentDiv.Html()
					if err != nil {
						return
					}

					// We found our content
					content = parentHTML
					found = true
					fmt.Printf("Found content using pattern: %s (parent container)\n", selector)
				})

				return content, found
			},
		},
		{
			Name:        "FallbackPattern",
			Description: ".post-body h4",
			Selector:    ".post-body h4",
			Extract: func(doc *goquery.Document, selector string) (string, bool) {
				var content string
				found := false

				doc.Find(selector).Each(func(i int, s *goquery.Selection) {
					if found {
						return
					}

					// Get the parent div container
					parentDiv := s.Closest("div.post-body > div")
					if parentDiv.Length() == 0 {
						parentDiv = s.Closest("div")
					}
					if parentDiv.Length() == 0 {
						return
					}

					// Clone the content
					parentHTML, err := parentDiv.Html()
					if err != nil {
						return
					}

					// We found our content
					content = parentHTML
					found = true
					fmt.Printf("Found content using fallback pattern: %s\n", selector)
				})

				return content, found
			},
		},
		{
			Name:        "AlternativeExtraction",
			Description: "Extract main content when h4 is not found",
			Selector:    ".post-body",
			Extract: func(doc *goquery.Document, selector string) (string, bool) {
				var content string
				found := false

				// Try to extract the main content by focusing on the core post content
				mainContent := doc.Find(selector)
				if mainContent.Length() > 0 {
					mainHTML, err := mainContent.Html()
					if err == nil {
						// Create a document to clean the main content
						mainDoc, err := goquery.NewDocumentFromReader(strings.NewReader(mainHTML))
						if err == nil {
							// Remove headers, navigation, etc.
							mainDoc.Find(".post-header, .post-footer, .post-bottom").Remove()

							// Get the cleaned content
							cleanedHTML, err := mainDoc.Html()
							if err == nil {
								content = cleanedHTML
								found = true
								fmt.Printf("Used alternative extraction method\n")
							}
						}
					}
				}

				return content, found
			},
		},
	}
}
