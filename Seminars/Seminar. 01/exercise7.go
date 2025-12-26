package main

import "fmt"

// Въвежда се цяло число n - n лева. Да се изведе на конзолата как може да се разпределят по банкноти,
// така че да имаме минимален брой банкноти. В условието приемаме, че имаме банкноти от 1 и 2 лева.

func counterForTwo(mult int) int {
	return 2 * mult
}

func minMoney(number int) {
	var countOfTwo int = 0
	var countOfOne int = 0

	for counterForTwo(countOfTwo) <= number {
		countOfTwo += 1
	}

	countOfTwo -= 1
	countOfOne = number - 2*countOfTwo
	fmt.Println("The count of two levas is: ", countOfTwo)
	fmt.Println("The count of one levas is: ", countOfOne)
}

func main() {
	var number int = 0
	fmt.Println("Enter number:")
	fmt.Scan(&number)
	minMoney(number)
}
