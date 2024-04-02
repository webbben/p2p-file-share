package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// prompt the user for any string input
func ReadInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading user input:", err)
		return ""
	}
	return strings.TrimSpace(input)
}

// prompt the user for a yes or no answer. will return true for (any case of) "yes" or "y".
func YorN(prompt string) bool {
	if prompt != "" {
		fmt.Println(prompt)
	}
	input := ReadInput()
	if strings.ToLower(input) == "yes" || strings.ToLower(input) == "y" {
		return true
	}
	return false
}
