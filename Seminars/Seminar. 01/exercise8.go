package main

import (
	"fmt"
)

func isGreater(number1 int, number2 int) bool {
	return number1 >= number2
}

func main() {
	var number1 int = 0
	var number2 int = 0
	fmt.Println("Enter number1:")
	fmt.Scan(&number1)
	fmt.Println("Enter number1:")
	fmt.Scan(&number2)

	fmt.Printf("%v\n", isGreater(number1, number2))
}
