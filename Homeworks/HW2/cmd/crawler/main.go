package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/yourname/go-image-crawler/internal/crawl"
	"github.com/yourname/go-image-crawler/internal/storage"
)

func main() {
	var (
		mysqlDSN       = flag.String("mysql", "", "MySQL DSN (e.g. user:pass@tcp(host:3306)/db?parseTime=true)")
		workers        = flag.Int("workers", 8, "page worker pool size")
		imageWorkers   = flag.Int("image-workers", 8, "image download/thumbnail worker pool size")
		followExternal = flag.Bool("follow-external", false, "follow external page links (images may still be downloaded from CDNs)")
		timeout        = flag.Duration("timeout", 2*time.Minute, "crawl timeout (e.g. 2m, 30s)")
		maxPages       = flag.Int("max-pages", 1000, "maximum pages to process (safety)")
		maxDepth       = flag.Int("max-depth", 10, "maximum traversal depth (safety)")
		maxG           = flag.Int("max-goroutines", crawl.DefaultMaxGoroutines, "max goroutines created by this project (best-effort)")
		render         = flag.Bool("render", true, "use headless browser (chromedp) to render JS/SPA pages")
		thumbDir       = flag.String("thumbdir", "./thumbnails", "thumbnail directory")
		userAgent      = flag.String("user-agent", "GoImageCrawler/1.0 (+https://example.local)", "HTTP User-Agent")
	)
	flag.Parse()

	seeds := flag.Args()
	if len(seeds) == 0 {
		fmt.Fprintln(os.Stderr, "usage: crawler [flags] <seed_url1> <seed_url2> ...")
		flag.PrintDefaults()
		os.Exit(2)
	}
	if *mysqlDSN == "" {
		fmt.Fprintln(os.Stderr, "error: -mysql is required")
		os.Exit(2)
	}

	repo, err := storage.OpenMySQL(*mysqlDSN)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mysql:", err)
		os.Exit(1)
	}
	defer repo.Close()

	cfg := crawl.Config{
		Workers:        *workers,
		ImageWorkers:   *imageWorkers,
		FollowExternal: *followExternal,
		Timeout:        *timeout,
		MaxPages:       *maxPages,
		MaxDepth:       *maxDepth,
		MaxGoroutines:  *maxG,
		Render:         *render,
		UserAgent:      *userAgent,
		ThumbDir:       *thumbDir,
	}

	if err := crawl.Run(seeds, repo, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "crawl:", err)
		os.Exit(1)
	}
}
