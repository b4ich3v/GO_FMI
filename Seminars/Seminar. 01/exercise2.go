package main

import "fmt"

func countOfDigits(number int) int {
	var counter int = 0

	for number != 0 {
		number /= 10
		counter += 1
	}

	return counter
}

func isNumberThreeDigit(number int) bool {
	return countOfDigits(number) == 3
}

func convertBackwards(number *int) {
	var newNumber int = 0
	var counter1 int = 100
	var counter2 int = 3
	var state bool = false
	var copyOfNumber int = *number

	for {
		if state {
			break
		}
		var currentDigit int = copyOfNumber % 10
		copyOfNumber /= 10
		newNumber += counter1 * currentDigit
		counter1 /= 10
		counter2 -= 1

		if counter2 == 0 {
			state = true
		}

	}

	newNumber += 1
	*number = newNumber
}

func main() {
	var number int = 0
	fmt.Println("Enter a number: ")
	fmt.Scan(&number)

	if !isNumberThreeDigit(number) {
		fmt.Println("Invalid input")
		return
	}

	convertBackwards(&number)
	fmt.Println(number)

}
