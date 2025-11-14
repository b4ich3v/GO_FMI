package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func PrintStatsTable(users []UserStats) {
	writer := csv.NewWriter(os.Stdout)

	writer.Comma = ';'

	writer.Write([]string{"User", "Repos", "Followers", "Forks", "Languages", "Activity"})

	for _, u := range users {
		langs := summarizeLangs(u.LangBytes)
		years := summarizeYears(u.ReposPerYear)

		record := []string{
			u.Name,
			strconv.Itoa(u.TotalRepos),
			strconv.Itoa(u.Followers),
			strconv.Itoa(u.TotalForks),
			langs,
			years,
		}

		if err := writer.Write(record); err != nil {
			fmt.Fprintf(os.Stderr, "error writing record to csv: %v\n", err)
		}
	}

	writer.Flush()
}

func summarizeLangs(m map[string]int) string {
	var b strings.Builder
	sep := ""
	for lang, bytes := range m {
		fmt.Fprintf(&b, "%s%s:%d", sep, lang, bytes)
		sep = ", "
	}
	return b.String()
}

func summarizeYears(m map[int]int) string {
	var b strings.Builder
	sep := ""
	for year, count := range m {
		fmt.Fprintf(&b, "%s%d:%d", sep, year, count)
		sep = ", "
	}
	return b.String()
}
