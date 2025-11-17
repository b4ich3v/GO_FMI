package main

import (
	"bufio"
	"os"
)

// return true if the input byte string representation is equal to [' ', '\t', '\n', '\r', '\f', '\v']
func isSpace(inputByte byte) bool {
	switch inputByte {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

// removes all the whitespaces in the start and in the end
func trimSpace(inputString string) string {
	startIndex := 0
	endIndex := len(inputString)

	// loop for whitespaces in the front
	for startIndex < endIndex && isSpace(inputString[startIndex]) {
		startIndex += 1
	}

	// loop for whitespaces in the back
	for endIndex > startIndex && isSpace(inputString[endIndex-1]) {
		endIndex -= 1
	}

	return inputString[startIndex:endIndex]
}

// read line by line in file (validate + parse)
func ReadUsernamesFromFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var usernames []string
	for scanner.Scan() {
		line := trimSpace(scanner.Text())

		// to avoid empty lines
		if line != "" {
			usernames = append(usernames, line)
		}
	}
	return usernames, scanner.Err()
}
