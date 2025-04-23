# Active Context: SeireiTranslations EPUB Generator

## Current Work Focus
The SeireiTranslations EPUB Generator appears to be a functional tool that can successfully scrape and package content from SeireiTranslations blog posts into EPUB format. The current focus is on maintaining and potentially improving the content extraction patterns to adapt to any changes in the blog's HTML structure.

### Key Components Under Active Development
- **Content Extraction Patterns**: The most critical and complex part of the system, requiring ongoing refinement to handle blog variations
- **HTML Processing**: Ensuring clean, consistent HTML formatting for optimal e-reader display
- **Image Handling**: Proper downloading and processing of images within the content

## Recent Changes
Based on the observed file structure and code, recent development appears to have focused on:

1. Refactoring the codebase into a modular structure
2. Enhancing the content extraction logic with multiple pattern approaches
3. Improving debug capabilities for troubleshooting extraction issues
4. Adding special handling for formatted elements like "Part X" sections
5. Added an attribution chapter as the first chapter in generated EPUBs
   - Provides credit to SeireiTranslations for the translations
   - Includes links to support translators via Ko-Fi and Patreon
   - Lists all source URLs used in the EPUB

## Next Steps
Potential areas for further development:

1. **Pattern Enhancement**: Continuously refine extraction patterns to handle edge cases in blog formatting
2. **Performance Optimization**: Look for opportunities to improve processing speed for multi-chapter novels
3. **Error Handling**: Enhance error recovery to handle partial failures gracefully
4. **Testing**: Expand testing with various blog post formats to ensure robust extraction
5. **User Experience**: Consider streamlining the command-line interface or adding convenience features

## Active Decisions and Considerations

### Content Extraction Strategy
The current approach uses a multi-pattern strategy with fallbacks:
- Primary pattern looks for heading elements (h4) or centered paragraphs
- Advanced pattern uses multiple selector combinations to find content start
- Fallback pattern extracts from the main content area as a last resort

This approach prioritizes flexibility but may need further refinement for specific edge cases.

### HTML Cleaning Enhancement
We've integrated go-readability (github.com/go-shiori/go-readability) as a final HTML cleaning step:
- Custom extraction and initial cleaning is performed first using our existing patterns
- go-readability is then applied as a final step to further simplify and standardize the HTML
- Images are preserved and reinserted if needed to ensure they aren't lost during processing
- All debug info is preserved for troubleshooting
- The original cleaning method is used as a fallback if readability processing fails

### Debug Mode Implementation
Debug mode has been implemented with several features:
- Preservation of temporary files
- Detailed logging of processing steps
- Saving of intermediate extraction results
- Caching of downloaded files

This implementation provides good visibility into the extraction process but increases disk usage.

### HTML Cleaning Approach
The HTML cleaning approach focuses on:
- Initial cleaning that removes inline styles and unnecessary attributes
- Converting special elements to appropriate heading levels
- Removing navigation and advertising elements
- Preserving important structural elements
- Final standardization with go-readability for cleaner, more EPUB-friendly HTML

This multi-stage approach balances custom handling of SeireiTranslations-specific elements with standardized HTML cleaning for optimal e-reader display.

## Important Patterns and Preferences

### Code Organization
- Modular packages with clear responsibilities
- Internal packages for application-specific functionality
- Potential reusable utilities in pkg directory
- Clear separation of concerns between components

### Error Handling
- Detailed logging of errors with context
- Graceful continuation when possible (e.g., skipping problematic URLs)
- Appropriate exit codes for command-line usage

### Configuration Management
- Command-line arguments for all required parameters
- Debug flag for troubleshooting
- URL list file for specifying chapters

## Learnings and Project Insights

### Critical Components
The most critical components identified are:
1. Content extraction logic in the scraper package
2. HTML cleaning in the processor package
3. Chapter organization in the EPUB generator

These components require the most attention for maintenance and improvement.

### Dependency Considerations
The project relies on external libraries for:
- HTML parsing and manipulation (goquery)
- EPUB generation (go-epub)

These dependencies are stable but should be monitored for updates or issues.

### Performance Observations
- URL processing is sequential with a small delay between requests
- Image processing may be a performance bottleneck for image-heavy content
- Debug mode significantly increases disk usage
