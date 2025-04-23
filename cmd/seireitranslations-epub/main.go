package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/ynsta/seireitranslations-epub/internal/assets"
	"github.com/ynsta/seireitranslations-epub/internal/config"
	"github.com/ynsta/seireitranslations-epub/internal/downloader"
	"github.com/ynsta/seireitranslations-epub/internal/epub"
	"github.com/ynsta/seireitranslations-epub/internal/processor"
	"github.com/ynsta/seireitranslations-epub/internal/scraper"
	"github.com/ynsta/seireitranslations-epub/pkg/utils"
)

func main() {
	// Parse command-line arguments
	cfg, err := config.ParseCommandLine()
	if err != nil {
		log.Fatalf("Error parsing command-line arguments: %v", err)
	}

	// Clean up temporary directory at the end, but only in non-debug mode
	if !cfg.Debug {
		defer cfg.Cleanup()
	} else {
		log.Printf("Debug mode: Temporary directory will not be cleaned up")
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
		log.Fatalf("Error downloading cover image: %v", err)
	}

	if err := epubGen.AddCover(coverData, cfg.CoverURL); err != nil {
		log.Fatalf("Error adding cover image: %v", err)
	}

	// Add CSS stylesheet for consistent formatting using embedded file
	cssContent, err := assets.GetCSS()
	if err != nil {
		log.Fatalf("Error reading embedded CSS file: %v", err)
	}

	if err := epubGen.AddCSS(cssContent); err != nil {
		log.Fatalf("Error adding CSS: %v", err)
	}

	// Read the list of URLs
	urlEntries, err := utils.ReadURLList(cfg.URLListFile)
	if err != nil {
		log.Fatalf("Error reading URL list file: %v", err)
	}

	// Create a scraper
	s := scraper.New(cfg.Debug)

	// Create HTML processor
	htmlProc := processor.NewHTMLProcessor()

	// Create image processor
	imgProc := processor.NewImageProcessor(dl, cfg.TempDir, cfg.Debug, epubGen.GetEpub())

	// Process each URL
	var currentChapter *epub.Chapter
	var chapterIndex int = 1

	for i, entry := range urlEntries {
		log.Printf("Processing (%d/%d): %s - %s", i+1, len(urlEntries), entry.Title, entry.URL)

		// Download and process the page
		content, err := s.ExtractContent(entry.URL)
		if err != nil {
			log.Printf("Error processing %s: %v", entry.URL, err)
			continue
		}

		// Cleanup the HTML - remove inline styles, fix formatting
		cleanedHTML := htmlProc.CleanHTML(content.HTML)

		// Process images in the content
		processedHTML, err := imgProc.ProcessImages(cleanedHTML, entry.URL)
		if err != nil {
			log.Printf("Error processing images: %v", err)
			processedHTML = cleanedHTML // Fallback to cleaned HTML without image processing
		}

		// Check if we're continuing the same chapter or starting a new one
		if currentChapter != nil && entry.Title == currentChapter.Title {
			// Continuing the same chapter - append the content
			log.Printf("Continuing chapter: %s", entry.Title)
			currentChapter.AppendContent(processedHTML)
		} else {
			// If we have content from the previous chapter, add it to the EPUB
			if currentChapter != nil && currentChapter.HasContent() {
				// Create chapter HTML with proper styling
				chapterHTML := htmlProc.ProcessChapterContent(currentChapter.Title, currentChapter.GetContent())

				// Add the chapter to the EPUB
				if err := epubGen.AddChapter(currentChapter.Title, chapterHTML); err != nil {
					log.Printf("Error adding chapter to EPUB: %v", err)
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
			log.Printf("Error adding final chapter to EPUB: %v", err)
		}
	}

	// Write the EPUB file
	if err := epubGen.Write(); err != nil {
		log.Fatalf("Error writing EPUB: %v", err)
	}

	fmt.Printf("Successfully created EPUB: %s\n", cfg.OutputFile)
}
