# Product Context: SeireiTranslations EPUB Generator

## Why This Project Exists
The SeireiTranslations EPUB Generator exists to solve the common problem faced by light novel readers who prefer reading content in an e-book format rather than directly from blog posts. SeireiTranslations publishes translated light novels through their blog, but reading directly from the blog has several limitations:

1. The reading experience on blogs is suboptimal for long-form content
2. Readers cannot easily read offline
3. Blog formatting is inconsistent and may include distracting elements
4. Tracking progress across multiple chapters is difficult
5. No easy way to organize multiple chapters into a cohesive book

This tool bridges the gap between web-published content and e-reader convenience.

## Problems It Solves

### For Readers
- **Improved Reading Experience**: Converts blog posts into a properly formatted EPUB file that can be read on any e-reader device
- **Offline Reading**: Allows readers to download content for offline reading
- **Consistent Formatting**: Removes inconsistent styling, ads, and navigation elements
- **Chapter Organization**: Properly organizes chapters into a cohesive book
- **Progress Tracking**: E-readers can track reading progress across the entire novel
- **Custom Metadata**: Allows setting proper metadata (title, author) for better organization

### For Technical Users
- **Automation**: Automates the process of collecting and formatting content from multiple blog posts
- **Flexibility**: Command-line interface allows for scripting and integration with other tools
- **Debugging**: Debug mode allows for troubleshooting extraction issues
- **Content Cleaning**: Automatically removes non-content elements like navigation, ads, etc.
- **Image Handling**: Properly processes and includes images within the content

## How It Should Work

### User Workflow
1. User prepares a text file with a list of chapter URLs in the format `Chapter Name::URL`
2. User runs the command with required parameters:
   - Title of the novel
   - Author name
   - Cover image URL
   - Output file name
   - Path to the URL list file
   - Optional debug flag
3. The tool processes each URL, extracting only the relevant content
4. The tool packages all chapters into a single EPUB file
5. The user receives a properly formatted EPUB file ready for reading on any e-reader

### System Workflow
1. Parse command-line arguments
2. Set up temporary directory and EPUB structure
3. Download cover image
4. Read and parse URL list
5. For each URL:
   - Download and parse HTML
   - Extract relevant content using extraction patterns
   - Clean HTML (remove inline styles, fix formatting)
   - Process and download images
   - Add chapter to EPUB
6. Generate final EPUB file
7. Clean up temporary files (unless in debug mode)

## User Experience Goals
- **Simplicity**: Simple command-line interface with clear parameters
- **Reliability**: Consistent extraction of relevant content from blog posts
- **Quality**: High-quality EPUB output with proper formatting and organization
- **Transparency**: Clear feedback on progress and any issues encountered
- **Flexibility**: Support for customizing title, author, cover image
- **Troubleshooting**: Debug mode for examining intermediate files and understanding extraction issues
