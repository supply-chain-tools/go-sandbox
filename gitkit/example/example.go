package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

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

	var outputBuffer bytes.Buffer
	progressWriter := io.Writer(&outputBuffer)

	cloneOpts := gitkit.CloneOptions{
		// Depth:    1,
		Bare:     false,
		Progress: progressWriter,
	}

	// monitor output buffer and print updates
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				if outputBuffer.Len() > 0 {
					fmt.Print(outputBuffer.String())
					outputBuffer.Reset() // clear buffer on print
				}
				time.Sleep(20 * time.Millisecond) // avoid busy loop
			}
		}
	}()

	for _, path := range repoPaths {
		repos, err := client.GetRepositories(path)
		if err != nil {
			fmt.Printf("Failed to list repositories for path %s: %v\n", path, err)
			continue
		}

		for _, repoURL := range repos {
			fmt.Printf("Cloning repository: %s\n", repoURL)
			result, err := client.CloneOrFetchRepo(repoURL, dir, &cloneOpts)
			if err != nil {
				fmt.Printf("Error cloning repository %s: %v\n", repoURL, err)
			} else {
				fmt.Println("Result:", result)
			}
		}

		done <- true
		time.Sleep(100 * time.Millisecond) // wait for last print
	}
}
