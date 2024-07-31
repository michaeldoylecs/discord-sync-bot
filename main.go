package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("%s\n%s\n", "Error loading .env file", err)
		os.Exit(1)
	}

	fmt.Println("Hello World!")
}
