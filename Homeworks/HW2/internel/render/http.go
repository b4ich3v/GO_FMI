package render

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

type HTTPFetcher struct {
	Client    *http.Client
	UserAgent string
}

func NewHTTPFetcher(userAgent string) *HTTPFetcher {
	return &HTTPFetcher{
		Client: &http.Client{
			Timeout: 30 * time.Second,
			// redirect handled automatically; FinalURL is in resp.Request.URL
		},
		UserAgent: userAgent,
	}
}

func (f *HTTPFetcher) Fetch(ctx context.Context, url string) (FetchedPage, error) {
	if f == nil || f.Client == nil {
		return FetchedPage{}, errors.New("nil http fetcher")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return FetchedPage{}, err
	}
	if f.UserAgent != "" {
		req.Header.Set("User-Agent", f.UserAgent)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := f.Client.Do(req)
	if err != nil {
		return FetchedPage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB cap for pages
	if err != nil {
		return FetchedPage{}, err
	}

	ct := resp.Header.Get("Content-Type")
	finalURL := url
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	return FetchedPage{
		FinalURL:    finalURL,
		ContentType: ct,
		Body:        body,
		Rendered:    false,
	}, nil
}

func (f *HTTPFetcher) Close() error { return nil }
