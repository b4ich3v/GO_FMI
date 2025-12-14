# Go Image Crawler (Demo)

## Dependencies
- Go (1.22+)
- Docker + Docker Compose
- MySQL container in `docker-compose.yml` must be on **port 3307** (example: `"3307:3306"`)

---

## Container (start/stop)
```bash
docker compose up -d
docker compose down -v
docker ps | grep mysql
```

---

## Database initialization
```bash
docker compose exec mysql mysql -u crawler -pcrawler -e "CREATE DATABASE IF NOT EXISTS imagedb;"
docker compose exec -T mysql mysql -u crawler -pcrawler imagedb < migrations/001_init.sql
docker compose exec mysql mysql -u crawler -pcrawler imagedb -e "SHOW TABLES;"
```

---

## Dependencies
```bash
go mod tidy
```

---

## Demo: Crawl
```bash
go run ./cmd/crawler   -mysql "crawler:crawler@tcp(127.0.0.1:3307)/imagedb?parseTime=true"   -workers 16 -image-workers 16   -max-pages 30   -max-depth 1   -follow-external=false   -max-goroutines 128   -render=false   -thumbdir ./thumbnails   https://go.dev
```

---

## Check
```bash
ls -lah thumbnails | head
docker compose exec mysql mysql -u crawler -pcrawler imagedb -e "SELECT COUNT(*) AS cnt FROM images;"
```

---

## Web UI (http://localhost:8080)
```bash
go run ./cmd/web   -mysql "crawler:crawler@tcp(127.0.0.1:3307)/imagedb?parseTime=true"   -listen "127.0.0.1:8080"
```
