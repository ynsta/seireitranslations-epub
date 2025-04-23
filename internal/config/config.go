package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ynsta/seireitranslations-epub/internal/logger"
)

// Config holds the application configuration
type Config struct {
	Title       string
	Author      string
	CoverURL    string
	OutputFile  string
	URLListFile string
	Debug       bool
	TempDir     string
}

// ParseCommandLine parses command-line arguments and returns a Config
func ParseCommandLine() (*Config, error) {
	cfg := &Config{}

	// Define command-line flags
	flag.StringVar(&cfg.Title, "title", "", "EPUB title (required)")
	flag.StringVar(&cfg.Author, "author", "", "Author name (required)")
	flag.StringVar(&cfg.CoverURL, "cover", "", "Cover image URL (required)")
	flag.StringVar(&cfg.OutputFile, "output", "", "Output EPUB filename (required)")
	flag.StringVar(&cfg.URLListFile, "urls", "", "File containing list of URLs to scrape (required)")
	flag.BoolVar(&cfg.Debug, "debug", false, "Enable debug mode: store temp files in current directory with .tmp suffix and skip cleanup")
	flag.Parse()

	// Validate required parameters
	if cfg.Title == "" || cfg.Author == "" || cfg.CoverURL == "" || cfg.OutputFile == "" || cfg.URLListFile == "" {
		flag.Usage()
		return nil, fmt.Errorf("missing required parameters")
	}

	// Set up temporary directory
	if cfg.Debug {
		// In debug mode, use current directory with output filename as base
		cfg.TempDir = cfg.OutputFile + ".tmp"
	} else {
		// Normal mode - use system temp directory
		cfg.TempDir = filepath.Join(os.TempDir(), fmt.Sprintf("epub_files_%d", time.Now().UnixNano()))
	}

	// Create the temporary directory - using 0750 permissions for better security
	if err := os.MkdirAll(cfg.TempDir, 0750); err != nil {
		return nil, fmt.Errorf("error creating temp directory: %v", err)
	}

	// Initialize the logger with the debug setting
	logger.Init(cfg.Debug)

	return cfg, nil
}

// Cleanup removes the temporary directory if not in debug mode
func (c *Config) Cleanup() {
	if !c.Debug {
		logger.Logger.Info("Cleaning up temporary directory", "dir", c.TempDir)
		if err := os.RemoveAll(c.TempDir); err != nil {
			logger.Logger.Error("Failed to clean up temporary directory", "dir", c.TempDir, "error", err)
		}
	} else {
		logger.Logger.Info("Debug mode: Temporary directory will not be cleaned up", "dir", c.TempDir)
	}
}
