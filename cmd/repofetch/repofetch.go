package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"log/slog"

	"github.com/supply-chain-tools/go-sandbox/gitkit"
)

const usage = `Usage:
  repofetch [options] <path>...

Options:
  --gh-auth    Use GitHub CLI for authentication
  --debug      Enable debug logging
  -h, --help   Display help

Environment Variables:
  GITHUB_TOKEN  GitHub token (optional)

Notes:
  If no token is provided, repositories will be cloned unauthenticated.

Examples:
  Fetch all repos for one org/user:
    $ repofetch github.com/torvalds

  Fetch one repo:
    $ repofetch github.com/torvalds/linux`

func main() {
	useGitHubAuth, debug, args := parseFlagsAndArgs()

	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}

	logOptions := &slog.HandlerOptions{
		Level: logLevel,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, logOptions))
	slog.SetDefault(logger)

	token, err := getToken(useGitHubAuth)
	if err != nil {
		slog.Debug("Failed to get token", "error", err)
	}

	client, err := createGitHubClient(token)
	if err != nil {
		slog.Debug("Failed to create GitHub client", "error", err)
	}

	if err := fetchRepositories(client, args); err != nil {
		slog.Debug("Failed to fetch repositories", "error", err)
		os.Exit(1)
	}
}

func getToken(useGitHubAuth bool) (token string, err error) {
	if useGitHubAuth {
		token, err = getTokenFromCLI()
		if err != nil {
			return "", fmt.Errorf("failed to get token from CLI: %w", err)
		}
	} else {
		token, err = getTokenFromEnv("GITHUB_TOKEN")
		if err != nil {
			return "", fmt.Errorf("failed to get token from environment: %w", err)
		}
	}
	return token, err
}

func getTokenFromEnv(name string) (string, error) {
	token := os.Getenv(name)
	if token == "" {
		return "", fmt.Errorf("could not get token from environment variable %s", name)
	}
	return token, nil
}

func getTokenFromCLI() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error executing GitHub CLI `gh auth token`: %s, details: %w", strings.TrimSpace(string(output)), err)
	}
	return strings.TrimSpace(string(output)), nil
}

func createGitHubClient(token string) (*gitkit.GitHubClient, error) {
	if token == "" {
		slog.Debug("Using unauthenticated client")
		return gitkit.NewGitHubClient(), nil
	}
	slog.Debug("Using authenticated client")
	return gitkit.NewAuthenticatedGitHubClient(token), nil
}

func parseFlagsAndArgs() (bool, bool, []string) {
	var ghAuth, debug bool

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
	}

	help := flag.Bool("help", false, "")
	flag.BoolVar(help, "h", false, "")
	flag.BoolVar(&debug, "debug", false, "")
	flag.BoolVar(&ghAuth, "gh-auth", false, "")

	flag.Parse()
	args := flag.Args()

	if *help || len(args) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	return ghAuth, debug, args
}

func fetchRepositories(client *gitkit.GitHubClient, paths []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	for _, path := range paths {
		owner, repoName, err := gitkit.ExtractOwnerAndRepoName(path)
		if err != nil {
			return fmt.Errorf("failed to extract owner and repo name for path %s: %w", path, err)
		}

		if err := client.CloneOrFetchAllRepos(owner, repoName, dir); err != nil {
			return fmt.Errorf("failed to clone or fetch repos for %s/%s: %w", owner, *repoName, err)
		}
	}

	return nil
}
