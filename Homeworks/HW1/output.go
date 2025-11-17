package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// PrintStatsTable prints all users' statistics as a semicolon-separated table to stdout
// Columns: User; Repos; Followers; Forks; Languages; Activity
func PrintStatsTable(users []UserStats) {
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	// Header row
	if err := writeRow(writer, []string{"User", "Repos", "Followers", "Forks", "Languages", "Activity"}); err != nil {
		fmt.Fprintf(os.Stderr, "error writing header: %v\n", err)
		return
	}

	// One record per user
	for _, user := range users {
		languages := summarizeLangs(user.LangBytes)
		years := summarizeYears(user.ReposPerYear)

		record := []string{
			user.Name,
			intToString(user.TotalRepos),
			intToString(user.Followers),
			intToString(user.TotalForks),
			languages,
			years,
		}

		if err := writeRow(writer, record); err != nil {
			fmt.Fprintf(os.Stderr, "error writing record: %v\n", err)
		}
	}
}

// writeRow writes a single semicolon-separated row (fields) followed by '\n'
func writeRow(w *bufio.Writer, fields []string) error {
	for i, field := range fields {
		if i > 0 {
			if err := w.WriteByte(';'); err != nil {
				return err
			}
		}

		if _, err := w.WriteString(field); err != nil {
			return err
		}
	}

	if err := w.WriteByte('\n'); err != nil {
		return err
	}

	return nil
}

// intToString converts an int to its decimal string representation
func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	isNegative := false
	if n < 0 {
		isNegative = true
		n = -n
	}

	// 20 bytes are enough to hold any 64-bit integer in base 10
	var buffer [20]byte
	i := len(buffer)

	for n > 0 {
		i--
		buffer[i] = byte('0' + n%10)
		n /= 10
	}

	if isNegative {
		i--
		buffer[i] = '-'
	}

	return string(buffer[i:])
}

// summarizeLangs builds a "Lang:Bytes" comma-separated string from a map of language usage
func summarizeLangs(langBytes map[string]int) string {
	var buffer strings.Builder
	separator := ""

	for lang, bytes := range langBytes {
		fmt.Fprintf(&buffer, "%s%s:%d", separator, lang, bytes)
		separator = ", "
	}
	return buffer.String()
}

// summarizeYears builds a "Year:Count" comma-separated string from a map of activity per year
func summarizeYears(yearCounts map[int]int) string {
	var buffer strings.Builder
	separator := ""

	for year, count := range yearCounts {
		fmt.Fprintf(&buffer, "%s%d:%d", separator, year, count)
		separator = ", "
	}
	return buffer.String()
}
