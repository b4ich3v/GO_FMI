package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type GitHubUser struct {
	Login       string `json:"login"`
	Name        string `json:"name"`
	PublicRepos int    `json:"public_repos"`
	Followers   int    `json:"followers"`
}

type GitHubRepo struct {
	Name      string    `json:"name"`
	Forks     int       `json:"forks_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func FetchUser(username string) (*GitHubUser, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", username)
	var user GitHubUser
	if err := fetchJSON(url, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func FetchRepos(username string) ([]GitHubRepo, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100", username)
	var repos []GitHubRepo
	if err := fetchJSON(url, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

func FetchLanguages(username, repo string) (map[string]int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/languages", username, repo)
	var langs map[string]int
	if err := fetchJSON(url, &langs); err != nil {
		return nil, err
	}
	return langs, nil
}

func FetchAllLanguages(username string, repos []GitHubRepo) []map[string]int {
	var all []map[string]int
	for _, repo := range repos {
		langs, err := FetchLanguages(username, repo.Name)
		if err != nil {
			continue
		}
		all = append(all, langs)
	}
	return all
}

func fetchJSON(url string, target interface{}) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "GitHubStatsClient")

	if token := os.Getenv("GOLANG_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
