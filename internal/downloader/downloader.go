package downloader

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Downloader handles file downloading with caching support
type Downloader struct {
	tempDir string
	debug   bool
}

// New creates a new Downloader instance
func New(tempDir string, debug bool) *Downloader {
	return &Downloader{
		tempDir: tempDir,
		debug:   debug,
	}
}

// DownloadFile downloads a file from a URL or uses cached version in debug mode
func (d *Downloader) DownloadFile(url string, filename string) ([]byte, error) {
	// Handle empty or invalid URLs
	if url == "" {
		return nil, fmt.Errorf("empty URL provided")
	}

	// If in debug mode and filename is provided, check if the file already exists
	if d.debug && filename != "" {
		tempFilePath := filepath.Join(d.tempDir, filename)
		if fileData, err := os.ReadFile(tempFilePath); err == nil {
			fmt.Printf("Using cached file: %s\n", tempFilePath)
			return fileData, nil
		}
	}

	// Create a client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make the request
	fmt.Printf("Downloading: %s\n", url)
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
	if d.debug && filename != "" {
		tempFilePath := filepath.Join(d.tempDir, filename)
		if err := os.WriteFile(tempFilePath, buf.Bytes(), 0644); err != nil {
			fmt.Printf("Warning: Could not cache file to %s: %v\n", tempFilePath, err)
		} else {
			fmt.Printf("Cached file to: %s\n", tempFilePath)
		}
	}

	return buf.Bytes(), nil
}

// SaveToFile saves data to a file in the temporary directory
func (d *Downloader) SaveToFile(data []byte, filename string) (string, error) {
	tempFilePath := filepath.Join(d.tempDir, filename)
	if err := os.WriteFile(tempFilePath, data, 0644); err != nil {
		return "", fmt.Errorf("error saving file to %s: %v", tempFilePath, err)
	}

	// Verify the file exists
	if _, err := os.Stat(tempFilePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file was not created at %s", tempFilePath)
	}

	return tempFilePath, nil
}
