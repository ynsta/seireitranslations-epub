package assets

import (
	"embed"
)

//go:embed styles.css
var FS embed.FS

// GetCSS returns the content of the CSS file
func GetCSS() ([]byte, error) {
	return FS.ReadFile("styles.css")
}
