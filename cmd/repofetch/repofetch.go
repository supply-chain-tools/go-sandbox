package main

import (
	"flag"
	"fmt"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const usage = `Usage:
    repofetch <path>

Currently only the path prefix 'github.com/' is supported

Fetch all repos for one org/user
    $ repofetch github.com/torvalds

Fetch one repo
    $ repofetch github.com/torvalds/linux`

const configDirectory = ".supply-chain-tools"
const githubTokenFileName = "github-token"

func main() {
	flag.Usage = func() {
		fmt.Println(usage)
	}

	flags := flag.NewFlagSet("all", flag.ExitOnError)
	var help, h, debugMode bool
	flags.BoolVar(&help, "help", false, "")
	flags.BoolVar(&h, "h", false, "")
	flags.BoolVar(&debugMode, "debug", false, "")

	err := flags.Parse(os.Args[1:])
	if err != nil || help || h {
		fmt.Println(usage)
		os.Exit(1)
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if debugMode {
		opts.Level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	var client *gitkit.GitHubClient
	token, err := readGitHubBasicAuth()
	if err != nil {
		slog.Debug("GitHub token not found, using unauthenticated client")
		client = gitkit.NewGitHubClient()
	} else {
		slog.Debug("GitHub token found")
		client = gitkit.NewAuthenticatedGitHubClient(token)
	}

	localBasePath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	args := flags.Args()
	if len(args) == 0 {
		fmt.Println(usage)
		os.Exit(1)
	}

	for _, arg := range args {
		owner, repoName, err := gitkit.ExtractOwnerAndRepoName(arg)
		if err != nil {
			log.Fatal(err)
		}

		err = client.CloneOrFetchAllRepos(owner, repoName, localBasePath)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func readGitHubBasicAuth() (string, error) {
	configDir, err := getConfigDirectory()
	if err != nil {
		return "", err
	}

	path := filepath.Join(configDir, githubTokenFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	result := strings.TrimSuffix(string(data), "\n")
	return result, nil
}

func getConfigDirectory() (string, error) {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDirectory, configDirectory), nil
}
