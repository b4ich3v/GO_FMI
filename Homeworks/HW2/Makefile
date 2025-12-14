.PHONY: crawl serve migrate

MYSQL_DSN ?= crawler:crawler@tcp(127.0.0.1:3306)/imagedb?parseTime=true

migrate:
	mysql -h 127.0.0.1 -P 3306 -u crawler -pcrawler imagedb < migrations/001_init.sql

crawl:
	go run ./cmd/crawler -mysql "$(MYSQL_DSN)" -workers 8 -image-workers 8 -timeout 2m -render=true -thumbdir ./thumbnails https://example.com

serve:
	go run ./cmd/web -mysql "$(MYSQL_DSN)" -listen :8080
