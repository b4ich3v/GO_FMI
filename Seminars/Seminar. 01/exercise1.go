package main

import "fmt"

func isNumberEven(number int) bool {
	return number%2 == 0
}

func main() {
	var number int = 0
	fmt.Println("Enter a number: ")
	fmt.Scan(&number)
	var result bool = isNumberEven(number)

	if result == true {
		fmt.Println(1)
	} else {
		fmt.Println(0)
	}

}
