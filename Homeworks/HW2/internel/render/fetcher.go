package render

import "context"

type FetchedPage struct {
	FinalURL    string
	ContentType string
	Body        []byte
	// Rendered indicates the HTML came from a DOM-rendered fetcher (chromedp).
	Rendered bool
}

type Fetcher interface {
	Fetch(ctx context.Context, url string) (FetchedPage, error)
	Close() error
}
