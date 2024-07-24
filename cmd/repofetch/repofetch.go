package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/supply-chain-tools/go-sandbox/gitkit"
)

const usage = `Usage:
    repofetch [options] <path>

Options:
	--debug           Enable debug logging
    --github-auth     Use GitHub CLI for authentication
    -h, --help        Show help message

Environment Variables:
    GITHUB_TOKEN  GitHub token for authenticated requests

Currently, only the path prefix 'github.com/' is supported

Examples:
    Fetch all repos for one org/user:
        $ repofetch github.com/torvalds

    Fetch one repo:
        $ repofetch github.com/torvalds/linux`

func main() {
	flag.Usage = func() {
		fmt.Println(usage)
	}

	flags := flag.NewFlagSet("all", flag.ExitOnError)
	var help, h, useGh, debugMode bool
	flags.BoolVar(&help, "help", false, "")
	flags.BoolVar(&h, "h", false, "")
	flags.BoolVar(&useGh, "github-auth", false, "")
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

	client, err := setupGitHubClient(useGh)
	if err != nil {
		log.Fatalf("Failed to set up GitHub client: %v", err)
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

func setupGitHubClient(useGh bool) (*gitkit.GitHubClient, error) {
	if useGh {
		authenticated, err := checkGhAuth()
		if err != nil {
			return nil, fmt.Errorf("error checking gh authentication: %v", err)
		}
		if !authenticated {
			return nil, fmt.Errorf("you are not authenticated with GitHub CLI. Please run 'gh auth login' to authenticate")
		}
		token, err := getGitHubTokenFromGh()
		if err != nil {
			return nil, fmt.Errorf("using unauthenticated client: %v", err)
		}
		slog.Debug("GitHub token found using gh CLI")
		return gitkit.NewAuthenticatedGitHubClient(token), nil
	}

	token, err := getGitHubTokenFromEnv()
	if err != nil {
		slog.Debug(fmt.Sprintf("Using unauthenticated client: %v", err))
		return gitkit.NewGitHubClient(), nil
	}
	slog.Debug("GitHub token found")
	return gitkit.NewAuthenticatedGitHubClient(token), nil
}

func getGitHubTokenFromEnv() (string, error) {
	const envVarName = "GITHUB_TOKEN"
	token := os.Getenv(envVarName)
	if token == "" {
		return "", fmt.Errorf("GitHub token not set in environment variable %s", envVarName)
	}
	return token, nil
}

func getGitHubTokenFromGh() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub token using gh CLI: %v", err)
	}
	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("GitHub token not found using gh CLI")
	}
	return token, nil
}

func checkGhAuth() (bool, error) {
	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to check gh authentication: %v", err)
	}
	if strings.Contains(string(output), "Logged in to github.com account") {
		return true, nil
	}
	return false, nil
}
