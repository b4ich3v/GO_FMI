package main

import "fmt"

func validateNumbre(number int) bool {
	return number >= 1 && number <= 7
}

func main() {

	var number int = 0
	fmt.Printf("Enter a number:")
	fmt.Scan(&number)

	if !validateNumbre(number) {
		fmt.Println("Invalid input data")
		return
	}

	switch number {
	case 1:
		fmt.Println("Monday")
	case 2:
		fmt.Println("Tuesday")
	case 3:
		fmt.Println("Wednesday")
	case 4:
		fmt.Println("Thursday")
	case 5:
		fmt.Println("Friday")
	case 6:
		fmt.Println("Saturday")
	case 7:
		fmt.Println("Sunday")
	}
}
