package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env (e.g. GOLANG_TOKEN, GITHUB_API_URL)
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	if len(os.Args) != 2 {
		fmt.Println("Usage: githubstats <input_file>")
		os.Exit(1)
	}

	inputPath := os.Args[1]

	// Read GitHub usernames from the input file (one username per line)
	usernames, err := ReadUsernamesFromFile(inputPath)
	if err != nil {
		log.Fatalf("Error reading usernames: %v", err)
	}

	var results []UserStats

	// For each username: fetch data from GitHub and compute statistics
	for _, username := range usernames {
		user, err := FetchUser(username)
		if err != nil {
			log.Printf("Skipping user %s: %v", username, err)
			continue
		}

		repos, err := FetchRepos(username)
		if err != nil {
			log.Printf("Skipping repos for %s: %v", username, err)
			continue
		}

		languages := FetchAllLanguages(username, repos)
		stats := ComputeStats(user, repos, languages)
		results = append(results, stats)
	}

	// Print final statistics table (CSV) to stdout
	PrintStatsTable(results)
}
