package main

import (
	"fmt"
	"os"

	"github.com/supply-chain-tools/go-sandbox/gitkit"
)

func main() {
	repoPaths := []string{
		"github.com/torvalds", // all repos for user/org
		"https://github.com/golang/go",
		"github.com/kubernetes/kubernetes.git",
	}

	client := gitkit.NewGitHubClient()

	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	for _, path := range repoPaths {
		repos, err := client.GetRepositories(path)
		if err != nil {
			fmt.Printf("Failed to list repositories for path %s: %v\n", path, err)
			continue
		}

		for _, repoURL := range repos {
			fmt.Printf("Cloning repository: %s\n", repoURL)

			result, err := client.CloneOrFetchRepo(repoURL, dir, nil, nil)
			if err != nil {
				fmt.Printf("Error cloning repository %s: %v\n", repoURL, err)
			} else {
				fmt.Println("Result:", result)
			}
		}
	}
}
