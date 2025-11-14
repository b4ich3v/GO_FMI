package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
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

	var results []UserStats
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

	PrintStatsTable(results)
}
