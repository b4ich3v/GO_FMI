package main

type UserStats struct {
	Name         string
	TotalRepos   int
	Followers    int
	TotalForks   int
	LangBytes    map[string]int
	ReposPerYear map[int]int
}

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

	for _, repo := range repos {
		stats.TotalForks += repo.Forks

		createdYear := repo.CreatedAt.Year()
		if createdYear > 0 {
			stats.ReposPerYear[createdYear]++
		}

		updatedYear := repo.UpdatedAt.Year()
		if updatedYear > 0 && updatedYear != createdYear {
			stats.ReposPerYear[updatedYear]++
		}
	}

	for _, m := range langs {
		for lang, bytes := range m {
			stats.LangBytes[lang] += bytes
		}
	}

	return stats
}
