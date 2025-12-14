package extract

import (
	"bytes"
	"net/url"
	"path"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type ImageRef struct {
	URL      string
	Alt      string
	Title    string
	PageURL  string
	Filename string
}

type ResourceRef struct {
	URL     string
	Kind    string // "css", "js", "other"
	PageURL string
}

type Extracted struct {
	Links     []string      // page links to traverse
	Resources []ResourceRef // non-page web resources to crawl (css/js)
	Images    []ImageRef
}

var cssURLRe = regexp.MustCompile(`url\((?P<u>[^)]+)\)`)

func FromHTML(pageURL string, htmlBytes []byte) (Extracted, error) {
	root, err := html.Parse(bytes.NewReader(htmlBytes))
	if err != nil {
		return Extracted{}, err
	}

	base := pageURL
	if b := findBaseHref(root); b != "" {
		base = resolve(pageURL, b)
	}

	seenLinks := map[string]struct{}{}
	seenRes := map[string]struct{}{}
	seenImgs := map[string]struct{}{}

	var out Extracted
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "a":
				if href := attr(n, "href"); href != "" {
					addLink(&out, seenLinks, resolve(base, href))
				}
			case "iframe", "frame":
				if src := attr(n, "src"); src != "" {
					addLink(&out, seenLinks, resolve(base, src))
				}
			case "script":
				if src := attr(n, "src"); src != "" {
					addRes(&out, seenRes, ResourceRef{
						URL:     resolve(base, src),
						Kind:    "js",
						PageURL: pageURL,
					})
				}
			case "link":
				rel := strings.ToLower(attr(n, "rel"))
				as := strings.ToLower(attr(n, "as"))
				href := attr(n, "href")
				if href != "" && (strings.Contains(rel, "stylesheet") || as == "style") {
					addRes(&out, seenRes, ResourceRef{
						URL:     resolve(base, href),
						Kind:    "css",
						PageURL: pageURL,
					})
				}
				// icons as images
				if href != "" && (strings.Contains(rel, "icon") || strings.Contains(rel, "apple-touch-icon") || strings.Contains(rel, "shortcut")) {
					ru := resolve(base, href)
					addImg(&out, seenImgs, ImageRef{
						URL:      ru,
						PageURL:  pageURL,
						Filename: filenameFromURL(ru),
					})
				}
			case "style":
				// inline CSS inside <style> ... </style>
				cssText := nodeText(n)
				for _, u := range parseCSSURLs(cssText) {
					ru := resolve(base, u)
					addImg(&out, seenImgs, ImageRef{
						URL:      ru,
						PageURL:  pageURL,
						Filename: filenameFromURL(ru),
					})
				}
			case "img":
				src := firstNonEmpty(
					attr(n, "src"),
					attr(n, "data-src"),
					attr(n, "data-original"),
					attr(n, "data-lazy-src"),
				)
				if src != "" {
					ru := resolve(base, src)
					addImg(&out, seenImgs, ImageRef{
						URL:      ru,
						Alt:      attr(n, "alt"),
						Title:    attr(n, "title"),
						PageURL:  pageURL,
						Filename: filenameFromURL(ru),
					})
				}
				if ss := attr(n, "srcset"); ss != "" {
					for _, u := range parseSrcset(ss) {
						ru := resolve(base, u)
						addImg(&out, seenImgs, ImageRef{
							URL:      ru,
							Alt:      attr(n, "alt"),
							Title:    attr(n, "title"),
							PageURL:  pageURL,
							Filename: filenameFromURL(ru),
						})
					}
				}
			case "source":
				if ss := attr(n, "srcset"); ss != "" {
					for _, u := range parseSrcset(ss) {
						ru := resolve(base, u)
						addImg(&out, seenImgs, ImageRef{
							URL:      ru,
							PageURL:  pageURL,
							Filename: filenameFromURL(ru),
						})
					}
				}
			case "meta":
				if strings.EqualFold(attr(n, "property"), "og:image") || strings.EqualFold(attr(n, "name"), "og:image") {
					if c := attr(n, "content"); c != "" {
						ru := resolve(base, c)
						addImg(&out, seenImgs, ImageRef{
							URL:      ru,
							PageURL:  pageURL,
							Filename: filenameFromURL(ru),
						})
					}
				}
			case "image":
				// SVG <image href="..."> or xlink:href
				if href := firstNonEmpty(attr(n, "href"), attr(n, "xlink:href")); href != "" {
					ru := resolve(base, href)
					addImg(&out, seenImgs, ImageRef{
						URL:      ru,
						PageURL:  pageURL,
						Filename: filenameFromURL(ru),
					})
				}
			}
		}

		// Parse style attribute on any element for background-image:url(...)
		if n.Type == html.ElementNode {
			if st := attr(n, "style"); st != "" {
				for _, u := range parseCSSURLs(st) {
					ru := resolve(base, u)
					addImg(&out, seenImgs, ImageRef{
						URL:      ru,
						PageURL:  pageURL,
						Filename: filenameFromURL(ru),
					})
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)

	out.Links = normalizeHTTPURLs(out.Links)
	out.Resources = normalizeResources(out.Resources)
	return out, nil
}

func findBaseHref(root *html.Node) string {
	var href string
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if href != "" {
			return
		}
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "base" {
			href = attr(n, "href")
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return href
}

func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func resolve(baseStr, refStr string) string {
	refStr = strings.TrimSpace(refStr)
	if refStr == "" {
		return ""
	}
	l := strings.ToLower(refStr)
	if strings.HasPrefix(l, "javascript:") || strings.HasPrefix(l, "mailto:") || strings.HasPrefix(l, "tel:") {
		return ""
	}
	if strings.HasPrefix(refStr, "//") {
		if b, err := url.Parse(baseStr); err == nil && b.Scheme != "" {
			return b.Scheme + ":" + refStr
		}
		return "https:" + refStr
	}
	b, err := url.Parse(baseStr)
	if err != nil {
		return refStr
	}
	r, err := url.Parse(refStr)
	if err != nil {
		return ""
	}
	u := b.ResolveReference(r)
	u.Fragment = ""
	return u.String()
}

func addLink(out *Extracted, seen map[string]struct{}, u string) {
	if u == "" {
		return
	}
	if _, ok := seen[u]; ok {
		return
	}
	seen[u] = struct{}{}
	out.Links = append(out.Links, u)
}

func addRes(out *Extracted, seen map[string]struct{}, r ResourceRef) {
	if r.URL == "" {
		return
	}
	if _, ok := seen[r.URL]; ok {
		return
	}
	seen[r.URL] = struct{}{}
	out.Resources = append(out.Resources, r)
}

func addImg(out *Extracted, seen map[string]struct{}, img ImageRef) {
	if img.URL == "" {
		return
	}
	if _, ok := seen[img.URL]; ok {
		return
	}
	seen[img.URL] = struct{}{}
	out.Images = append(out.Images, img)
}

func parseSrcset(srcset string) []string {
	parts := strings.Split(srcset, ",")
	var urls []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		fields := strings.Fields(p)
		if len(fields) > 0 {
			urls = append(urls, fields[0])
		}
	}
	return urls
}

func parseCSSURLs(css string) []string {
	matches := cssURLRe.FindAllStringSubmatch(css, -1)
	var out []string
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		u := strings.TrimSpace(m[1])
		u = strings.Trim(u, `"'`)
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}

func filenameFromURL(u string) string {
	pu, err := url.Parse(u)
	if err != nil {
		return ""
	}
	fn := path.Base(pu.Path)
	if fn == "." || fn == "/" {
		return ""
	}
	return fn
}

func normalizeHTTPURLs(in []string) []string {
	out := make([]string, 0, len(in))
	for _, u := range in {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		pu, err := url.Parse(u)
		if err != nil {
			continue
		}
		if pu.Scheme != "http" && pu.Scheme != "https" {
			continue
		}
		pu.Fragment = ""
		out = append(out, pu.String())
	}
	return out
}

func normalizeResources(in []ResourceRef) []ResourceRef {
	out := make([]ResourceRef, 0, len(in))
	for _, r := range in {
		u := strings.TrimSpace(r.URL)
		if u == "" {
			continue
		}
		pu, err := url.Parse(u)
		if err != nil {
			continue
		}
		if pu.Scheme != "http" && pu.Scheme != "https" && !strings.HasPrefix(strings.ToLower(u), "data:") {
			continue
		}
		pu.Fragment = ""
		r.URL = pu.String()
		out = append(out, r)
	}
	return out
}
