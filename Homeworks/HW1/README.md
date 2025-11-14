# How to run

`usernames.txt` must contain one GitHub username per line.

## 1. Generate CSV (locally)

```bash
go run . usernames.txt > web/out.csv
```

## 2. Generate CSV (with Docker + Makefile)

```
make build
make run
```

## 3. View HTML table from out.csv

```
cd web
python3 -m http.server 8080
# then open in browser:
http://localhost:8080/index.html
```
