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

// addActivityYear increases the activity count for the year of inputTime, if valid
func addActivityYear(activity map[int]int, inputTime time.Time) {
	if inputTime.IsZero() {
		return
	}

	year := inputTime.Year()
	if year > 0 {
		activity[year] += 1
	}
}

// mergeLangUsage add bytes from source to destination
func mergeLangUsage(destination, source map[string]int) {
	for lang, bytes := range source {
		destination[lang] += bytes
	}
}

// ComputeStats aggregates all relevant statistics for a GitHub user:
// total repos, followers, total forks, language usage and activity per year
func ComputeStats(user *GitHubUser, repos []GitHubRepo, langs []map[string]int) UserStats {
	stats := UserStats{
		Name:         user.Name,
		TotalRepos:   user.PublicRepos,
		Followers:    user.Followers,
		LangBytes:    make(map[string]int),
		ReposPerYear: make(map[int]int),
	}

	if stats.Name == "" {
		stats.Name = user.Login
	}

	for _, currentRepo := range repos {
		stats.TotalForks += currentRepo.Forks

		addActivityYear(stats.ReposPerYear, currentRepo.CreatedAt)

		if !currentRepo.UpdatedAt.IsZero() && currentRepo.UpdatedAt.Year() != currentRepo.CreatedAt.Year() {
			addActivityYear(stats.ReposPerYear, currentRepo.UpdatedAt)
		}
	}

	for _, currentLangMap := range langs {
		mergeLangUsage(stats.LangBytes, currentLangMap)
	}

	return stats
}
