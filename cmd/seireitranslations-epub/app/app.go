package app

import (
	"log/slog"
	"path/filepath"

	"github.com/ynsta/seireitranslations-epub/internal/assets"
	"github.com/ynsta/seireitranslations-epub/internal/config"
	"github.com/ynsta/seireitranslations-epub/internal/downloader"
	"github.com/ynsta/seireitranslations-epub/internal/epub"
	"github.com/ynsta/seireitranslations-epub/internal/processor"
	"github.com/ynsta/seireitranslations-epub/internal/scraper"
	"github.com/ynsta/seireitranslations-epub/pkg/utils"
)

// Execute runs the main program logic and returns an exit code
func Execute() int {
	// Parse command-line arguments
	cfg, err := config.ParseCommandLine()
	if err != nil {
		slog.Error("Error parsing command-line arguments", "error", err)
		return 1
	}

	// Clean up temporary directory at the end, but only in non-debug mode
	if !cfg.Debug {
		defer cfg.Cleanup()
	} else {
		slog.Debug("Debug mode: Temporary directory will not be cleaned up")
	}

	// Create a downloader
	dl := downloader.New(cfg.TempDir, cfg.Debug)

	// Create an EPUB generator
	epubGen := epub.New(epub.Config{
		Title:      cfg.Title,
		Author:     cfg.Author,
		CoverURL:   cfg.CoverURL,
		OutputFile: cfg.OutputFile,
		TempDir:    cfg.TempDir,
		Debug:      cfg.Debug,
	})

	// Download and add the cover image
	coverData, err := dl.DownloadFile(cfg.CoverURL, "cover"+filepath.Ext(cfg.CoverURL))
	if err != nil {
		slog.Error("Error downloading cover image", "error", err)
		return 1
	}

	if err := epubGen.AddCover(coverData, cfg.CoverURL); err != nil {
		slog.Error("Error adding cover image", "error", err)
		return 1
	}

	// Add CSS stylesheet for consistent formatting using embedded file
	cssContent, err := assets.GetCSS()
	if err != nil {
		slog.Error("Error reading embedded CSS file", "error", err)
		return 1
	}

	if err := epubGen.AddCSS(cssContent); err != nil {
		slog.Error("Error adding CSS", "error", err)
		return 1
	}

	// Read the list of URLs
	urlEntries, err := utils.ReadURLList(cfg.URLListFile)
	if err != nil {
		slog.Error("Error reading URL list file", "error", err)
		return 1
	}

	// Add attribution chapter as the first chapter
	slog.Info("Adding attribution chapter")
	if err := epubGen.AddAttributionChapter("Attribution and Sources", urlEntries); err != nil {
		slog.Warn("Error adding attribution chapter", "error", err)
	}

	// Create a scraper
	s := scraper.New(cfg.TempDir, cfg.Debug)

	// Create HTML processor
	htmlProc := processor.NewHTMLProcessor()
	htmlProc.SetDebug(cfg.Debug)
	htmlProc.SetTempDir(cfg.TempDir)

	// Create image processor
	imgProc := processor.NewImageProcessor(dl, cfg.TempDir, cfg.Debug, epubGen.GetEpub())

	// Process each URL
	var currentChapter *epub.Chapter
	var chapterIndex int = 1

	for i, entry := range urlEntries {
		slog.Info("Processing URL", "index", i+1, "total", len(urlEntries), "title", entry.Title, "url", entry.URL)

		// Download and process the page
		content, err := s.ExtractContent(entry.URL, i)
		if err != nil {
			slog.Warn("Error processing URL", "url", entry.URL, "error", err)
			continue
		}

		// Cleanup the HTML - remove inline styles, fix formatting
		cleanedHTML := htmlProc.CleanHTML(content.HTML, entry.Title)

		// Process images in the content
		processedHTML, err := imgProc.ProcessImages(cleanedHTML, entry.URL)
		if err != nil {
			slog.Warn("Error processing images", "error", err)
			processedHTML = cleanedHTML // Fallback to cleaned HTML without image processing
		}

		// Check if we're continuing the same chapter or starting a new one
		if currentChapter != nil && entry.Title == currentChapter.Title {
			// Continuing the same chapter - append the content
			slog.Info("Continuing chapter", "title", entry.Title)
			currentChapter.AppendContent(processedHTML)
		} else {
			// If we have content from the previous chapter, add it to the EPUB
			if currentChapter != nil && currentChapter.HasContent() {
				// Create chapter HTML with proper styling
				chapterHTML := htmlProc.ProcessChapterContent(currentChapter.Title, currentChapter.GetContent())

				// Add the chapter to the EPUB
				if err := epubGen.AddChapter(currentChapter.Title, chapterHTML); err != nil {
					slog.Warn("Error adding chapter to EPUB", "title", currentChapter.Title, "error", err)
				}
			}

			// Start a new chapter
			currentChapter = epub.NewChapter(entry.Title)
			currentChapter.AppendContent(processedHTML)
			chapterIndex++
		}
	}

	// Don't forget to add the last chapter if there is one
	if currentChapter != nil && currentChapter.HasContent() {
		// Create chapter HTML with proper styling
		chapterHTML := htmlProc.ProcessChapterContent(currentChapter.Title, currentChapter.GetContent())

		// Add the chapter to the EPUB
		if err := epubGen.AddChapter(currentChapter.Title, chapterHTML); err != nil {
			slog.Warn("Error adding final chapter to EPUB", "title", currentChapter.Title, "error", err)
		}
	}

	// Write the EPUB file
	if err := epubGen.Write(); err != nil {
		slog.Error("Error writing EPUB", "error", err)
		return 1
	}

	slog.Info("Successfully created EPUB", "file", cfg.OutputFile)
	return 0
}
