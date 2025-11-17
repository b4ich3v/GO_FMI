package main

import "time"

type UserStats struct {
	Name         string
	TotalRepos   int
	Followers    int
	TotalForks   int
	LangBytes    map[string]int
	ReposPerYear map[int]int
}

// addActivityYear increases the activity count for the year of t, if valid
func addActivityYear(activity map[int]int, t time.Time) {
	if t.IsZero() {
		return
	}

	year := t.Year()
	if year > 0 {
		activity[year]++
	}
}

// mergeLangUsage adds bytes from source to destination by language
func mergeLangUsage(destination, source map[string]int) {
	for language, bytes := range source {
		destination[language] += bytes
	}
}

// ComputeStats aggregates all relevant statistics for a GitHub user:
// total repos, followers, total forks, language usage and activity per year
func ComputeStats(user *GitHubUser, repos []GitHubRepo, languages []map[string]int) UserStats {
	stats := UserStats{
		Name:         user.Name,
		TotalRepos:   user.PublicRepos,
		Followers:    user.Followers,
		LangBytes:    make(map[string]int),
		ReposPerYear: make(map[int]int),
	}

	// Fallback to login if the user has no display name
	if stats.Name == "" {
		stats.Name = user.Login
	}

	// Aggregate forks and activity per year from repositories
	for _, repo := range repos {
		stats.TotalForks += repo.Forks

		addActivityYear(stats.ReposPerYear, repo.CreatedAt)

		if !repo.UpdatedAt.IsZero() && repo.UpdatedAt.Year() != repo.CreatedAt.Year() {
			addActivityYear(stats.ReposPerYear, repo.UpdatedAt)
		}
	}

	// Aggregate language usage across all repositories
	for _, langMap := range languages {
		mergeLangUsage(stats.LangBytes, langMap)
	}

	return stats
}
