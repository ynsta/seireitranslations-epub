package processor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// HTMLProcessor handles HTML content processing
type HTMLProcessor struct{}

// NewHTMLProcessor creates a new HTMLProcessor
func NewHTMLProcessor() *HTMLProcessor {
	return &HTMLProcessor{}
}

// CleanHTML removes inline styles and other unnecessary attributes
func (p *HTMLProcessor) CleanHTML(html string) string {
	// Create a document to work with
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	// Remove inline styles from all elements except images
	doc.Find("[style]").Each(func(i int, s *goquery.Selection) {
		// Skip images - we want to keep their styles for responsive display
		if !s.Is("img") {
			s.RemoveAttr("style")
		}
	})

	// Remove other unnecessary attributes that might affect styling
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		// Skip images - already handled
		if !s.Is("img") {
			s.RemoveAttr("align")
			s.RemoveAttr("width")
			s.RemoveAttr("height")
			s.RemoveAttr("border")
			s.RemoveAttr("bgcolor")
			s.RemoveAttr("color")
		}
	})

	// Remove empty paragraphs
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		trimmedHTML := strings.TrimSpace(html)
		if trimmedHTML == "" || trimmedHTML == "&nbsp;" {
			s.Remove()
		}
	})

	// Clean up line breaks and spacing
	html, _ = doc.Html()

	// Remove excessive whitespace
	re := regexp.MustCompile(`\s{2,}`)
	html = re.ReplaceAllString(html, " ")

	// Remove empty lines
	re = regexp.MustCompile(`(?m)^\s*$[\r\n]*`)
	html = re.ReplaceAllString(html, "")

	return html
}

// ProcessChapterContent creates properly formatted HTML for a chapter
func (p *HTMLProcessor) ProcessChapterContent(title string, content string) string {
	// Create chapter HTML with proper styling and minimal margins
	return fmt.Sprintf(`<html>
<head>
    <title>%s</title>
</head>
<body class="chapter" style="margin:0; padding:0;">
    <h2>%s</h2>
    %s
</body>
</html>`, title, title, content)
}
