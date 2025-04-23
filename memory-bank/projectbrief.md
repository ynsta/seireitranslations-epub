# Project Brief: SeireiTranslations EPUB Generator

## Project Purpose
A Go-based command-line tool that scrapes content from SeireiTranslations blog posts and packages them into an EPUB file format for easier reading, with proper formatting and organization.

## Core Requirements
1. Scrape web content from SeireiTranslations blog posts
2. Extract only the relevant novel content (excluding navigation, ads, etc.)
3. Process and clean HTML to ensure consistent formatting
4. Download and process images within the content
5. Package content into properly formatted EPUB files
6. Support custom covers, titles, and chapter organization
7. Provide debug mode for troubleshooting

## Project Scope
- EPUB generation for light novels/web novels from SeireiTranslations
- Command-line interface for configuration
- Support for processing multiple chapters from a list of URLs
- HTML/content cleaning and formatting specific to SeireiTranslations blog structure
- Image processing and inclusion in the EPUB
- Debug mode with cached downloads and temporary file preservation

## Out of Scope
- GUI interface
- Support for other blog/novel sites beyond SeireiTranslations
- Complex EPUB formatting beyond standard novel layout
- Content modification beyond cleaning/formatting

## Project Success Criteria
1. Successfully scrape and extract clean content from SeireiTranslations blog
2. Generate valid, well-formatted EPUB files with proper chapter organization
3. Support all required command-line arguments
4. Handle errors gracefully with appropriate user feedback
5. Performance: reasonable processing speed for multi-chapter novels
