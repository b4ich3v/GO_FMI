package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// GitHubUser holds only the user fields we care about from the GitHub API
type GitHubUser struct {
	Login       string `json:"login"`
	Name        string `json:"name"`
	PublicRepos int    `json:"public_repos"`
	Followers   int    `json:"followers"`
}

// GitHubRepo holds only the repository fields we care about from the GitHub API
type GitHubRepo struct {
	Name      string    `json:"name"`
	Forks     int       `json:"forks_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FetchUser retrieves basic profile information for a given GitHub username
func FetchUser(username string) (*GitHubUser, error) {
	apiURL := userURL(username)
	var user GitHubUser

	if err := fetchJSON(apiURL, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// FetchRepos retrieves the list of public repositories for a given user
func FetchRepos(username string) ([]GitHubRepo, error) {
	apiURL := reposURL(username)
	var repos []GitHubRepo

	if err := fetchJSON(apiURL, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// FetchLanguages retrieves the language usage (bytes per language) for a given repo
func FetchLanguages(username, repo string) (map[string]int, error) {
	apiURL := languagesURL(username, repo)
	var languageUsage map[string]int

	if err := fetchJSON(apiURL, &languageUsage); err != nil {
		return nil, err
	}

	return languageUsage, nil
}

// FetchAllLanguages fetches languages for all repos and returns a slice of language maps
func FetchAllLanguages(username string, repos []GitHubRepo) []map[string]int {
	var allLanguages []map[string]int

	for _, repo := range repos {
		languages, err := FetchLanguages(username, repo.Name)
		if err != nil {
			// Skip repos we failed to fetch languages for
			continue
		}
		allLanguages = append(allLanguages, languages)
	}

	return allLanguages
}

// fetchJSON performs an HTTP GET to the given URL and decodes the JSON response into target
func fetchJSON(apiURL string, target interface{}) error {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "GitHubStatsClient")

	// Optional: authenticate with a personal access token if GITHUB_TOKEN is set
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Treat any non-200 response as an error
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
