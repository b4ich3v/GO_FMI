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

	if err := writeRow(writer, []string{"User", "Repos", "Followers", "Forks", "Languages", "Activity"}); err != nil {
		fmt.Fprintf(os.Stderr, "error writing header: %v\n", err)
		return
	}

	// Write one record per user
	for _, currentUser := range users {
		langs := summarizeLangs(currentUser.LangBytes)
		years := summarizeYears(currentUser.ReposPerYear)

		record := []string{
			currentUser.Name,
			intToString(currentUser.TotalRepos),
			intToString(currentUser.Followers),
			intToString(currentUser.TotalForks),
			langs,
			years,
		}

		if err := writeRow(writer, record); err != nil {
			fmt.Fprintf(os.Stderr, "error writing record: %v\n", err)
		}

	}
}

// writeRow writes a single semicolon-separated row (fields) followed by '\n'
func writeRow(inputWriter *bufio.Writer, fields []string) error {
	for currentIndex, currentField := range fields {
		if currentIndex > 0 {
			if err := inputWriter.WriteByte(';'); err != nil {
				return err
			}
		}

		if _, err := inputWriter.WriteString(currentField); err != nil {
			return err
		}
	}

	if err := inputWriter.WriteByte('\n'); err != nil {
		return err
	}

	return nil
}

// intToString converts an int to its decimal string representation
func intToString(inputNumber int) string {
	if inputNumber == 0 {
		return "0"
	}

	isNegative := false
	if inputNumber < 0 {
		isNegative = true
		inputNumber = -inputNumber
	}

	// 20 bytes are enough to hold any 64-bit integer in base 10
	var buffer [20]byte
	currentIndex := len(buffer)

	for inputNumber > 0 {
		currentIndex -= 1
		buffer[currentIndex] = byte('0' + inputNumber%10)
		inputNumber /= 10
	}

	if isNegative {
		currentIndex -= 1
		buffer[currentIndex] = '-'
	}

	return string(buffer[currentIndex:])
}

// summarizeLangs builds a "Lang:Bytes" comma-separated string from a map of language usage
func summarizeLangs(mapForLangs map[string]int) string {
	var buffer strings.Builder
	separator := ""

	for lang, bytes := range mapForLangs {
		fmt.Fprintf(&buffer, "%s%s:%d", separator, lang, bytes)
		separator = ", "
	}
	return buffer.String()
}

// summarizeYears builds a "Year:Count" comma-separated string from a map of activity per year
func summarizeYears(mapForYears map[int]int) string {
	var buffer strings.Builder
	separator := ""

	for year, count := range mapForYears {
		fmt.Fprintf(&buffer, "%s%d:%d", separator, year, count)
		separator = ", "
	}
	return buffer.String()
}
