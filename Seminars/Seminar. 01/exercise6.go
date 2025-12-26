package main

import "fmt"

func toPower(number int, power int) int {
	var counter int = 0
	var result int = 1

	for counter != power {
		result *= number
		counter += 1
	}

	return result
}

func main() {
	var number1 int = 0
	var number2 int = 0
	fmt.Println("Enter number1:")
	fmt.Scan(&number1)
	fmt.Println("Enter number2:")
	fmt.Scan(&number2)

	fmt.Printf("%v\n", toPower((number1+number2), 4)-toPower((number1-number2), 2))

}
