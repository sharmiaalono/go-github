package main

import (
	"fmt"

	"github.com/sharmiaalono/go-github/github"
)

func main() {
	// Demonstrate PerPage=0 omission vs explicit size (addOptions is tested in-package).
	fmt.Println("go-github pagination helpers — see github/ and: go test ./...")
	_ = github.ListOptions{}
}
