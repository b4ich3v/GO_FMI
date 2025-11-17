package main

import (
	"bufio"
	"os"
)

// isSpace reports whether b is an ASCII whitespace character
func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

// trimSpace removes leading and trailing ASCII whitespace from s
func trimSpace(s string) string {
	start := 0
	end := len(s)

	// skip whitespace at the beginning
	for start < end && isSpace(s[start]) {
		start++
	}

	// skip whitespace at the end
	for end > start && isSpace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// ReadUsernamesFromFile reads one username per line from the given file path,
// trims surrounding whitespace and skips empty lines
func ReadUsernamesFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var usernames []string

	for scanner.Scan() {
		line := trimSpace(scanner.Text())

		// ignore empty lines
		if line != "" {
			usernames = append(usernames, line)
		}
	}

	return usernames, scanner.Err()
}
