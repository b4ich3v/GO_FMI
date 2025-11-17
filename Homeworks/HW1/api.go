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
	url := userURL(username)
	var user GitHubUser

	if err := fetchJSON(url, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// FetchRepos retrieves the list of public repositories for a given user
func FetchRepos(username string) ([]GitHubRepo, error) {
	url := reposURL(username)
	var repos []GitHubRepo

	if err := fetchJSON(url, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

// FetchLanguages retrieves the language usage (bytes per language) for a given repo
func FetchLanguages(username, repo string) (map[string]int, error) {
	url := languagesURL(username, repo)
	var mapForLanguages map[string]int

	if err := fetchJSON(url, &mapForLanguages); err != nil {
		return nil, err
	}

	return mapForLanguages, nil
}

// FetchAllLanguages fetches languages for all repos and returns a slice of language maps
func FetchAllLanguages(username string, repos []GitHubRepo) []map[string]int {
	var all []map[string]int

	for _, currentRepo := range repos {
		languages, err := FetchLanguages(username, currentRepo.Name)
		if err != nil {
			// skip repos we failed to fetch languages for
			continue
		}
		all = append(all, languages)
	}

	return all
}

// fetchJSON performs an HTTP GET to the given URL and decodes the JSON response into target
func fetchJSON(url string, target interface{}) error {
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "GitHubStatsClient")

	if token := os.Getenv("GOLANG_TOKEN"); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", response.Status)
	}

	return json.NewDecoder(response.Body).Decode(target)
}
