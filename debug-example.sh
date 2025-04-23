#!/bin/bash
# Example of using the debug mode

# Run with debug mode enabled
./seireitranslations-epub \
  --title "Example EPUB" \
  --author "Example Author" \
  --cover "https://example.com/cover.jpg" \
  --output "example.epub" \
  --urls "example-urls.txt" \
  --debug

# This will:
# 1. Store temporary files in the current directory with name "example.epub.tmp"
# 2. Not clean up the temporary directory after completion
# 3. Cache downloaded files for reuse in subsequent runs
