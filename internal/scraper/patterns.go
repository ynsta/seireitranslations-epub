package scraper

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ynsta/seireitranslations-epub/internal/logger"
)

// ChapterTitleSelector is the CSS selector for identifying chapter titles
// This is used both for extraction patterns and for removing redundant titles
const ChapterTitleSelector = "h4[style*=center]:first-of-type, p[style*=center]:first-of-type, p>span[style*='800']"

// debugConfig holds debugging configuration
type debugConfig struct {
	enabled bool
	tempDir string
}

// global debug configuration
var debugCfg debugConfig

// SetDebug enables or disables debug output
func SetDebug(enabled bool) {
	debugCfg.enabled = enabled
}

// SetTempDir sets the temporary directory for debug files
func SetTempDir(dir string) {
	debugCfg.tempDir = dir
}

// sanitizeFilename creates a safe filename from a title by removing special characters
func sanitizeFilename(s string) string {
	// Replace special characters with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	safe := re.ReplaceAllString(s, "_")

	// Truncate if too long (filesystem limits)
	if len(safe) > 50 {
		safe = safe[:50]
	}

	return safe
}

// saveDebugHTML saves HTML content to a file for debugging
func saveDebugHTML(lineNum int, suffix, content string, urlOrTitle string) {
	if !debugCfg.enabled || debugCfg.tempDir == "" {
		return
	}

	// Create a safe identifier from the URL or title
	identifier := sanitizeFilename(urlOrTitle)

	// Truncate the identifier if it's too long
	if len(identifier) > 30 {
		identifier = identifier[:30]
	}

	filePath := filepath.Join(debugCfg.tempDir, fmt.Sprintf("debug_%d_%s_%s.html", lineNum, identifier, suffix))
	err := os.WriteFile(filePath, []byte(content), 0600)
	if err != nil {
		slog.Error("Error saving debug HTML", "error", err)
	} else if logger.Debug {
		slog.Debug("Saved debug HTML", "path", filePath)
	}
}

// isSameElement checks if two goquery.Selection objects represent the same DOM element
func isSameElement(a, b *goquery.Selection) bool {
	if a.Length() == 0 || b.Length() == 0 {
		return false
	}

	// Compare node types
	if a.Get(0).Type != b.Get(0).Type {
		return false
	}

	// Compare tag names
	if a.Get(0).Data != b.Get(0).Data {
		return false
	}

	// Compare HTML content
	aHtml, _ := a.Html()
	bHtml, _ := b.Html()
	return aHtml == bHtml
}

// ExtractionPattern defines a pattern for extracting content from a web page
type ExtractionPattern struct {
	Name        string
	Description string
	Selector    string
	Extract     func(doc *goquery.Document, selector string, url string, lineNum int) (string, bool)
}

// DefaultPatterns returns the default set of extraction patterns
func DefaultPatterns() []ExtractionPattern {
	return []ExtractionPattern{
		{
			Name:        "AdvancedPattern",
			Description: "Advanced selector matching h4, p, or span with style",
			Selector:    "h4[style*=center]:first-of-type, p[style*=center]:first-of-type, p>span[style*='800']",
			Extract: func(doc *goquery.Document, selector string, url string, lineNum int) (string, bool) {
				var content string
				found := false
				urlIdForDebug := url // Used in debug file names

				if debugCfg.enabled {
					// Save the original HTML for debugging
					html, _ := doc.Html()
					saveDebugHTML(lineNum, "original", html, urlIdForDebug)
					if logger.Debug {
						slog.Debug("Processing URL", "url", url)
					}
				}

				// First, try to find the title element
				titleElement := doc.Find(selector).First()
				if titleElement.Length() == 0 {
					if logger.Debug {
						slog.Debug("Title element not found with selector", "selector", selector)
					}

					if debugCfg.enabled {
						// Try alternative selectors
						if logger.Debug {
							slog.Debug("Trying alternative selectors for content", "url", url)
						}
						alternativeSelectors := []string{
							"h4[style*='text-align: center']",
							"div.separator h4",
							"div.separator span h4",
							"h4",
						}

						for _, altSelector := range alternativeSelectors {
							if logger.Debug {
								slog.Debug("Trying alternative selector", "selector", altSelector)
							}
							titleElement = doc.Find(altSelector).First()
							if titleElement.Length() > 0 {
								if logger.Debug {
									slog.Debug("Found title with alternative selector", "selector", altSelector)
								}

								// Save the title element HTML
								if debugCfg.enabled {
									titleHtml, _ := titleElement.Html()
									saveDebugHTML(lineNum, "title_element", titleHtml, urlIdForDebug)

									// Save the title element's outer HTML
									titleOuterHtml, _ := titleElement.Parent().Html()
									saveDebugHTML(lineNum, "title_outer", titleOuterHtml, urlIdForDebug)
								}
								break
							}
						}

						if titleElement.Length() == 0 {
							if logger.Debug {
								slog.Debug("Could not find title element with any selector")
							}
							return content, found
						}
					} else {
						return content, found
					}
				}

				if debugCfg.enabled {
					// Save the title element HTML
					titleHtml, _ := titleElement.Html()
					saveDebugHTML(lineNum, "title", titleHtml, urlIdForDebug)

					// Save the title element's parent HTML
					parentHtml, _ := titleElement.Parent().Html()
					saveDebugHTML(lineNum, "title_parent", parentHtml, urlIdForDebug)

					if logger.Debug {
						slog.Debug("Title element details",
							"tag", titleElement.Get(0).Data,
							"html", titleHtml)
					}
				}

				// Find the parent div with "post-body" or "post-content" class
				var parentDiv *goquery.Selection
				titleElement.Parents().Each(func(i int, s *goquery.Selection) {
					if parentDiv != nil {
						return
					}
					if s.Is("div") && (s.HasClass("post-body") || s.HasClass("post-content")) {
						parentDiv = s
						if logger.Debug {
							slog.Debug("Found parent div with post-body/post-content class")
						}

						if debugCfg.enabled {
							// Save the parent div HTML
							parentHtml, _ := s.Html()
							saveDebugHTML(lineNum, "parent_div", parentHtml, urlIdForDebug)
						}
					}
				})

				// If we couldn't find the specific parent, try any parent div as fallback
				if parentDiv == nil {
					titleElement.Parents().Each(func(i int, s *goquery.Selection) {
						if parentDiv != nil {
							return
						}
						if s.Is("div") {
							parentDiv = s
							if logger.Debug {
								slog.Debug("Found fallback parent div")
							}

							if debugCfg.enabled {
								// Save the fallback parent div HTML
								parentHtml, _ := s.Html()
								saveDebugHTML(lineNum, "fallback_parent", parentHtml, urlIdForDebug)
								if logger.Debug {
									slog.Debug("Fallback parent div details", "tag", s.Get(0).Data)

									// Log parent div classes
									classes, exists := s.Attr("class")
									if exists {
										slog.Debug("Fallback parent div classes", "classes", classes)
									} else {
										slog.Debug("Fallback parent div has no classes")
									}
								}
							}
						}
					})
				}

				// If still no parent div, use the document body as last resort
				if parentDiv == nil {
					parentDiv = doc.Find("body")
					if logger.Debug {
						slog.Debug("Using body as parent container")
					}
					if parentDiv.Length() == 0 {
						if logger.Debug {
							slog.Debug("Could not find a suitable parent container")
						}
						return content, found
					}

					if debugCfg.enabled {
						// Save the body HTML
						bodyHtml, _ := parentDiv.Html()
						saveDebugHTML(lineNum, "body", bodyHtml, urlIdForDebug)
					}
				}

				// Clone the parent div for processing
				parentClone := parentDiv.Clone()

				if debugCfg.enabled {
					// Save the cloned parent HTML
					clonedHtml, _ := parentClone.Html()
					saveDebugHTML(lineNum, "cloned_parent", clonedHtml, urlIdForDebug)
				}

				// Find the title element in the cloned content
				clonedTitle := parentClone.Find(selector).First()
				if clonedTitle.Length() == 0 {
					if logger.Debug {
						slog.Debug("Could not find title element in cloned content")
					}

					if debugCfg.enabled {
						// Try to find with alternative selectors
						for _, altSelector := range []string{
							"h4[style*='text-align: center']",
							"div.separator h4",
							"div.separator span h4",
							"h4",
						} {
							clonedTitle = parentClone.Find(altSelector).First()
							if clonedTitle.Length() > 0 {
								if logger.Debug {
									slog.Debug("Found cloned title with alternative selector", "selector", altSelector)
								}
								break
							}
						}

						if clonedTitle.Length() == 0 {
							if logger.Debug {
								slog.Debug("Could not find cloned title with any selector")
							}
							return content, found
						}
					} else {
						return content, found
					}
				}

				if debugCfg.enabled {
					// Save the cloned title HTML
					clonedTitleHtml, _ := clonedTitle.Html()
					saveDebugHTML(lineNum, "cloned_title", clonedTitleHtml, urlIdForDebug)
				}

				// Convert all elements to an array for easier processing
				var allElements []*goquery.Selection
				parentClone.Find("*").Each(func(i int, el *goquery.Selection) {
					allElements = append(allElements, el)

					if debugCfg.enabled {
						if logger.Debug {
							// Log element info for debugging
							tagName := el.Get(0).Data
							html, _ := el.Html()
							htmlPreview := html
							if len(html) > 20 {
								htmlPreview = html[:20]
							}
							slog.Debug("Element details", "index", i, "tag", tagName, "html_preview", htmlPreview)
						}
					}
				})

				if debugCfg.enabled && logger.Debug {
					slog.Debug("Found elements in parent container", "count", len(allElements))
				}

				// Find the index of our title element
				titleIndex := -1
				for i, el := range allElements {
					if isSameElement(el, clonedTitle) {
						titleIndex = i
						if debugCfg.enabled && logger.Debug {
							slog.Debug("Found title element", "index", i)
						}
						break
					}
				}

				if titleIndex == -1 {
					fmt.Printf("Could not find title element index in array\n")

					if debugCfg.enabled {
						// Try a different approach to find the title
						for i, el := range allElements {
							elHtml, _ := el.Html()
							titleHtml, _ := clonedTitle.Html()

							if strings.Contains(elHtml, titleHtml) {
								titleIndex = i
								if logger.Debug {
									slog.Debug("Found title element using HTML comparison", "index", i)
								}
								break
							}
						}

						if titleIndex == -1 {
							if logger.Debug {
								slog.Debug("Still could not find title element index")
							}
							return content, found
						}
					} else {
						return content, found
					}
				}

				// Remove all elements that come before the title element
				for i := 0; i < titleIndex; i++ {
					el := allElements[i]
					// Skip if this is the title element or contains the title
					if isSameElement(el, clonedTitle) || el.Find(selector).Length() > 0 {
						if debugCfg.enabled && logger.Debug {
							slog.Debug("Skipping element (is title or contains title)", "index", i)
						}
						continue
					}

					// Skip if this is a parent of the title
					isParent := false
					clonedTitle.Parents().Each(func(j int, parent *goquery.Selection) {
						if isSameElement(parent, el) {
							isParent = true
							if debugCfg.enabled && logger.Debug {
								slog.Debug("Element is a parent of the title", "index", i)
							}
						}
					})

					if !isParent {
						el.Remove()
						if logger.Debug {
							slog.Debug("Removed element before title")
						}
					}
				}

				if debugCfg.enabled {
					// Save the HTML after removing elements before title
					afterRemovalHtml, _ := parentClone.Html()
					saveDebugHTML(lineNum, "after_removal", afterRemovalHtml, urlIdForDebug)
				}

				// Get the processed HTML
				processedHTML, err := parentClone.Html()
				if err != nil {
					slog.Error("Error getting HTML from processed content", "error", err)
					return content, found
				}

				if debugCfg.enabled {
					// Save the final processed HTML
					saveDebugHTML(lineNum, "final", processedHTML, urlIdForDebug)
					if logger.Debug {
						slog.Debug("Final HTML details", "length", len(processedHTML))
					}
				}

				content = processedHTML
				found = true
				if logger.Debug {
					slog.Debug("Successfully extracted content after title element")
				}

				return content, found
			},
		},
		{
			Name:        "FallbackPattern",
			Description: "Extract main content as fallback",
			Selector:    ".post-body",
			Extract: func(doc *goquery.Document, selector string, url string, lineNum int) (string, bool) {
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
								if logger.Debug {
									slog.Debug("Used fallback extraction method")
								}
							}
						}
					}
				}

				return content, found
			},
		},
	}
}
