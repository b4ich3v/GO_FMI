package main

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/yourname/go-image-crawler/internal/storage"
	"github.com/yourname/go-image-crawler/internal/webui"
)

func main() {
	var (
		mysqlDSN  = flag.String("mysql", "", "MySQL DSN")
		listen    = flag.String("listen", ":8080", "listen address")
		pageSize  = flag.Int("page-size", 40, "results per page")
		templates = flag.String("templates", "./web/templates", "templates directory")
	)
	flag.Parse()

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

	funcs := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}
	tmpl, err := template.New("").Funcs(funcs).ParseGlob(filepath.Join(*templates, "*.html"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "templates:", err)
		os.Exit(1)
	}

	s := &webui.Server{
		Repo:     repo,
		Tmpl:     tmpl,
		PageSize: *pageSize,
	}

	srv := &http.Server{
		Addr:              *listen,
		Handler:           s.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Println("listening on", *listen)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "server:", err)
		os.Exit(1)
	}
}
