package main

import "fmt"

func isValidGrad(number float32) bool {
	return number >= 2 && number <= 6
}

func main() {
	var number float32 = 0
	fmt.Println("Enter grade number:")
	fmt.Scan(&number)
	fmt.Printf("%v", isValidGrad(number))
}
