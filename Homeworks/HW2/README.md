# Go Image Crawler (Commands)

## Prereqs
- Go 1.22+
- Docker + Docker Compose
- `docker-compose.yml`: MySQL exposed on `3307:3306`

---

## 1) MySQL (start/stop)
```bash
docker compose up -d
docker compose ps
docker compose down -v
```

---

## 2) DB schema
```bash
# (run inside the mysql container over TCP)
docker compose exec -T mysql mysql -h 127.0.0.1 -P 3306 -u crawler -pcrawler -e "CREATE DATABASE IF NOT EXISTS imagedb;"
docker compose exec -T mysql mysql -h 127.0.0.1 -P 3306 -u crawler -pcrawler imagedb < migrations/001_init.sql
docker compose exec -T mysql mysql -h 127.0.0.1 -P 3306 -u crawler -pcrawler imagedb -e "SHOW TABLES;"
```

---

## 3) DB indexes (optional but recommended)
```bash
docker compose exec -T mysql mysql -h 127.0.0.1 -P 3306 -u crawler -pcrawler imagedb < migrations/002_add_indexes.sql
```

---

## 4) Go deps
```bash
go mod tidy
```

---

## 5) Reset data (zsh-safe)
```bash
docker compose exec -T mysql mysql -h 127.0.0.1 -P 3306 -u crawler -pcrawler imagedb -e "TRUNCATE TABLE images;"
rm -rf thumbnails
mkdir -p thumbnails
```

---

## 6) Demo: normal sites (no JS render)
```bash
go run ./cmd/crawler \
  -mysql "crawler:crawler@tcp(127.0.0.1:3307)/imagedb?parseTime=true" \
  -workers 16 -image-workers 16 \
  -max-pages 30 -max-depth 1 \
  -follow-external=false \
  -max-goroutines 128 \
  -render=false \
  -thumbdir ./thumbnails \
  https://go.dev

go run ./cmd/crawler \
  -mysql "crawler:crawler@tcp(127.0.0.1:3307)/imagedb?parseTime=true" \
  -workers 16 -image-workers 16 \
  -max-pages 30 -max-depth 1 \
  -follow-external=false \
  -max-goroutines 128 \
  -render=false \
  -thumbdir ./thumbnails \
  https://www.rust-lang.org
```

---

## 7) Demo: SPA (render=false vs render=true)

### Start local SPA server
```bash
cd demo-spa
python3 -m http.server 9000
```

### Crawl without render (expect 0)
```bash
cd ..
docker compose exec -T mysql mysql -h 127.0.0.1 -P 3306 -u crawler -pcrawler imagedb -e "TRUNCATE TABLE images;"
rm -rf thumbnails && mkdir -p thumbnails

go run ./cmd/crawler \
  -mysql "crawler:crawler@tcp(127.0.0.1:3307)/imagedb?parseTime=true" \
  -workers 8 -image-workers 8 \
  -max-pages 2 -max-depth 0 \
  -follow-external=true \
  -max-goroutines 128 \
  -render=false \
  -timeout 20s \
  -thumbdir ./thumbnails \
  http://localhost:9000
```

### Crawl with render (expect >0)
```bash
go run ./cmd/crawler \
  -mysql "crawler:crawler@tcp(127.0.0.1:3307)/imagedb?parseTime=true" \
  -workers 8 -image-workers 8 \
  -max-pages 2 -max-depth 0 \
  -follow-external=true \
  -max-goroutines 128 \
  -render=true \
  -timeout 40s \
  -thumbdir ./thumbnails \
  http://localhost:9000
```

### Check
```bash
docker compose exec -T mysql mysql -h 127.0.0.1 -P 3306 -u crawler -pcrawler imagedb -e "SELECT COUNT(*) AS cnt FROM images;"
ls -lah thumbnails | head
```

---

## 8) Prove indexes (EXPLAIN)
```bash
docker compose exec -T mysql mysql -h 127.0.0.1 -P 3306 -u crawler -pcrawler imagedb -e \
"EXPLAIN ANALYZE SELECT id FROM images WHERE format='svg' ORDER BY created_at DESC LIMIT 50;"
```

---

## 9) Web UI (http://localhost:8080)
```bash
go run ./cmd/web \
  -mysql "crawler:crawler@tcp(127.0.0.1:3307)/imagedb?parseTime=true" \
  -listen "127.0.0.1:8080"
```

---

## 10) If chromedp errors (PermissionBlock)
```bash
go get github.com/chromedp/chromedp@latest
go get github.com/chromedp/cdproto@latest
go mod tidy
```
