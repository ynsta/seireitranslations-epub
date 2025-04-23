package epub

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmaupin/go-epub"
	"github.com/ynsta/seireitranslations-epub/internal/logger"
	"github.com/ynsta/seireitranslations-epub/pkg/utils"
)

// Generator handles EPUB file generation
type Generator struct {
	epub       *epub.Epub
	cssPath    string
	tempDir    string
	outputFile string
	debug      bool
}

// Config holds the configuration for the EPUB generator
type Config struct {
	Title      string
	Author     string
	CoverURL   string
	OutputFile string
	TempDir    string
	Debug      bool
}

// New creates a new EPUB generator
func New(config Config) *Generator {
	// Create a new EPUB
	e := epub.NewEpub(config.Title)
	e.SetAuthor(config.Author)

	return &Generator{
		epub:       e,
		tempDir:    config.TempDir,
		outputFile: config.OutputFile,
		debug:      config.Debug,
	}
}

// GetEpub returns the underlying EPUB object
func (g *Generator) GetEpub() *epub.Epub {
	return g.epub
}

// AddCover adds a cover image to the EPUB
func (g *Generator) AddCover(coverData []byte, coverURL string) error {
	// Determine the file extension from the URL
	coverFilename := "cover" + filepath.Ext(coverURL)

	// Save the cover image to a temporary file
	tempCoverPath := filepath.Join(g.tempDir, coverFilename)
	if err := os.WriteFile(tempCoverPath, coverData, 0644); err != nil {
		return fmt.Errorf("error saving cover image: %v", err)
	}

	// Add the cover image to the EPUB
	coverImagePath, err := g.epub.AddImage(tempCoverPath, coverFilename)
	if err != nil {
		return fmt.Errorf("error adding cover image to EPUB: %v", err)
	}

	// Set the cover in the EPUB
	g.epub.SetCover(coverImagePath, "")

	return nil
}

// AddCSS adds a CSS stylesheet to the EPUB
func (g *Generator) AddCSS(cssData []byte) error {
	// Create a temporary file to store the CSS
	tempCSSFile := filepath.Join(g.tempDir, "epub_styles.css")
	if err := os.WriteFile(tempCSSFile, cssData, 0644); err != nil {
		return fmt.Errorf("error writing temporary CSS file: %v", err)
	}

	// Add the CSS file to the EPUB
	cssPath, err := g.epub.AddCSS(tempCSSFile, "stylesheet.css")
	if err != nil {
		return fmt.Errorf("error adding CSS to EPUB: %v", err)
	}

	// Store the CSS path for later use
	g.cssPath = cssPath

	return nil
}

// AddAttributionChapter adds a chapter with attribution information and support links
func (g *Generator) AddAttributionChapter(title string, urlEntries []utils.URLEntry) error {
	// Create HTML content for the attribution chapter
	content := `<div class="attribution">
<h1>Attribution</h1>
<p>This e-book contains content translated by <strong>SeireiTranslations</strong>.</p>

<h2>Support the Translators</h2>
<p>If you enjoy this translation, please consider supporting the translators to help them continue their work:</p>
<ul>
<li><a href="https://ko-fi.com/seireitranslations">Support on Ko-Fi</a></li>
<li><a href="https://www.patreon.com/seireitl">Support on Patreon</a></li>
</ul>

<h2>Original Content Sources</h2>
<p>The content in this e-book was sourced from the following links:</p>
<ul>
`

	// Add each URL as a list item
	for _, entry := range urlEntries {
		content += fmt.Sprintf("<li><a href=\"%s\">%s</a></li>\n", entry.URL, entry.Title)
	}

	// Close the HTML tags
	content += `</ul>
</div>`

	// Add the attribution chapter to the EPUB
	_, err := g.epub.AddSection(content, title, "", g.cssPath)
	if err != nil {
		return fmt.Errorf("error adding attribution chapter to EPUB: %v", err)
	}

	return nil
}

// AddChapter adds a chapter to the EPUB
func (g *Generator) AddChapter(title string, content string) error {
	// Debug logging when debug mode is enabled
	if g.debug {
		if logger.Debug {
			slog.Debug("AddChapter called", "title", title, "content_length", len(content))
		}

		// Check if content is empty or very short
		if len(content) < 100 {
			slog.Warn("Content is very short or empty in AddChapter", "title", title, "content", content)
		} else if logger.Debug {
			previewContent := content
			if len(content) > 100 {
				previewContent = content[:100]
			}
			slog.Debug("Content in AddChapter preview", "title", title, "preview", previewContent)
		}

		// Save debug file in temp directory
		safeTitle := sanitizeFilename(title)
		debugFilePath := filepath.Join(g.tempDir, fmt.Sprintf("debug_%s_epub.html", safeTitle))
		if err := os.WriteFile(debugFilePath, []byte(content), 0644); err != nil {
			slog.Error("Failed to save debug file", "error", err)
		} else if logger.Debug {
			slog.Debug("Saved content to debug file", "title", title, "path", debugFilePath)
		}
	}

	// Add the chapter to the EPUB
	_, err := g.epub.AddSection(content, title, "", g.cssPath)
	if err != nil {
		return fmt.Errorf("error adding chapter to EPUB: %v", err)
	}

	return nil
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Write writes the EPUB file to disk
func (g *Generator) Write() error {
	// Write the EPUB file
	err := g.epub.Write(g.outputFile)
	if err != nil {
		return fmt.Errorf("error writing EPUB: %v", err)
	}

	slog.Info("Successfully created EPUB", "file", g.outputFile)
	return nil
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

// Chapter represents a chapter in the EPUB
type Chapter struct {
	Title   string
	Content strings.Builder
	Debug   bool
}

// NewChapter creates a new Chapter
func NewChapter(title string) *Chapter {
	return &Chapter{
		Title: title,
	}
}

// SetDebug sets the debug flag for this chapter
func (c *Chapter) SetDebug(debug bool) {
	c.Debug = debug
}

// AppendContent appends content to the chapter
func (c *Chapter) AppendContent(content string) {
	// Debug logging if debug is enabled
	if c.Debug {
		if logger.Debug {
			slog.Debug("AppendContent called",
				"chapter", c.Title,
				"content_length", len(content),
				"current_total", c.Content.Len())
		}

		// Check if content is empty or very short
		if len(content) < 100 {
			slog.Warn("Content being appended is very short", "chapter", c.Title, "content", content)
		} else if logger.Debug {
			previewContent := content
			if len(content) > 100 {
				previewContent = content[:100]
			}
			slog.Debug("Content being appended preview", "chapter", c.Title, "preview", previewContent)
		}
	}

	c.Content.WriteString(content)

	// Debug logging after append
	if c.Debug && logger.Debug {
		slog.Debug("Content appended", "chapter", c.Title, "total_length", c.Content.Len())
	}
}

// GetContent returns the chapter content
func (c *Chapter) GetContent() string {
	return c.Content.String()
}

// HasContent returns true if the chapter has content
func (c *Chapter) HasContent() bool {
	hasContent := c.Content.Len() > 0

	// Debug logging if debug is enabled
	if c.Debug && logger.Debug {
		slog.Debug("HasContent check",
			"chapter", c.Title,
			"content_length", c.Content.Len(),
			"has_content", hasContent)
	}

	return hasContent
}
