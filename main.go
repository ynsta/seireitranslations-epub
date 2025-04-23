package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bmaupin/go-epub"
)

//go:embed styles.css
var cssFS embed.FS

func main() {
	// Define command-line flags
	title := flag.String("title", "", "EPUB title (required)")
	author := flag.String("author", "", "Author name (required)")
	coverURL := flag.String("cover", "", "Cover image URL (required)")
	outputFile := flag.String("output", "", "Output EPUB filename (required)")
	urlListFile := flag.String("urls", "", "File containing list of URLs to scrape (required)")
	debug := flag.Bool("debug", false, "Enable debug mode: store temp files in current directory with .tmp suffix and skip cleanup")
	flag.Parse()

	// Validate required parameters
	if *title == "" || *author == "" || *coverURL == "" || *outputFile == "" || *urlListFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Read the list of URLs
	urlsBytes, err := os.ReadFile(*urlListFile)
	if err != nil {
		log.Fatalf("Error reading URL list file: %v", err)
	}
	urls := strings.Split(strings.TrimSpace(string(urlsBytes)), "\n")

	// Create a new EPUB
	e := epub.NewEpub(*title)
	e.SetAuthor(*author)

	// Create a temporary directory for files
	var tempDir string
	if *debug {
		// In debug mode, use current directory with output filename as base
		tempDir = *outputFile + ".tmp"
		log.Printf("Debug mode enabled: Using temp directory %s", tempDir)
	} else {
		// Normal mode - use system temp directory
		tempDir = filepath.Join(os.TempDir(), fmt.Sprintf("epub_files_%d", time.Now().UnixNano()))
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Fatalf("Error creating temp directory: %v", err)
	}

	// Clean up temporary directory at the end, but only in non-debug mode
	if !*debug {
		defer func() {
			log.Printf("Cleaning up temporary directory: %s", tempDir)
			os.RemoveAll(tempDir)
		}()
	} else {
		log.Printf("Debug mode: Temporary directory will not be cleaned up")
	}

	// Add the cover image
	coverFilename := "cover" + filepath.Ext(*coverURL)
	coverData, err := downloadFile(*coverURL, tempDir, coverFilename, *debug)
	if err != nil {
		log.Fatalf("Error downloading cover image: %v", err)
	}

	tempCoverPath := filepath.Join(tempDir, "cover"+filepath.Ext(*coverURL))
	if err := os.WriteFile(tempCoverPath, coverData, 0644); err != nil {
		log.Fatalf("Error saving cover image: %v", err)
	}
	coverImagePath, err := e.AddImage(tempCoverPath, "cover"+filepath.Ext(*coverURL))
	if err != nil {
		log.Fatalf("Error adding cover image to EPUB: %v", err)
	}
	e.SetCover(coverImagePath, "")

	// Add CSS stylesheet for consistent formatting using embedded file
	cssContent, err := cssFS.ReadFile("styles.css")
	if err != nil {
		log.Fatalf("Error reading embedded CSS file: %v", err)
	}

	// Create a temporary file to store the CSS
	tempCSSFile := filepath.Join(tempDir, "epub_styles.css")
	if err := os.WriteFile(tempCSSFile, cssContent, 0644); err != nil {
		log.Fatalf("Error writing temporary CSS file: %v", err)
	}

	// Add the CSS file to the EPUB
	cssPath, err := e.AddCSS(tempCSSFile, "stylesheet.css")
	if err != nil {
		log.Fatalf("Error adding CSS to EPUB: %v", err)
	}

	// Process each URL
	var currentChapterTitle string
	var currentChapterContent strings.Builder
	var chapterIndex int = 1

	for i, line := range urls {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split line into chapter name and URL
		parts := strings.Split(line, "::")
		if len(parts) != 2 {
			log.Printf("Invalid format for line %d: %s (expected 'Chapter Name::URL')", i+1, line)
			continue
		}

		chapterTitle := strings.TrimSpace(parts[0])
		pageURL := strings.TrimSpace(parts[1])

		log.Printf("Processing (%d/%d): %s - %s", i+1, len(urls), chapterTitle, pageURL)

		// Download and process the page
		content, err := extractContent(pageURL, e, tempDir, *debug)
		if err != nil {
			log.Printf("Error processing %s: %v", pageURL, err)
			continue
		}

		// Cleanup the HTML - remove inline styles, fix formatting
		content = cleanHTML(content)

		// Check if we're continuing the same chapter or starting a new one
		if chapterTitle == currentChapterTitle && currentChapterTitle != "" {
			// Continuing the same chapter - append the content
			log.Printf("Continuing chapter: %s", chapterTitle)
			currentChapterContent.WriteString(content)
		} else {
			// If we have content from the previous chapter, add it to the EPUB
			if currentChapterContent.Len() > 0 {
				// Create chapter HTML with proper styling and minimal margins
				chapterHTML := fmt.Sprintf(`<html>
<head>
    <title>%s</title>
</head>
<body class="chapter" style="margin:0; padding:0;">
    <h2>%s</h2>
    %s
</body>
</html>`, currentChapterTitle, currentChapterTitle, currentChapterContent.String())

				// Add the chapter to the EPUB
				_, err = e.AddSection(chapterHTML, currentChapterTitle, "", cssPath)
				if err != nil {
					log.Printf("Error adding chapter to EPUB: %v", err)
				}
			}

			// Start a new chapter
			currentChapterTitle = chapterTitle
			currentChapterContent.Reset()
			currentChapterContent.WriteString(content)
			chapterIndex++
		}

		// Small delay to be nice to the server
		time.Sleep(500 * time.Millisecond)
	}

	// Don't forget to add the last chapter if there is one
	if currentChapterContent.Len() > 0 {
		// Create chapter HTML with proper styling and minimal margins
		chapterHTML := fmt.Sprintf(`<html>
<head>
    <title>%s</title>
</head>
<body class="chapter" style="margin:0; padding:0;">
    <h2>%s</h2>
    %s
</body>
</html>`, currentChapterTitle, currentChapterTitle, currentChapterContent.String())

		// Add the chapter to the EPUB
		_, err = e.AddSection(chapterHTML, currentChapterTitle, "", cssPath)
		if err != nil {
			log.Printf("Error adding final chapter to EPUB: %v", err)
		}
	}

	// Write the EPUB file
	err = e.Write(*outputFile)
	if err != nil {
		log.Fatalf("Error writing EPUB: %v", err)
	}

	log.Printf("Successfully created EPUB: %s", *outputFile)
}

// extractContent downloads a page and extracts content after the h4 element
func extractContent(pageURL string, e *epub.Epub, tempDir string, debug bool) (string, error) {
	// Get the page
	resp, err := http.Get(pageURL)
	if err != nil {
		return "", fmt.Errorf("error fetching URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP status code: %d", resp.StatusCode)
	}

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error parsing HTML: %v", err)
	}

	// Find the h4 element
	var content string
	found := false

	// Try different patterns for finding the h4 element
	// Pattern 1: .post-body > div > span > div > h4
	doc.Find(".post-body > div > span > div > h4").Each(func(i int, s *goquery.Selection) {
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
		log.Printf("Found content using pattern 1: .post-body > div > span > div > h4")
	})

	// If not found, try Pattern 2: .post-body > div > span > h4
	if !found {
		doc.Find(".post-body > div > span > h4").Each(func(i int, s *goquery.Selection) {
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
			log.Printf("Found content using pattern 2: .post-body > div > span > h4")
		})
	}

	// If not found, try Pattern 3: div.separator > div > span > h4
	if !found {
		doc.Find("div.separator > div > span > h4").Each(func(i int, s *goquery.Selection) {
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
				log.Printf("Found content using pattern 3: div.separator > div > span > h4 with separate content div")
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
			log.Printf("Found content using pattern 3 fallback: div.separator > div > span > h4 (parent container)")
		})
	}

	// If still not found, try a more general approach with any h4 in the post-body
	if !found {
		doc.Find(".post-body h4").Each(func(i int, s *goquery.Selection) {
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
			log.Printf("Found content using fallback pattern: .post-body h4")
		})
	}

	// Create a more comprehensive approach to handle the case when h4 is not found
	if !found {
		log.Printf("Warning: No h4 element found for content marker. Attempting alternative extraction method.")

		// Try to extract the main content by focusing on the core post content
		mainContent := doc.Find(".post-body")
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
						log.Printf("Used alternative extraction method")
					}
				}
			}
		}
	}

	if !found {
		return "", fmt.Errorf("could not find content in the blog post")
	}

	// Create a document from the extracted HTML to process it
	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("error parsing content HTML: %v", err)
	}

	// Remove the h4 element since it's redundant with the chapter title
	contentDoc.Find("h4").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	// Remove blog URL entries (seireitranslations.blogspot.com)
	contentDoc.Find("p, div").Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		htmlLower := strings.ToLower(html)

		if strings.Contains(htmlLower, "seireitranslations.blogspot.com") {
			s.Remove()
			log.Printf("Removed blog URL element")
		}
	})

	// Convert "Part X" paragraphs to h3 subtitles
	contentDoc.Find("p span[style*='font-weight: 800'], p span[style*='font-weight:800']").Each(func(i int, s *goquery.Selection) {
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
			if !strings.HasPrefix(fullImgSrc, "http://") && !strings.HasPrefix(fullImgSrc, "https://") {
				// If it's a relative URL, try to resolve it against the page URL
				baseURL, err := url.Parse(pageURL)
				if err != nil {
					log.Printf("Error parsing base URL: %v", err)
					return
				}

				relativeURL, err := url.Parse(fullImgSrc)
				if err != nil {
					log.Printf("Error parsing relative URL: %v", err)
					return
				}

				absoluteURL := baseURL.ResolveReference(relativeURL)
				fullImgSrc = absoluteURL.String()
			}

			// Generate a unique filename for the image
			imgExt := filepath.Ext(fullImgSrc)
			if imgExt == "" {
				imgExt = ".jpg" // Default extension
			}
			imgFilename := fmt.Sprintf("fullimg_%d_%d%s", time.Now().UnixNano(), i, imgExt)

			// Download the image
			log.Printf("Downloading full-size image: %s", fullImgSrc)
			imgData, err := downloadFile(fullImgSrc, tempDir, imgFilename, debug)
			if err != nil {
				log.Printf("Error downloading full-size image: %v", err)
				return
			}

			// Don't proceed if we got no data
			if len(imgData) == 0 {
				log.Printf("No image data received for full-size image, skipping")
				return
			}

			// Save image to a temporary file
			tempImgPath := filepath.Join(tempDir, imgFilename)
			if err := os.WriteFile(tempImgPath, imgData, 0644); err != nil {
				log.Printf("Error saving full-size image: %v", err)
				return
			}

			// Verify the file exists
			if _, err := os.Stat(tempImgPath); os.IsNotExist(err) {
				log.Printf("Full-size image file was not created")
				return
			}

			// Add image to EPUB
			internalImgPath, err := e.AddImage(tempImgPath, imgFilename)
			if err != nil {
				log.Printf("Error adding full-size image to EPUB: %v", err)
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
		if !strings.HasPrefix(imgSrc, "http://") && !strings.HasPrefix(imgSrc, "https://") {
			// If it's a relative URL, try to resolve it against the page URL
			baseURL, err := url.Parse(pageURL)
			if err != nil {
				log.Printf("Error parsing base URL: %v", err)
				return
			}

			relativeURL, err := url.Parse(imgSrc)
			if err != nil {
				log.Printf("Error parsing relative URL: %v", err)
				return
			}

			absoluteURL := baseURL.ResolveReference(relativeURL)
			imgSrc = absoluteURL.String()
		}

		// Skip image if URL is empty
		if imgSrc == "" {
			log.Printf("Empty image URL, skipping")
			return
		}

		// Generate a unique filename for the image
		imgExt := filepath.Ext(imgSrc)
		if imgExt == "" {
			imgExt = ".jpg" // Default extension
		}
		imgFilename := fmt.Sprintf("image_%d_%d%s", time.Now().UnixNano(), i, imgExt)

		// Download the image
		log.Printf("Downloading image: %s", imgSrc)
		imgData, err := downloadFile(imgSrc, tempDir, imgFilename, debug)
		if err != nil {
			log.Printf("Error downloading image: %v", err)
			return
		}

		// Don't proceed if we got no data
		if len(imgData) == 0 {
			log.Printf("No image data received, skipping")
			return
		}

		// Save image to a temporary file
		tempImgPath := filepath.Join(tempDir, imgFilename)
		if err := os.WriteFile(tempImgPath, imgData, 0644); err != nil {
			log.Printf("Error saving image to %s: %v", tempImgPath, err)
			return
		}

		// Verify the file exists before adding to EPUB
		if _, err := os.Stat(tempImgPath); os.IsNotExist(err) {
			log.Printf("Image file was not created at %s", tempImgPath)
			return
		}

		// Add image to EPUB
		internalImgPath, err := e.AddImage(tempImgPath, imgFilename)
		if err != nil {
			log.Printf("Error adding image to EPUB: %v", err)
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

	// Find and remove specific navigation or Patreon-related paragraphs
	contentDoc.Find("p[style*='text-align: center']").Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		htmlLower := strings.ToLower(html)

		// Remove paragraphs containing Patreon links
		if strings.Contains(htmlLower, "patreon") {
			s.Remove()
			log.Printf("Removed Patreon link paragraph")
			return
		}

		// Remove navigation paragraphs (Previous | Table of Contents | Next)
		if (strings.Contains(htmlLower, "previous") && strings.Contains(htmlLower, "next")) ||
			(strings.Contains(htmlLower, "previous") && strings.Contains(htmlLower, "table of contents")) ||
			(strings.Contains(htmlLower, "next") && strings.Contains(htmlLower, "table of contents")) {
			s.Remove()
			log.Printf("Removed navigation paragraph")
			return
		}
	})

	// Get the processed HTML
	processedHTML, err := contentDoc.Html()
	if err != nil {
		return "", fmt.Errorf("error generating processed HTML: %v", err)
	}

	return processedHTML, nil
}

// cleanHTML removes inline styles and other unnecessary attributes
func cleanHTML(html string) string {
	// Create a document to work with
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

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
	html, _ = doc.Html()

	// Remove excessive whitespace
	re := regexp.MustCompile(`\s{2,}`)
	html = re.ReplaceAllString(html, " ")

	// Remove empty lines
	re = regexp.MustCompile(`(?m)^\s*$[\r\n]*`)
	html = re.ReplaceAllString(html, "")

	return html
}

// downloadFile downloads a file from a URL or uses cached version in debug mode
func downloadFile(url string, tempDir string, filename string, debug bool) ([]byte, error) {
	// Handle empty or invalid URLs
	if url == "" {
		return nil, fmt.Errorf("empty URL provided")
	}

	// If in debug mode and filename is provided, check if the file already exists
	if debug && filename != "" {
		tempFilePath := filepath.Join(tempDir, filename)
		if fileData, err := os.ReadFile(tempFilePath); err == nil {
			log.Printf("Using cached file: %s", tempFilePath)
			return fileData, nil
		}
	}

	// Create a client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make the request
	log.Printf("Downloading: %s", url)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP status code: %d", resp.StatusCode)
	}

	// Read the response body
	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return nil, err
	}

	// Check if we actually got any data
	if buf.Len() == 0 {
		return nil, fmt.Errorf("zero bytes received")
	}

	// If in debug mode and filename is provided, save the file for future use
	if debug && filename != "" {
		tempFilePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(tempFilePath, buf.Bytes(), 0644); err != nil {
			log.Printf("Warning: Could not cache file to %s: %v", tempFilePath, err)
		} else {
			log.Printf("Cached file to: %s", tempFilePath)
		}
	}

	return buf.Bytes(), nil
}
