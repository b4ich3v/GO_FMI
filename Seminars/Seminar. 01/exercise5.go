package main

import "fmt"

func getCountOfDigits(number int) int {
	var counter int = 0

	for number != 0 {
		counter += 1
		number /= 10
	}

	return counter
}

func reverseNumber(number *int) {
	var newNumber int = 0
	var copyNumber int = *number
	var countOfDigits int = getCountOfDigits(*number)
	var iter int = 1

	for i := 0; i < countOfDigits-1; i++ {
		iter *= 10
	}

	for copyNumber != 0 {
		var currentDigit int = copyNumber % 10
		newNumber += iter * currentDigit
		iter /= 10
		copyNumber /= 10

	}

	*number = newNumber
}

func printWithoutLastDigit(number int) {
	reverseNumber(&number)
	number /= 10
	fmt.Println(number)
}

func main() {
	var number int = 0
	fmt.Println("Enter grade number:")
	fmt.Scan(&number)
	printWithoutLastDigit(number)
}
