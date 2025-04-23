# SeireiTranslations EPUB Generator

This Go program scrapes content from SeireiTranslations blog posts and packages them into an EPUB file for easier reading. It uses multiple extraction patterns to identify and extract relevant content, processes images, cleans up inline HTML styles, and organizes chapters into a properly formatted EPUB file.

## Features

- Scrapes multiple blog posts from SeireiTranslations based on a list of URLs
- Uses multiple extraction patterns to identify and extract relevant content
- Handles variations in blog post structure with fallback extraction methods
- Processes and includes images from the content
- Cleans all inline HTML styles for consistent EPUB formatting
- Adds proper chapter titles and organization, including handling multi-part chapters
- Includes a custom cover image
- Applies consistent styling throughout the EPUB
- Adds an attribution chapter with links to support the translators

## Project Structure

The codebase is organized into a modular structure:

- `cmd/seireitranslations-epub/`: Application entry point
  - `app/`: Application logic and orchestration
- `internal/`: Internal packages
  - `assets/`: Embedded assets (CSS)
  - `config/`: Configuration handling
  - `downloader/`: File downloading functionality
  - `epub/`: EPUB generation
  - `logger/`: Logging utilities
  - `processor/`: HTML and image processing
  - `scraper/`: Web scraping functionality
- `pkg/`: Potentially reusable packages
  - `utils/`: Utility functions

## Installation

1. Make sure you have Go installed (version 1.24.2 or later recommended)
2. Clone this repository or download the source code
3. Install dependencies:

```bash
go mod download
```

## Usage

1. Create a text file containing a list of URLs to scrape, one per line, in the format `Chapter Name::URL` (see `urls.txt` for an example)
2. Run the program with the required parameters:

```bash
go run ./cmd/seireitranslations-epub --title "Novel Title" --author "Author Name" --cover "https://example.com/cover.jpg" --output "output.epub" --urls "urls.txt"
```

### Command Line Arguments

- `--title`: The title of the EPUB (required)
- `--author`: The author name (required)
- `--cover`: URL of the cover image (required)
- `--output`: Output EPUB filename (required)
- `--urls`: Path to a file containing the list of URLs to scrape (required)
- `--debug`: Enable debug mode (optional)

### Debug Mode

When the `--debug` flag is enabled, the program will:

1. Store temporary files in the current directory with the output filename + `.tmp` suffix
2. Not clean up the temporary directory after completion
3. Cache downloaded files for reuse in subsequent runs
4. Save intermediate extraction results for analysis
5. Provide detailed logging of processing steps
6. Save debug files for each extraction pattern attempt

This is useful for:
- Debugging issues with content extraction
- Examining the intermediate files and extraction results
- Speeding up repeated runs by caching downloaded content
- Troubleshooting when specific blog posts don't extract correctly

Example usage with debug mode:

```bash
./seireitranslations-epub --title "Novel Title" --author "Author Name" --cover "https://example.com/cover.jpg" --output "output.epub" --urls "urls.txt" --debug
```

## Example URLs File Format

Each line in the URLs file should follow this format:
```
Chapter Name::URL
```

For example:
```
Chapter 1: The Beginning::https://seireitranslations.blogspot.com/2023/08/chapter-1.html
Chapter 2: New Horizons::https://seireitranslations.blogspot.com/2023/08/chapter-2.html
```

This allows you to specify custom chapter titles for each URL. For multi-part chapters, use the same chapter title for multiple URLs, and they will be combined into a single chapter in the EPUB:

```
Chapter 1: The Beginning::https://seireitranslations.blogspot.com/2023/08/chapter-1-part1.html
Chapter 1: The Beginning::https://seireitranslations.blogspot.com/2023/08/chapter-1-part2.html
```

## Build Executable

To build a standalone executable:

```bash
go build .
```

Then you can run it without needing Go installed:

```bash
./seireitranslations-epub --title "My Light Novel" --author "Author Name" --cover "https://example.com/cover.jpg" --output "mynovel.epub" --urls "urls.txt"
```

## Content Extraction

The program uses a sophisticated multi-pattern approach to extract content:

1. **Primary Pattern**: Looks for heading elements (h4) or centered paragraphs to identify the start of content
2. **Advanced Pattern**: Uses multiple selector combinations to handle variations in blog formatting
3. **Fallback Pattern**: Extracts from the main content area as a last resort if other patterns fail

This approach ensures robust content extraction even with variations in blog post structure.

## Attribution Chapter

The program automatically adds an attribution chapter as the first chapter in each generated EPUB, which includes:

- Credit to SeireiTranslations for the translations
- Links to support the translators via Ko-Fi and Patreon
- A list of all source URLs used in the EPUB

## Notes

- The program includes a small delay between requests to be respectful to the server
- URLs should be to specific chapter pages on the SeireiTranslations blog
- Images within the content are downloaded and included in the EPUB
- Special handling for "Part X" sections formats them as subtitles in the EPUB

## Example Command

```bash
go run ./cmd/seireitranslations-epub --title "My Light Novel" --author "Author Name" --cover "https://example.com/cover.jpg" --output "mynovel.epub" --urls "urls.txt"
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
