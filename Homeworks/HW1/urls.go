package main

import "os"

// githubBaseURL returns the base URL for the GitHub API
func githubBaseURL() string {
	if base := os.Getenv("GITHUB_API_URL"); base != "" {
		return base
	}
	return "https://api.github.com"
}

// userURL returns the API URL for fetching user information
func userURL(username string) string {
	return githubBaseURL() + "/users/" + username
}

// reposURL returns the API URL for fetching a user's repositories
func reposURL(username string) string {
	return githubBaseURL() + "/users/" + username + "/repos?per_page=100"
}

// languagesURL returns the API URL for fetching language usage of a repository
func languagesURL(username, repo string) string {
	return githubBaseURL() + "/repos/" + username + "/" + repo + "/languages"
}
