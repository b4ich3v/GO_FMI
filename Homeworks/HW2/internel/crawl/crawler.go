package crawl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/yourname/go-image-crawler/internal/extract"
	"github.com/yourname/go-image-crawler/internal/images"
	"github.com/yourname/go-image-crawler/internal/render"
	"github.com/yourname/go-image-crawler/internal/storage"
	"golang.org/x/net/publicsuffix"
)

const DefaultMaxGoroutines = 64

type Config struct {
	Workers        int
	ImageWorkers   int
	FollowExternal bool
	Timeout        time.Duration
	MaxPages       int
	MaxDepth       int
	MaxGoroutines  int
	Render         bool
	UserAgent      string
	ThumbDir       string
	Logf           func(format string, args ...any)
}

type URLTask struct {
	URL   string
	Depth int
	Kind  string // "page" or "resource"
}

type pageResult struct {
	Task      URLTask
	FinalURL  string
	Links     []string              // page links
	Resources []extract.ResourceRef // css/js resources
	Images    []extract.ImageRef
	Err       error
}

type imageTask struct {
	Ref extract.ImageRef
}

type imageResult struct {
	Task imageTask
	Proc images.Processed
	Err  error
}

func Run(seeds []string, repo *storage.Repository, cfg Config) error {
	if len(seeds) == 0 {
		return errors.New("no seed URLs provided")
	}
	if repo == nil {
		return errors.New("nil repository")
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 8
	}
	if cfg.ImageWorkers <= 0 {
		cfg.ImageWorkers = cfg.Workers
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Minute
	}
	if cfg.MaxPages <= 0 {
		cfg.MaxPages = 1000
	}
	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = 10
	}
	if cfg.MaxGoroutines <= 0 {
		cfg.MaxGoroutines = DefaultMaxGoroutines
	}
	if cfg.ThumbDir == "" {
		cfg.ThumbDir = "./thumbnails"
	}
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}

	// Best-effort cap for our goroutines.
	overhead := 8
	maxWorkersTotal := cfg.MaxGoroutines - overhead
	if maxWorkersTotal < 1 {
		maxWorkersTotal = 1
	}
	if cfg.Workers+cfg.ImageWorkers > maxWorkersTotal {
		if cfg.Workers > maxWorkersTotal {
			cfg.Workers = maxWorkersTotal
			cfg.ImageWorkers = 0
		} else {
			cfg.ImageWorkers = maxWorkersTotal - cfg.Workers
		}
		cfg.Logf("adjusted workers to respect -max-goroutines: workers=%d imageWorkers=%d", cfg.Workers, cfg.ImageWorkers)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	allowedDomains := make(map[string]struct{})
	for _, s := range seeds {
		d := effectiveDomain(s)
		if d != "" {
			allowedDomains[d] = struct{}{}
		}
	}

	// Fetchers
	httpFetcher := render.NewHTTPFetcher(cfg.UserAgent)
	var domFetcher render.Fetcher
	var err error
	if cfg.Render {
		domFetcher, err = render.NewChromedpFetcher(cfg.UserAgent)
		if err != nil {
			cfg.Logf("chromedp unavailable (%v), falling back to HTTP fetcher", err)
			domFetcher = httpFetcher
		}
	} else {
		domFetcher = httpFetcher
	}
	defer func() {
		_ = domFetcher.Close()
		_ = httpFetcher.Close()
	}()

	downloader := images.NewDownloader(cfg.UserAgent, cfg.ThumbDir)

	// Channels
	jobs := make(chan URLTask, cfg.Workers*4)
	pageResults := make(chan pageResult, cfg.Workers*4)
	imgJobs := make(chan imageTask, max(1, cfg.ImageWorkers)*8)
	imgResults := make(chan imageResult, max(1, cfg.ImageWorkers)*8)
	dbInserts := make(chan storage.ImageInsert, 256)

	// Workers
	var workerWG sync.WaitGroup
	var dbWG sync.WaitGroup
	startPageWorkers(ctx, &workerWG, cfg.Workers, jobs, pageResults, domFetcher, httpFetcher)
	startImageWorkers(ctx, &workerWG, cfg.ImageWorkers, imgJobs, imgResults, downloader)
	startDBWriter(ctx, &dbWG, repo, dbInserts, cfg.Logf)

	visited := make(map[string]struct{})       // all crawled URLs (pages + resources)
	visitedImages := make(map[string]struct{}) // dedupe downloads

	// Seed enqueue
	activeTasks := 0
	activeImages := 0
	processedTasks := 0

	for _, s := range seeds {
		u := canonicalizeHTTP(s)
		if u == "" {
			continue
		}
		if _, ok := visited[u]; ok {
			continue
		}
		visited[u] = struct{}{}
		activeTasks++
		jobs <- URLTask{URL: u, Depth: 0, Kind: "page"}
	}

	cfg.Logf("crawl start: workers=%d imageWorkers=%d followExternal=%v render=%v timeout=%s",
		cfg.Workers, cfg.ImageWorkers, cfg.FollowExternal, cfg.Render, cfg.Timeout)

	for {
		if ctx.Err() != nil {
			cfg.Logf("crawl stopped: %v", ctx.Err())
			break
		}
		if activeTasks == 0 && activeImages == 0 {
			break
		}

		select {
		case <-ctx.Done():
			cfg.Logf("timeout reached")
			goto done
		case pr := <-pageResults:
			if pr.Task.URL == "" && pr.Err == nil && len(pr.Links) == 0 && len(pr.Resources) == 0 && len(pr.Images) == 0 {
				continue
			}
			activeTasks--
			processedTasks++
			if pr.Err != nil {
				cfg.Logf("fetch error: %s: %v", pr.Task.URL, pr.Err)
				continue
			}
			if processedTasks >= cfg.MaxPages {
				continue
			}

			scopeBase := pr.FinalURL
			if scopeBase == "" {
				scopeBase = pr.Task.URL
			}

			// Enqueue page links (subject to external + depth)
			if pr.Task.Kind == "page" && pr.Task.Depth < cfg.MaxDepth {
				for _, l := range pr.Links {
					lc := canonicalizeHTTP(l)
					if lc == "" {
						continue
					}
					if _, ok := visited[lc]; ok {
						continue
					}
					if !cfg.FollowExternal && isExternal(scopeBase, lc, allowedDomains) {
						continue
					}
					visited[lc] = struct{}{}
					activeTasks++
					jobs <- URLTask{URL: lc, Depth: pr.Task.Depth + 1, Kind: "page"}
				}
			}

			// Enqueue resources (CSS/JS) regardless of FollowExternal (CDNs should be allowed)
			for _, r := range pr.Resources {
				rc := canonicalizeHTTP(r.URL)
				if rc == "" {
					continue
				}
				if _, ok := visited[rc]; ok {
					continue
				}
				visited[rc] = struct{}{}
				activeTasks++
				jobs <- URLTask{URL: rc, Depth: pr.Task.Depth, Kind: "resource"}
			}

			// Enqueue images (may be on CDNs; do not apply FollowExternal)
			for _, im := range pr.Images {
				key := imageKey(im.URL)
				if key == "" {
					continue
				}
				if _, ok := visitedImages[key]; ok {
					continue
				}
				visitedImages[key] = struct{}{}
				activeImages++
				imgJobs <- imageTask{Ref: im}
			}

		case ir := <-imgResults:
			if ir.Task.Ref.URL == "" && ir.Err == nil && ir.Proc.OriginalURL == "" {
				continue
			}
			activeImages--
			if ir.Err != nil {
				cfg.Logf("image error: %s: %v", ir.Task.Ref.URL, ir.Err)
				continue
			}
			dbInserts <- storage.ImageInsert{
				URL:       ir.Task.Ref.URL,
				PageURL:   ir.Task.Ref.PageURL,
				Filename:  nonEmpty(ir.Task.Ref.Filename, filenameFromURL(ir.Task.Ref.URL)),
				Alt:       ir.Task.Ref.Alt,
				Title:     ir.Task.Ref.Title,
				Width:     ir.Proc.Width,
				Height:    ir.Proc.Height,
				Format:    ir.Proc.Format,
				ThumbPath: ir.Proc.ThumbPath,
				ThumbMIME: ir.Proc.ThumbMIME,
				ThumbBlob: ir.Proc.ThumbBytes,
			}

		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

done:
	close(jobs)
	close(imgJobs)
	workerWG.Wait()

	close(dbInserts)
	dbWG.Wait()

	cfg.Logf("crawl finished: tasks_processed=%d visited_urls=%d unique_images=%d",
		processedTasks, len(visited), len(visitedImages))
	return nil
}

func startPageWorkers(ctx context.Context, wg *sync.WaitGroup, n int, jobs <-chan URLTask, out chan<- pageResult, domFetcher render.Fetcher, httpFetcher render.Fetcher) {
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case t, ok := <-jobs:
					if !ok {
						return
					}

					// Use DOM renderer for pages; for resources use HTTP.
					var fp render.FetchedPage
					var err error
					if t.Kind == "page" {
						fp, err = domFetcher.Fetch(ctx, t.URL)
						if err != nil && httpFetcher != nil && httpFetcher != domFetcher {
							fp, err = httpFetcher.Fetch(ctx, t.URL)
						}
					} else {
						fp, err = httpFetcher.Fetch(ctx, t.URL)
					}
					if err != nil {
						select {
						case out <- pageResult{Task: t, Err: err}:
						case <-ctx.Done():
							return
						}
						continue
					}

					finalURL := fp.FinalURL
					if finalURL == "" {
						finalURL = t.URL
					}

					if looksLikeHTML(fp.ContentType, fp.Body) {
						ext, err := extract.FromHTML(finalURL, fp.Body)
						if err != nil {
							select {
							case out <- pageResult{Task: t, FinalURL: finalURL, Err: err}:
							case <-ctx.Done():
								return
							}
							continue
						}
						select {
						case out <- pageResult{
							Task:      t,
							FinalURL:  finalURL,
							Links:     ext.Links,
							Resources: ext.Resources,
							Images:    ext.Images,
						}:
						case <-ctx.Done():
							return
						}
						continue
					}

					// Parse CSS resources to find images and imported css
					if looksLikeCSS(fp.ContentType, finalURL) {
						cssText := string(fp.Body)
						imports, imgURLs := extract.FromCSS(cssText)
						var res []extract.ResourceRef
						for _, imp := range imports {
							res = append(res, extract.ResourceRef{URL: resolveURL(finalURL, imp), Kind: "css", PageURL: finalURL})
						}
						var imgs []extract.ImageRef
						for _, u := range imgURLs {
							ru := resolveURL(finalURL, u)
							imgs = append(imgs, extract.ImageRef{URL: ru, PageURL: finalURL, Filename: filenameFromURL(ru)})
						}
						select {
						case out <- pageResult{
							Task:      t,
							FinalURL:  finalURL,
							Resources: res,
							Images:    imgs,
						}:
						case <-ctx.Done():
							return
						}
						continue
					}

					// JS and other resources: nothing to extract
					select {
					case out <- pageResult{Task: t, FinalURL: finalURL}:
					case <-ctx.Done():
						return
					}
				}
			}
		}(i)
	}
}

func startImageWorkers(ctx context.Context, wg *sync.WaitGroup, n int, jobs <-chan imageTask, out chan<- imageResult, dl *images.Downloader) {
	if n <= 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case _, ok := <-jobs:
					if !ok {
						return
					}
				}
			}
		}()
		return
	}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case t, ok := <-jobs:
					if !ok {
						return
					}
					ictx, cancel := context.WithTimeout(ctx, 45*time.Second)
					proc, err := dl.DownloadAndThumbnail(ictx, t.Ref.URL)
					cancel()
					select {
					case out <- imageResult{Task: t, Proc: proc, Err: err}:
					case <-ctx.Done():
						return
					}
				}
			}
		}(i)
	}
}

func startDBWriter(ctx context.Context, wg *sync.WaitGroup, repo *storage.Repository, in <-chan storage.ImageInsert, logf func(string, ...any)) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for rec := range in {
			ictx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := repo.InsertImage(ictx, rec)
			cancel()
			if err != nil {
				logf("db insert error: %v", err)
			}
		}
	}()
}

func looksLikeHTML(contentType string, body []byte) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml") {
		return true
	}
	sniff := strings.ToLower(string(body[:min(len(body), 256)]))
	return strings.Contains(sniff, "<html") || strings.Contains(sniff, "<!doctype html")
}

func looksLikeCSS(contentType, urlStr string) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "text/css") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(stripQuery(urlStr)), ".css")
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

func canonicalizeHTTP(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	u.Fragment = ""
	if strings.HasSuffix(u.Host, ":80") && u.Scheme == "http" {
		u.Host = strings.TrimSuffix(u.Host, ":80")
	}
	if strings.HasSuffix(u.Host, ":443") && u.Scheme == "https" {
		u.Host = strings.TrimSuffix(u.Host, ":443")
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

func effectiveDomain(u string) string {
	pu, err := url.Parse(u)
	if err != nil {
		return ""
	}
	host := pu.Hostname()
	if host == "" {
		return ""
	}
	d, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return host
	}
	return d
}

func isExternal(baseURL string, link string, allowed map[string]struct{}) bool {
	ld := effectiveDomain(link)
	if ld == "" {
		return true
	}
	if len(allowed) > 0 {
		_, ok := allowed[ld]
		return !ok
	}
	bd := effectiveDomain(baseURL)
	return bd != "" && ld != bd
}

func resolveURL(baseStr, refStr string) string {
	refStr = strings.TrimSpace(refStr)
	if refStr == "" {
		return ""
	}
	// allow data: for CSS images
	if strings.HasPrefix(strings.ToLower(refStr), "data:") {
		return refStr
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

func imageKey(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	lu := strings.ToLower(u)
	if strings.HasPrefix(lu, "data:") {
		h := sha256.Sum256([]byte(u))
		return "data:" + hex.EncodeToString(h[:])
	}
	return canonicalizeHTTP(u)
}

func nonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
