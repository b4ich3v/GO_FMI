package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env (e.g. GITHUB_TOKEN, GITHUB_API_URL)
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if len(os.Args) != 2 {
		fmt.Println("Usage: githubstats <input_file>")
		os.Exit(1)
	}

	usernames, err := ReadUsernamesFromFile(os.Args[1])
	if err != nil {
		log.Fatalf("Error reading usernames: %v", err)
	}

	// For each username: fetch data from GitHub and compute statistics
	var results []UserStats
	for _, currentUsername := range usernames {
		user, err := FetchUser(currentUsername)
		if err != nil {
			log.Printf("Skipping user %s: %v", currentUsername, err)
			continue
		}

		repos, err := FetchRepos(currentUsername)
		if err != nil {
			log.Printf("Skipping repos for %s: %v", currentUsername, err)
			continue
		}

		languages := FetchAllLanguages(currentUsername, repos)
		stats := ComputeStats(user, repos, languages)
		results = append(results, stats)
	}

	// Print final statistics table (CSV) to stdout
	PrintStatsTable(results)
}
