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
		y := repo.CreatedAt.Year()
		stats.ReposPerYear[y]++
	}

	for _, m := range langs {
		for lang, size := range m {
			stats.LangBytes[lang] += size
		}
	}

	return stats
}
