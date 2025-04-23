package processor

import (
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bmaupin/go-epub"
	"github.com/ynsta/seireitranslations-epub/internal/logger"
)

// ImageProcessor handles image processing for EPUB content
type ImageProcessor struct {
	downloader Downloader
	tempDir    string
	debug      bool
	epub       *epub.Epub
}

// Downloader interface defines methods needed for downloading files
type Downloader interface {
	DownloadFile(url string, filename string) ([]byte, error)
	SaveToFile(data []byte, filename string) (string, error)
}

// NewImageProcessor creates a new ImageProcessor
func NewImageProcessor(downloader Downloader, tempDir string, debug bool, epub *epub.Epub) *ImageProcessor {
	return &ImageProcessor{
		downloader: downloader,
		tempDir:    tempDir,
		debug:      debug,
		epub:       epub,
	}
}

// ProcessImages processes all images in the HTML content
func (p *ImageProcessor) ProcessImages(content string, pageURL string) (string, error) {
	// Create a document from the HTML content
	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("error parsing content HTML: %v", err)
	}

	// Process full-size images from links first
	contentDoc.Find("a").Each(func(i int, s *goquery.Selection) {
		// Check if this is an image link
		if s.Find("img").Length() > 0 {
			// Get the full-size image URL from the link
			fullImgSrc, exists := s.Attr("href")
			if !exists || fullImgSrc == "" {
				return
			}

			// Skip non-image links
			if !strings.Contains(fullImgSrc, ".jpg") && !strings.Contains(fullImgSrc, ".jpeg") &&
				!strings.Contains(fullImgSrc, ".png") && !strings.Contains(fullImgSrc, ".gif") {
				return
			}

			// Make sure we have a valid URL
			fullImgSrc = p.resolveURL(fullImgSrc, pageURL)

			// Generate a unique filename for the image
			imgExt := filepath.Ext(fullImgSrc)
			if imgExt == "" {
				imgExt = ".jpg" // Default extension
			}
			imgFilename := fmt.Sprintf("fullimg_%d_%d%s", time.Now().UnixNano(), i, imgExt)

			// Download the image
			if logger.Debug {
				slog.Info("Downloading full-size image", "url", fullImgSrc)
			}
			imgData, err := p.downloader.DownloadFile(fullImgSrc, imgFilename)
			if err != nil {
				slog.Warn("Error downloading full-size image", "error", err)
				return
			}

			// Don't proceed if we got no data
			if len(imgData) == 0 {
				slog.Warn("No image data received for full-size image, skipping")
				return
			}

			// Save image to a temporary file
			tempImgPath, err := p.downloader.SaveToFile(imgData, imgFilename)
			if err != nil {
				slog.Warn("Error saving full-size image", "error", err)
				return
			}

			// Add image to EPUB
			internalImgPath, err := p.epub.AddImage(tempImgPath, imgFilename)
			if err != nil {
				slog.Warn("Error adding full-size image to EPUB", "error", err)
				return
			}

			// Get the img tag inside the a tag
			img := s.Find("img")

			// Replace the img src with the internal EPUB path to the full-size image
			img.SetAttr("src", internalImgPath)

			// Remove unnecessary attributes
			img.RemoveAttr("border")
			img.RemoveAttr("data-original-height")
			img.RemoveAttr("data-original-width")
			img.RemoveAttr("height")
			img.RemoveAttr("width")

			// Add style for responsive image
			img.SetAttr("style", "max-width: 100%; height: auto;")

			// Unwrap the <a> tag by replacing it with just the <img>
			imgOuterHtml, _ := goquery.OuterHtml(img)
			s.ReplaceWithHtml(imgOuterHtml)
		}
	})

	// Process any remaining images
	contentDoc.Find("img").Each(func(i int, s *goquery.Selection) {
		// Skip if this image was already processed (has a path in the EPUB)
		src, _ := s.Attr("src")
		if strings.HasPrefix(src, "../") {
			return // Already processed
		}

		// Get the image URL
		imgSrc, exists := s.Attr("src")
		if !exists {
			return
		}

		// Make sure we have a valid URL (handle relative URLs)
		imgSrc = p.resolveURL(imgSrc, pageURL)

		// Skip image if URL is empty
		if imgSrc == "" {
			slog.Debug("Empty image URL, skipping")
			return
		}

		// Generate a unique filename for the image
		imgExt := filepath.Ext(imgSrc)
		if imgExt == "" {
			imgExt = ".jpg" // Default extension
		}
		imgFilename := fmt.Sprintf("image_%d_%d%s", time.Now().UnixNano(), i, imgExt)

		// Download the image
		if logger.Debug {
			slog.Info("Downloading image", "url", imgSrc)
		}
		imgData, err := p.downloader.DownloadFile(imgSrc, imgFilename)
		if err != nil {
			slog.Warn("Error downloading image", "error", err)
			return
		}

		// Don't proceed if we got no data
		if len(imgData) == 0 {
			slog.Warn("No image data received, skipping")
			return
		}

		// Save image to a temporary file
		tempImgPath, err := p.downloader.SaveToFile(imgData, imgFilename)
		if err != nil {
			slog.Warn("Error saving image", "error", err)
			return
		}

		// Add image to EPUB
		internalImgPath, err := p.epub.AddImage(tempImgPath, imgFilename)
		if err != nil {
			slog.Warn("Error adding image to EPUB", "error", err)
			return
		}

		// Replace the img src with the internal EPUB path
		s.SetAttr("src", internalImgPath)

		// Remove unnecessary attributes
		s.RemoveAttr("border")
		s.RemoveAttr("data-original-height")
		s.RemoveAttr("data-original-width")
		s.RemoveAttr("height")
		s.RemoveAttr("width")

		// Add style for responsive image
		s.SetAttr("style", "max-width: 100%; height: auto;")
	})

	// Get the processed HTML
	processedHTML, err := contentDoc.Html()
	if err != nil {
		return "", fmt.Errorf("error generating processed HTML: %v", err)
	}

	return processedHTML, nil
}

// resolveURL resolves a potentially relative URL against a base URL
func (p *ImageProcessor) resolveURL(imgSrc string, pageURL string) string {
	if !strings.HasPrefix(imgSrc, "http://") && !strings.HasPrefix(imgSrc, "https://") {
		// If it's a relative URL, try to resolve it against the page URL
		baseURL, err := url.Parse(pageURL)
		if err != nil {
			slog.Error("Error parsing base URL", "error", err)
			return imgSrc
		}

		relativeURL, err := url.Parse(imgSrc)
		if err != nil {
			slog.Error("Error parsing relative URL", "error", err)
			return imgSrc
		}

		absoluteURL := baseURL.ResolveReference(relativeURL)
		return absoluteURL.String()
	}
	return imgSrc
}
