package images

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "golang.org/x/image/ico"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type Processed struct {
	OriginalURL string
	Filename    string
	Format      string
	Width       int
	Height      int
	ThumbPath   string
	ThumbMIME   string
	ThumbBytes  []byte
}

type Downloader struct {
	Client    *http.Client
	UserAgent string
	ThumbDir  string
	MaxBytes  int64
}

func NewDownloader(userAgent, thumbDir string) *Downloader {
	return &Downloader{
		Client: &http.Client{
			Timeout: 40 * time.Second,
		},
		UserAgent: userAgent,
		ThumbDir:  thumbDir,
		MaxBytes:  30 << 20, // 30MB
	}
}

// DownloadAndThumbnail downloads the image (including SVG), detects resolution when possible,
// generates a thumbnail with max width 200px, writes it to filesystem, and returns metadata.
func (d *Downloader) DownloadAndThumbnail(ctx context.Context, imgURL string) (Processed, error) {
	if d == nil || d.Client == nil {
		return Processed{}, errors.New("nil downloader")
	}
	if err := os.MkdirAll(d.ThumbDir, 0o755); err != nil {
		return Processed{}, err
	}

	if strings.HasPrefix(strings.ToLower(imgURL), "data:") {
		return d.fromDataURL(imgURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		return Processed{}, err
	}
	if d.UserAgent != "" {
		req.Header.Set("User-Agent", d.UserAgent)
	}
	req.Header.Set("Accept", "image/*,*/*;q=0.8")

	resp, err := d.Client.Do(req)
	if err != nil {
		return Processed{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, d.MaxBytes))
	if err != nil {
		return Processed{}, err
	}

	ct := resp.Header.Get("Content-Type")
	return d.processBytes(imgURL, body, ct)
}

func (d *Downloader) fromDataURL(dataURL string) (Processed, error) {
	// data:[<mediatype>][;base64],<data>
	comma := strings.IndexByte(dataURL, ',')
	if comma < 0 {
		return Processed{}, fmt.Errorf("invalid data url")
	}
	meta := dataURL[:comma]
	raw := dataURL[comma+1:]

	mimeType := "text/plain"
	if strings.HasPrefix(meta, "data:") && len(meta) > 5 {
		mimeType = meta[5:]
	}
	isB64 := strings.Contains(meta, ";base64")
	if idx := strings.IndexByte(mimeType, ';'); idx >= 0 {
		mimeType = mimeType[:idx]
	}
	if mimeType == "" {
		mimeType = "text/plain"
	}

	var b []byte
	var err error
	if isB64 {
		b, err = base64.StdEncoding.DecodeString(raw)
	} else {
		var s string
		s, err = url.QueryUnescape(raw)
		if err == nil {
			b = []byte(s)
		}
	}
	if err != nil {
		return Processed{}, err
	}

	return d.processBytes(dataURL, b, mimeType)
}

func (d *Downloader) processBytes(srcURL string, b []byte, contentType string) (Processed, error) {
	// Guess by content type or extension
	lct := strings.ToLower(contentType)
	ext := strings.ToLower(filepath.Ext(stripQuery(srcURL)))
	if strings.Contains(lct, "image/svg") || ext == ".svg" || bytes.Contains(b[:min(len(b), 256)], []byte("<svg")) {
		return d.processSVG(srcURL, b)
	}
	return d.processRaster(srcURL, b, contentType)
}

func (d *Downloader) processRaster(srcURL string, b []byte, contentType string) (Processed, error) {
	img, format, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		ct := strings.TrimSpace(contentType)
		if ct == "" {
			ct = "(unknown content-type)"
		}
		// Helpful when a server returns HTML (e.g. 403 page) for an image URL.
		sn := strings.TrimSpace(string(b[:min(len(b), 96)]))
		if len(sn) > 0 {
			return Processed{}, fmt.Errorf("image decode failed: %w (content-type=%s, sniff=%q)", err, ct, sn)
		}
		return Processed{}, fmt.Errorf("image decode failed: %w (content-type=%s)", err, ct)
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	thumb := resizeMaxWidth(img, 200)
	tb := new(bytes.Buffer)
	if err := jpeg.Encode(tb, thumb, &jpeg.Options{Quality: 85}); err != nil {
		return Processed{}, err
	}

	hash := sha256.Sum256([]byte(srcURL))
	name := hex.EncodeToString(hash[:]) + ".jpg"
	path := filepath.Join(d.ThumbDir, name)
	if err := os.WriteFile(path, tb.Bytes(), 0o644); err != nil {
		return Processed{}, err
	}

	return Processed{
		OriginalURL: srcURL,
		Filename:    filenameFromURL(srcURL),
		Format:      strings.ToLower(format),
		Width:       w,
		Height:      h,
		ThumbPath:   path,
		ThumbMIME:   "image/jpeg",
		ThumbBytes:  tb.Bytes(),
	}, nil
}

var (
	reSVGWidth  = regexp.MustCompile(`(?i)\bwidth\s*=\s*["']\s*([0-9]+(?:\.[0-9]+)?)`)
	reSVGHeight = regexp.MustCompile(`(?i)\bheight\s*=\s*["']\s*([0-9]+(?:\.[0-9]+)?)`)
	reViewBox   = regexp.MustCompile(`(?i)\bviewBox\s*=\s*["']\s*([0-9\s\.-]+)\s*["']`)
)

func (d *Downloader) processSVG(srcURL string, b []byte) (Processed, error) {
	w, h := svgSize(b)

	// Thumbnail for SVG: keep SVG bytes. Best-effort: if width missing, inject width="200" and keep viewBox.
	tb := b
	if w == 0 {
		tb = injectSVGWidth(b, 200)
	}

	hash := sha256.Sum256([]byte(srcURL))
	name := hex.EncodeToString(hash[:]) + ".svg"
	path := filepath.Join(d.ThumbDir, name)
	if err := os.WriteFile(path, tb, 0o644); err != nil {
		return Processed{}, err
	}

	return Processed{
		OriginalURL: srcURL,
		Filename:    filenameFromURL(srcURL),
		Format:      "svg",
		Width:       w,
		Height:      h,
		ThumbPath:   path,
		ThumbMIME:   "image/svg+xml",
		ThumbBytes:  tb,
	}, nil
}

func svgSize(b []byte) (w, h int) {
	s := string(b)
	if m := reSVGWidth.FindStringSubmatch(s); len(m) == 2 {
		w = atoiSafe(m[1])
	}
	if m := reSVGHeight.FindStringSubmatch(s); len(m) == 2 {
		h = atoiSafe(m[1])
	}
	if w == 0 || h == 0 {
		if m := reViewBox.FindStringSubmatch(s); len(m) == 2 {
			fields := strings.Fields(m[1])
			if len(fields) == 4 {
				vw := atoiSafe(fields[2])
				vh := atoiSafe(fields[3])
				if w == 0 {
					w = vw
				}
				if h == 0 {
					h = vh
				}
			}
		}
	}
	return
}

func injectSVGWidth(b []byte, width int) []byte {
	s := string(b)
	// insert width attr after "<svg"
	i := strings.Index(strings.ToLower(s), "<svg")
	if i < 0 {
		return b
	}
	j := strings.Index(s[i:], ">")
	if j < 0 {
		return b
	}
	j = i + j
	openTag := s[i:j]
	if strings.Contains(strings.ToLower(openTag), "width=") {
		return b
	}
	openTagNew := openTag + fmt.Sprintf(` width="%d"`, width)
	return []byte(s[:i] + openTagNew + s[j:])
}

func resizeMaxWidth(img image.Image, maxW int) image.Image {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 || w <= maxW {
		return img
	}
	newW := maxW
	newH := int(float64(h) * float64(newW) / float64(w))
	if newH <= 0 {
		newH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}

func stripQuery(u string) string {
	pu, err := url.Parse(u)
	if err != nil {
		return u
	}
	pu.RawQuery = ""
	pu.Fragment = ""
	return pu.String()
}

func filenameFromURL(u string) string {
	pu, err := url.Parse(u)
	if err != nil {
		return ""
	}
	fn := filepath.Base(pu.Path)
	if fn == "." || fn == "/" {
		return ""
	}
	return fn
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func atoiSafe(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// strip units (px, etc.)
	s = strings.TrimRightFunc(s, func(r rune) bool { return (r < '0' || r > '9') && r != '.' && r != '-' })
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	if err != nil {
		return 0
	}
	if f < 0 {
		f = 0
	}
	return int(f + 0.5)
}
