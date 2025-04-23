package epub

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmaupin/go-epub"
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

// AddChapter adds a chapter to the EPUB
func (g *Generator) AddChapter(title string, content string) error {
	// Add the chapter to the EPUB
	_, err := g.epub.AddSection(content, title, "", g.cssPath)
	if err != nil {
		return fmt.Errorf("error adding chapter to EPUB: %v", err)
	}

	return nil
}

// Write writes the EPUB file to disk
func (g *Generator) Write() error {
	// Write the EPUB file
	err := g.epub.Write(g.outputFile)
	if err != nil {
		return fmt.Errorf("error writing EPUB: %v", err)
	}

	fmt.Printf("Successfully created EPUB: %s\n", g.outputFile)
	return nil
}

// Chapter represents a chapter in the EPUB
type Chapter struct {
	Title   string
	Content strings.Builder
}

// NewChapter creates a new Chapter
func NewChapter(title string) *Chapter {
	return &Chapter{
		Title: title,
	}
}

// AppendContent appends content to the chapter
func (c *Chapter) AppendContent(content string) {
	c.Content.WriteString(content)
}

// GetContent returns the chapter content
func (c *Chapter) GetContent() string {
	return c.Content.String()
}

// HasContent returns true if the chapter has content
func (c *Chapter) HasContent() bool {
	return c.Content.Len() > 0
}
