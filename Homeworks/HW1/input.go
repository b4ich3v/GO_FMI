package main

import (
	"bufio"
	"os"
	"strings"
)

func ReadUsernamesFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var usernames []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			usernames = append(usernames, line)
		}
	}
	return usernames, scanner.Err()
}
