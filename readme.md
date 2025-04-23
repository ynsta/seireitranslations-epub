## Example Command

```bash
go run main.go --title "My Light Novel" --author "Author Name" --cover "https://example.com/cover.jpg" --output "mynovel.epub" --urls "urls.txt"
```# SeireiTranslations EPUB Generator

This Go program scrapes content from SeireiTranslations blog posts and packages them into an EPUB file for easier reading. It extracts content after the H4 heading, removes the last three center-aligned paragraphs, and cleans up inline HTML styles for better EPUB formatting.

## Features

- Scrapes multiple blog posts from SeireiTranslations based on a list of URLs
- Extracts only the relevant content from each page (after the H4 heading)
- Removes the last three center-aligned paragraphs (typically navigation/credits)
- Cleans all inline HTML styles for consistent EPUB formatting
- Adds proper chapter titles and organization
- Includes a custom cover image
- Applies consistent styling throughout the EPUB

## Installation

1. Make sure you have Go installed (version 1.18 or later recommended)
2. Clone this repository or download the source code
3. Install dependencies:

```bash
go mod download
```

## Usage

1. Create a text file containing a list of URLs to scrape, one per line, in the format `Chapter Name::URL` (see `urls.txt` for an example)
2. Run the program with the required parameters:

```bash
go run main.go --title "Novel Title" --author "Author Name" --cover "https://example.com/cover.jpg" --output "output.epub" --urls "urls.txt"
```

### Command Line Arguments

- `--title`: The title of the EPUB (required)
- `--author`: The author name (required)
- `--cover`: URL of the cover image (required)
- `--output`: Output EPUB filename (required)
- `--urls`: Path to a file containing the list of URLs to scrape (required)

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

This allows you to specify custom chapter titles for each URL.

## Build Executable

To build a standalone executable:

```bash
go build -o epub-generator
```

Then you can run it without needing Go installed:

```bash
./epub-generator --title "My Light Novel" --author "Author Name" --cover "https://example.com/cover.jpg" --output "mynovel.epub" --urls "urls.txt"
```

## Notes

- The program includes a small delay between requests to be respectful to the server
- URLs should be to specific chapter pages on the SeireiTranslations blog
- Chapter titles are extracted from the URL structure
