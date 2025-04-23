package processor

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

	// Clean up line breaks and spacing
	cleanedHtml, _ := doc.Html()

	// Remove excessive whitespace
	re := regexp.MustCompile(`\s{2,}`)
	cleanedHtml = re.ReplaceAllString(cleanedHtml, " ")

	// Remove empty lines
	re = regexp.MustCompile(`(?m)^\s*$[\r\n]*`)
	cleanedHtml = re.ReplaceAllString(cleanedHtml, "")

	// Debug logging for output HTML
	if p.debug {
		if logger.Debug {
			slog.Debug("CleanHTML output", "title", contentTitle, "length", len(cleanedHtml))

			// Log a sample of the output HTML for debugging
			if len(cleanedHtml) > 100 {
				slog.Debug("CleanHTML output sample", "sample", cleanedHtml[:100])
			}
		}

		// Save debug file
		if p.tempDir != "" {
			safeTitle := sanitizeFilename(contentTitle)
			debugFilePath := filepath.Join(p.tempDir, fmt.Sprintf("debug_%s_cleaned.html", safeTitle))
			if err := os.WriteFile(debugFilePath, []byte(cleanedHtml), 0644); err != nil {
				slog.Error("Failed to save cleaned HTML debug file", "error", err)
			} else if logger.Debug {
				slog.Debug("Saved cleaned HTML", "title", contentTitle, "path", debugFilePath)
			}
		}
	}

	return cleanedHtml
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
			if err := os.WriteFile(debugFilePath, []byte(content), 0644); err != nil {
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
			if err := os.WriteFile(debugFilePath, []byte(chapterHTML), 0644); err != nil {
				slog.Error("Failed to save final HTML debug file", "error", err)
			} else if logger.Debug {
				slog.Debug("Saved final HTML", "title", title, "path", debugFilePath)
			}
		}
	}

	return chapterHTML
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
