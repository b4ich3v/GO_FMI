# How to run

## 1. Generate CSV (locally)

```bash
go run . usernames.txt > out.csv
```

## 2. Generate CSV (with Docker + Makefile)
```
make build
make run
```

## 3. View HTML table from out.csv

```
python3 -m http.server 8080
http://localhost:8080/index.html
```
