# Technical Context: SeireiTranslations EPUB Generator

## Technologies Used

### Primary Language
- **Go** (version 1.24.2): A modern, compiled programming language with strong typing, concurrency support, and a robust standard library.

### Key Dependencies
1. **github.com/PuerkitoBio/goquery** (v1.10.3)
   - HTML parsing and manipulation library for Go
   - Implements jQuery-like functionality for HTML documents
   - Used for extracting and processing content from blog posts

2. **github.com/bmaupin/go-epub** (v1.1.0)
   - EPUB generation library for Go
   - Handles creating valid EPUB structures
   - Manages EPUB metadata, content files, and navigation

3. **Indirect Dependencies**
   - **github.com/andybalholm/cascadia**: CSS selector parsing for goquery
   - **github.com/gabriel-vasile/mimetype**: MIME type detection
   - **github.com/gofrs/uuid**: UUID generation for EPUB components
   - **github.com/vincent-petithory/dataurl**: Data URL handling for embedded content
   - **golang.org/x/net**: Network functionality and HTML utilities

## Development Setup

### Environment Requirements
- Go 1.24.2 or later
- Standard Go toolchain (go build, go run, etc.)
- No special build tools or databases required

### Project Structure
- **cmd/**: Application entry points
  - **seireitranslations-epub/**: Main command-line application
    - **app/**: Application logic and orchestration
- **internal/**: Internal packages not intended for external use
  - **assets/**: Embedded static assets (CSS)
  - **config/**: Configuration handling
  - **downloader/**: File downloading functionality
  - **epub/**: EPUB generation logic
  - **logger/**: Logging utilities
  - **processor/**: HTML and image processing
  - **scraper/**: Web scraping functionality
- **pkg/**: Potentially reusable packages
  - **utils/**: Utility functions

### Build Process
- Standard Go build process
- No special pre-processing or build steps required
- Uses Go modules for dependency management

### Development Commands
```bash
# Build the application
go build .

# Run the application with required parameters
go run ./cmd/seireitranslations-epub --title "Novel Title" --author "Author Name" --cover "https://example.com/cover.jpg" --output "output.epub" --urls "urls.txt"

# Run with debug mode
go run ./cmd/seireitranslations-epub --title "Novel Title" --author "Author Name" --cover "https://example.com/cover.jpg" --output "output.epub" --urls "urls.txt" --debug
```

## Technical Constraints

### Platform Compatibility
- Cross-platform (Windows, macOS, Linux)
- Command-line interface only
- No GUI components

### Performance Considerations
- Sequential processing of URLs with small delay between requests
- Memory usage depends on the size of the novel and images
- Temporary directory used for intermediate files
- Debug mode increases disk usage due to preserved temporary files

### Limitations
- Specifically designed for SeireiTranslations blog structure
- May require updates if blog HTML structure changes significantly
- EPUB formatting limited to standard novel layout
- No support for complex interactive EPUB features

## Tools and Utilities

### Logging
- Uses Go's standard log/slog package for structured logging
- Debug mode enables detailed logging of processing steps
- Logs progress, warnings, and errors to console

### File Management
- Temporary directory for intermediate files
- Automatic cleanup in normal mode
- Preserved files in debug mode for inspection

### HTML Processing
- goquery for HTML parsing and manipulation
- Custom HTML cleaning functions for EPUB compatibility
- Special handling for blog-specific elements

### Image Handling
- Downloads and processes images found in content
- Embeds images in EPUB
- Handles various image formats

## Dependency Management

### Go Modules
- Uses Go modules for dependency management
- Dependencies specified in go.mod file
- No vendoring of dependencies

### External Libraries Purpose
- **goquery**: HTML parsing and manipulation (critical for content extraction)
- **go-epub**: EPUB file generation (handles EPUB specification compliance)
- **mimetype**: File type detection for images and other content
- **uuid**: Generating unique identifiers for EPUB components
- **dataurl**: Handling data URLs in HTML content
