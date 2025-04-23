# System Patterns: SeireiTranslations EPUB Generator

## System Architecture
The SeireiTranslations EPUB Generator follows a modular, pipeline-based architecture that processes content through distinct stages from web scraping to EPUB generation.

### High-Level Architecture

```
[User Input] -> [Config Parser] -> [Web Scraper] -> [Content Processor] -> [Image Processor] -> [EPUB Generator] -> [Output File]
```

### Main Components
1. **Command-Line Interface & Configuration (cmd/seireitranslations-epub/app)**
   - Parses command-line arguments
   - Sets up configuration for all other components
   - Orchestrates the overall workflow

2. **Web Scraper (internal/scraper)**
   - Downloads HTML content from URLs
   - Uses extraction patterns to identify and extract relevant content
   - Applies initial cleanup of blog-specific elements

3. **Content Processor (internal/processor)**
   - Cleans HTML by removing inline styles
   - Standardizes formatting for EPUB compatibility
   - Structures content for proper chapter organization

4. **Image Processor (internal/processor)**
   - Identifies images within the content
   - Downloads image files
   - Processes images for inclusion in the EPUB

5. **Downloader (internal/downloader)**
   - Handles file downloads (images, cover)
   - Manages caching in debug mode

6. **EPUB Generator (internal/epub)**
   - Creates EPUB structure
   - Adds chapters, images, and metadata
   - Generates the final EPUB file

## Key Technical Decisions

### Extraction Pattern Approach
The system uses a pattern-based approach for content extraction, allowing it to adapt to variations in blog post structure:
- Multiple extraction patterns are attempted in sequence
- Debug mode saves intermediate extraction results for analysis
- Fallback patterns ensure content can be extracted even with blog changes

### Modular Component Design
- Each major function is encapsulated in its own package
- Components communicate through well-defined interfaces
- Enables easier testing, maintenance, and future extensions

### HTML Processing Pipeline
Content goes through a multi-stage processing pipeline:
1. Initial extraction of relevant content from blog HTML
2. Removal of blog-specific elements (navigation, ads, etc.)
3. Cleaning of HTML (removing inline styles, standardizing tags)
4. Processing of special elements (e.g., "Part X" paragraphs to h3 subtitles)
5. Final formatting for EPUB compatibility

### Debug Mode Implementation
- Temporary files preserved for inspection
- Intermediate extraction results saved
- Detailed logging of processing steps
- Caching of downloaded files to speed up repeated runs

### Content Extraction Strategy
1. Identify title/heading element as the starting point
2. Extract content that follows the title
3. Remove navigation elements, advertisements, and other non-content
4. Process and retain important formatting elements

## Component Relationships

### Data Flow
```
[URL List] -> [Config] -> [Scraper] -> [Raw HTML] -> [HTML Processor] 
                      |
                      v
[EPUB Generator] <- [Processed HTML] <- [Image Processor]
```

### Dependency Structure
- The app package depends on all other packages
- Scraper depends on logger but is independent of other components
- Processor depends on downloader for image processing
- EPUB generator is mostly independent but uses processed content

## Critical Implementation Paths

### Content Extraction
The most critical path is the content extraction logic in `internal/scraper/patterns.go`:
- Uses multiple strategies to identify the starting point of content
- Removes preceding elements to isolate relevant content
- Handles variations in blog post structure
- Provides detailed debug output when enabled

### HTML Cleaning
The HTML processing in `internal/processor/html.go` is essential for proper EPUB formatting:
- Removes inline styles that could disrupt EPUB rendering
- Standardizes HTML elements for consistent appearance
- Preserves important structural elements like headings

### Chapter Management
The chapter handling in `internal/epub/generator.go` is crucial for proper book organization:
- Manages chapter boundaries based on titles
- Handles multi-part chapters
- Ensures proper sequencing and navigation
