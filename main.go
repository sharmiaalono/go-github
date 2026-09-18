package main

import (
	"fmt"

	"github.com/sharmiaalono/go-github/github"
)

func main() {
	opts := &github.ListOptions{
		Page:    1,
		PerPage: 0,
	}

	urlStr, err := github.AddOptions("https://api.github.com/user/repos", opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Constructed URL: %s\n", urlStr)
}
