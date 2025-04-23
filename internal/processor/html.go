package processor

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-readability"
	"github.com/ynsta/seireitranslations-epub/internal/logger"
)

// HTMLProcessor handles HTML content processing
type HTMLProcessor struct {
	debug   bool
	tempDir string
}

// NewHTMLProcessor creates a new HTMLProcessor
func NewHTMLProcessor() *HTMLProcessor {
	return &HTMLProcessor{}
}

// SetDebug sets the debug flag
func (p *HTMLProcessor) SetDebug(debug bool) {
	p.debug = debug
}

// SetTempDir sets the temporary directory for debug files
func (p *HTMLProcessor) SetTempDir(dir string) {
	p.tempDir = dir
}

// sanitizeFilename creates a safe filename from a title by removing special characters
func sanitizeFilename(title string) string {
	// Replace special characters with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	safe := re.ReplaceAllString(title, "_")

	// Truncate if too long (filesystem limits)
	if len(safe) > 50 {
		safe = safe[:50]
	}

	return safe
}

// CleanHTML removes inline styles and other unnecessary attributes
func (p *HTMLProcessor) CleanHTML(html string, contentTitle string) string {
	// Debug logging for input HTML
	if p.debug {
		if logger.Debug {
			slog.Debug("CleanHTML input", "title", contentTitle, "length", len(html))

			// Log a sample of the HTML for debugging
			if len(html) > 100 {
				slog.Debug("CleanHTML input sample", "sample", html[:100])
			}
		}

		// Check for sharethis-inline-reaction-buttons div
		if strings.Contains(html, "sharethis-inline-reaction-buttons") {
			slog.Warn("Found sharethis-inline-reaction-buttons div in content", "title", contentTitle)

			// Check if this is the only content
			if len(html) < 200 && strings.Contains(html, "sharethis-inline-reaction-buttons") {
				slog.Error("Content appears to be just a sharethis div, not actual content", "title", contentTitle)

				// Try to read the debug file with the actual content
				safeTitle := sanitizeFilename(contentTitle)
				debugFilePath := filepath.Join(p.tempDir, fmt.Sprintf("debug_%s_final.html", safeTitle))
				if debugContent, err := os.ReadFile(debugFilePath); err == nil {
					slog.Info("Found debug file with actual content", "length", len(debugContent))
					html = string(debugContent)
				} else {
					slog.Error("Failed to read debug file", "error", err)
				}
			}
		}
	}

	// Create a document to work with
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		slog.Error("Failed to parse HTML in CleanHTML", "error", err)
		return html
	}

	// Remove sharethis-inline-reaction-buttons div
	doc.Find(".sharethis-inline-reaction-buttons").Remove()

	// Remove inline styles from all elements except images
	doc.Find("[style]").Each(func(i int, s *goquery.Selection) {
		// Skip images - we want to keep their styles for responsive display
		if !s.Is("img") {
			s.RemoveAttr("style")
		}
	})

	// Remove other unnecessary attributes that might affect styling
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		// Skip images - already handled
		if !s.Is("img") {
			s.RemoveAttr("align")
			s.RemoveAttr("width")
			s.RemoveAttr("height")
			s.RemoveAttr("border")
			s.RemoveAttr("bgcolor")
			s.RemoveAttr("color")
		}
	})

	// Remove empty paragraphs
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		trimmedHTML := strings.TrimSpace(html)
		if trimmedHTML == "" || trimmedHTML == "&nbsp;" {
			s.Remove()
		}
	})

	// Get the HTML after initial cleaning
	initialCleanedHtml, _ := doc.Html()

	// Save initial cleaned HTML for debugging if needed
	if p.debug {
		if p.tempDir != "" {
			safeTitle := sanitizeFilename(contentTitle)
			debugFilePath := filepath.Join(p.tempDir, fmt.Sprintf("debug_%s_initial_cleaned.html", safeTitle))
			if err := os.WriteFile(debugFilePath, []byte(initialCleanedHtml), 0600); err != nil {
				slog.Error("Failed to save initial cleaned HTML debug file", "error", err)
			} else if logger.Debug {
				slog.Debug("Saved initial cleaned HTML", "title", contentTitle, "path", debugFilePath)
			}
		}
	}

	// Now apply go-readability as a final cleaning step
	// We'll wrap the content in a simple HTML structure to ensure readability processes it correctly
	wrappedHTML := fmt.Sprintf("<html><body>%s</body></html>", initialCleanedHtml)

	// Extract and preserve images before readability processing
	var images []struct {
		Src      string
		Alt      string
		Original string
	}

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		alt, _ := s.Attr("alt")
		html, _ := s.Parent().Html()

		images = append(images, struct {
			Src      string
			Alt      string
			Original string
		}{
			Src:      src,
			Alt:      alt,
			Original: html,
		})
	})

	// Use readability to simplify the HTML structure
	article, err := readability.FromReader(strings.NewReader(wrappedHTML), nil)

	var finalHtml string
	if err != nil {
		slog.Error("Failed to process with readability", "error", err)
		// If readability fails, use the result of initial cleaning
		finalHtml = initialCleanedHtml
	} else {
		// Get the content from readability
		finalHtml = article.Content

		// Check if images were preserved, if not, reinsert them
		readabilityDoc, err := goquery.NewDocumentFromReader(strings.NewReader(finalHtml))
		if err == nil {
			for _, img := range images {
				// Check if this image still exists in the readability output
				found := false
				readabilityDoc.Find("img").Each(func(i int, s *goquery.Selection) {
					src, _ := s.Attr("src")
					if src == img.Src {
						found = true
					}
				})

				// If not found, reinsert at the end
				if !found && img.Src != "" {
					if logger.Debug {
						slog.Debug("Reinserting missing image", "src", img.Src)
					}
					readabilityDoc.Find("body").AppendHtml(fmt.Sprintf("<p><img src=\"%s\" alt=\"%s\"></p>", img.Src, img.Alt))
				}
			}

			// Get the HTML with potentially reinserted images
			finalHtml, _ = readabilityDoc.Html()
		}

		// Save readability output for debugging if needed
		if p.debug {
			if p.tempDir != "" {
				safeTitle := sanitizeFilename(contentTitle)
				debugFilePath := filepath.Join(p.tempDir, fmt.Sprintf("debug_%s_readability.html", safeTitle))
				if err := os.WriteFile(debugFilePath, []byte(finalHtml), 0600); err != nil {
					slog.Error("Failed to save readability HTML debug file", "error", err)
				} else if logger.Debug {
					slog.Debug("Saved readability HTML", "title", contentTitle, "path", debugFilePath)
				}
			}
		}
	}

	// Clean up line breaks and spacing
	// Remove excessive whitespace
	re := regexp.MustCompile(`\s{2,}`)
	finalHtml = re.ReplaceAllString(finalHtml, " ")

	// Remove empty lines
	re = regexp.MustCompile(`(?m)^\s*$[\r\n]*`)
	finalHtml = re.ReplaceAllString(finalHtml, "")

	// Debug logging for output HTML
	if p.debug {
		if logger.Debug {
			slog.Debug("CleanHTML output", "title", contentTitle, "length", len(finalHtml))

			// Log a sample of the output HTML for debugging
			if len(finalHtml) > 100 {
				slog.Debug("CleanHTML output sample", "sample", finalHtml[:100])
			}
		}

		// Save debug file
		if p.tempDir != "" {
			safeTitle := sanitizeFilename(contentTitle)
			debugFilePath := filepath.Join(p.tempDir, fmt.Sprintf("debug_%s_cleaned.html", safeTitle))
			if err := os.WriteFile(debugFilePath, []byte(finalHtml), 0600); err != nil {
				slog.Error("Failed to save cleaned HTML debug file", "error", err)
			} else if logger.Debug {
				slog.Debug("Saved cleaned HTML", "title", contentTitle, "path", debugFilePath)
			}
		}
	}

	return finalHtml
}

// ProcessChapterContent creates properly formatted HTML for a chapter
func (p *HTMLProcessor) ProcessChapterContent(title string, content string) string {
	// Debug logging for chapter content
	if p.debug {
		if logger.Debug {
			slog.Debug("ProcessChapterContent called", "title", title, "content_length", len(content))
		}

		// Check if content is empty or very short
		if len(content) < 100 {
			slog.Warn("Content is very short or empty", "title", title, "content", content)

			// Try to use debug file content if available
			if p.tempDir != "" {
				safeTitle := sanitizeFilename(title)
				debugFilePath := filepath.Join(p.tempDir, fmt.Sprintf("debug_%s_final.html", safeTitle))
				if debugContent, err := os.ReadFile(debugFilePath); err == nil {
					slog.Info("Using debug file content", "title", title, "length", len(debugContent))
					content = string(debugContent)
				} else {
					slog.Error("Failed to read debug file", "title", title, "error", err)
				}
			}
		} else if logger.Debug {
			previewContent := content
			if len(content) > 100 {
				previewContent = content[:100]
			}
			slog.Debug("Content preview", "title", title, "preview", previewContent)
		}

		// Save debug file
		if p.tempDir != "" {
			safeTitle := sanitizeFilename(title)
			debugFilePath := filepath.Join(p.tempDir, fmt.Sprintf("debug_%s_processed.html", safeTitle))
			if err := os.WriteFile(debugFilePath, []byte(content), 0600); err != nil {
				slog.Error("Failed to save processed content debug file", "error", err)
			} else if logger.Debug {
				slog.Debug("Saved processed content", "title", title, "path", debugFilePath)
			}
		}
	}

	// Create chapter HTML with proper styling and minimal margins
	chapterHTML := fmt.Sprintf(`<html>
<head>
    <title>%s</title>
</head>
<body class="chapter" style="margin:0; padding:0;">
    <h2>%s</h2>
    %s
</body>
</html>`, title, title, content)

	// Debug logging for final chapter HTML
	if p.debug {
		if logger.Debug {
			slog.Debug("Final HTML", "title", title, "length", len(chapterHTML))
		}

		// Save final debug file
		if p.tempDir != "" {
			safeTitle := sanitizeFilename(title)
			debugFilePath := filepath.Join(p.tempDir, fmt.Sprintf("debug_%s_final.html", safeTitle))
			if err := os.WriteFile(debugFilePath, []byte(chapterHTML), 0600); err != nil {
				slog.Error("Failed to save final HTML debug file", "error", err)
			} else if logger.Debug {
				slog.Debug("Saved final HTML", "title", title, "path", debugFilePath)
			}
		}
	}

	return chapterHTML
}
