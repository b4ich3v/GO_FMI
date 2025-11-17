package main

import "os"

// load the GITHUB_API_URL variable + fallback mechanism
func githubBaseURL() string {
	if base := os.Getenv("GITHUB_API_URL"); base != "" {
		return base
	}
	return "https://api.github.com"
}

// https://api.github.com/users/{username}
func userURL(username string) string {
	return githubBaseURL() + "/users/" + username
}

// https://api.github.com/users/{username}/repos?per_page=100
func reposURL(username string) string {
	return githubBaseURL() + "/users/" + username + "/repos?per_page=100"
}

// https://api.github.com/repos/{username}/{repo}/languages
func languagesURL(username, repo string) string {
	return githubBaseURL() + "/repos/" + username + "/" + repo + "/languages"
}
