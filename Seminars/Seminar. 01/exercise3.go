package main

import "fmt"

func isEnglish(ch string) bool {
	return (ch >= "A" && ch <= "Z") ||
		(ch >= "a" && ch <= "z")
}

func main() {
	var ch string = ""
	fmt.Println("Enter character:")
	fmt.Scan(&ch)
	fmt.Printf("%v", isEnglish(ch))

}
