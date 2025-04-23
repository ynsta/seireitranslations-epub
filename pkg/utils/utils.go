package utils

import (
	"fmt"
	"os"
	"strings"
)

// ReadURLList reads a file containing a list of URLs in the format "Title::URL"
func ReadURLList(filename string) ([]URLEntry, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading URL list file: %v", err)
	}

	// Split into lines
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	// Parse each line
	var entries []URLEntry
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split line into chapter name and URL
		parts := strings.Split(line, "::")
		if len(parts) != 2 {
			fmt.Printf("Invalid format for line %d: %s (expected 'Chapter Name::URL')\n", i+1, line)
			continue
		}

		entries = append(entries, URLEntry{
			Title: strings.TrimSpace(parts[0]),
			URL:   strings.TrimSpace(parts[1]),
		})
	}

	return entries, nil
}

// URLEntry represents a single entry in the URL list
type URLEntry struct {
	Title string
	URL   string
}

// GroupURLsByChapter groups URL entries by chapter title
func GroupURLsByChapter(entries []URLEntry) map[string][]URLEntry {
	chapters := make(map[string][]URLEntry)

	for _, entry := range entries {
		chapters[entry.Title] = append(chapters[entry.Title], entry)
	}

	return chapters
}
