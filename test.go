package main

import (
	"fmt"
	"strconv"
	"strings"
)

func main() {
	var priceStr string
	var quantityStr string
	var amount float64
	var data string
	fmt.Print("data: ")
	fmt.Scanln(&data)
	fmt.Println(data)
	parts := strings.Fields(data)
	if len(parts) != 2 {
		fmt.Println("incorrect input")
		return
	}
	priceStr = parts[0]
	quantityStr = parts[1]
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		fmt.Println("error:", err)
	}
	quantity, err := strconv.ParseFloat(quantityStr, 64)
	if err != nil {
		fmt.Println("error:", err)
	}
	amount = price * quantity
	fmt.Printf("%.2f", amount)
}
