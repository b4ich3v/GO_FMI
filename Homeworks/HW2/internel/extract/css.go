package extract

import (
	"regexp"
	"strings"
)

// FromCSS extracts @import'ed CSS resources and image URLs referenced via url(...).
// It's a best-effort extractor (not a full CSS parser) but works well for coursework.
func FromCSS(cssText string) (imports []string, imageURLs []string) {
	// @import "a.css";  @import url(a.css);
	reImport := regexp.MustCompile(`(?i)@import\s+(?:url\()?\s*["']?([^"')\s;]+)`)
	for _, m := range reImport.FindAllStringSubmatch(cssText, -1) {
		if len(m) == 2 {
			imports = append(imports, strings.TrimSpace(m[1]))
		}
	}

	for _, u := range parseCSSURLs(cssText) {
		if isLikelyImageURL(u) {
			imageURLs = append(imageURLs, u)
		}
	}
	return
}

func isLikelyImageURL(u string) bool {
	lu := strings.ToLower(u)
	if strings.HasPrefix(lu, "data:image/") {
		return true
	}
	// ignore fonts
	if strings.Contains(lu, ".woff") || strings.Contains(lu, ".ttf") || strings.Contains(lu, ".eot") || strings.Contains(lu, ".otf") {
		return false
	}
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".ico", ".avif"} {
		if strings.Contains(lu, ext) {
			return true
		}
	}
	return false
}
