package main

import (
	"fmt"

	"github.com/supply-chain-tools/go-sandbox/gitkit"
)

func main() {
	repoPaths := []string{
		"github.com/torvalds", // all repos for user/org
		"https://github.com/golang/go",
		"github.com/kubernetes/kubernetes.git",
	}

	client := gitkit.NewGitHubClient()

	for _, path := range repoPaths {
		repos, err := client.GetRepositories(path)
		if err != nil {
			fmt.Printf("Failed to list repositories for path %s: %v\n", path, err)
			continue
		}

		for _, repoURL := range repos {
			result, err := client.CloneOrFetchRepo(repoURL, nil, nil)
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Result:", result)
			}
		}
	}
}
