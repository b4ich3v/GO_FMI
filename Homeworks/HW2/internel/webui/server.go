package webui

import (
	"context"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yourname/go-image-crawler/internal/storage"
)

type Server struct {
	Repo     *storage.Repository
	Tmpl     *template.Template
	PageSize int
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/thumb", s.handleThumb)
	mux.HandleFunc("/image", s.handleImage)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	return mux
}

type indexView struct {
	Params     storage.SearchParams
	Items      []storage.ImageRecord
	Total      int
	Pages      int
	PagePrefix string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	p := storage.SearchParams{
		URLContains:      r.URL.Query().Get("url"),
		PageURLContains:  r.URL.Query().Get("page_url"),
		FilenameContains: r.URL.Query().Get("filename"),
		AltContains:      r.URL.Query().Get("alt"),
		TitleContains:    r.URL.Query().Get("title"),
		FormatEquals:     r.URL.Query().Get("format"),
		Page:             atoiDefault(r.URL.Query().Get("page"), 1),
		PageSize:         s.PageSize,
	}
	p.MinWidth = atoiPtr(r.URL.Query().Get("min_w"))
	p.MaxWidth = atoiPtr(r.URL.Query().Get("max_w"))
	p.MinHeight = atoiPtr(r.URL.Query().Get("min_h"))
	p.MaxHeight = atoiPtr(r.URL.Query().Get("max_h"))

	q := r.URL.Query()
	q.Del("page")
	enc := q.Encode()
	pagePrefix := "/?page="
	if enc != "" {
		pagePrefix = "/?" + enc + "&page="
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, total, err := s.Repo.Search(ctx, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pages := total / p.PageSize
	if total%p.PageSize != 0 {
		pages++
	}
	if pages < 1 {
		pages = 1
	}
	view := indexView{
		Params:     p,
		Items:      items,
		Total:      total,
		Pages:      pages,
		PagePrefix: pagePrefix,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.Tmpl.ExecuteTemplate(w, "index.html", view)
}

func (s *Server) handleThumb(w http.ResponseWriter, r *http.Request) {
	id := atou64(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	mime, blob, err := s.Repo.GetThumb(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(blob)
}

func (s *Server) handleImage(w http.ResponseWriter, r *http.Request) {
	id := atou64(r.URL.Query().Get("id"))
	if id == 0 {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rec, err := s.Repo.GetImage(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.Tmpl.ExecuteTemplate(w, "image.html", rec)
}

func atoiDefault(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func atoiPtr(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}

func atou64(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
