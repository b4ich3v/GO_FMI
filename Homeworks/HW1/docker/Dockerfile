# syntax=docker/dockerfile:1

# Stage 1: Build binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

# Build binary
RUN go build -o githubstats .

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/githubstats .
COPY .env .  

ENTRYPOINT ["./githubstats"]
CMD ["usernames.txt"]
