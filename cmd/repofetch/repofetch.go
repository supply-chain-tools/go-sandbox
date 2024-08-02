package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"log/slog"

	"github.com/supply-chain-tools/go-sandbox/gitkit"
)

const usage = `Usage:
  repofetch [options] <path>...

Options:
  --gh-auth      Use GitHub CLI for authentication
  --token        Set GitHub token
  --depth        Set cloning/fetching depth
  --concurrency  Set number of concurrent fetches (default: 10)
  --bare         Enable bare cloning
  --debug        Enable debug logging
  -h, --help     Display help

Environment Variables:
  GITHUB_TOKEN  GitHub token (optional)

Notes:
  If no token is provided, repositories will be cloned unauthenticated.

Examples:
  Fetch all repos for one org/user:
    $ repofetch github.com/torvalds

  Fetch one repo:
    $ repofetch github.com/torvalds/linux`

type options struct {
	token         string
	useGitHubAuth bool
	debug         bool
	depth         int
	concurrency   int
	bare          bool
}

func main() {
	args, opts := parseArgsAndOptions()

	logLevel := slog.LevelInfo
	if opts.debug {
		logLevel = slog.LevelDebug
	}

	logOptions := &slog.HandlerOptions{
		Level: logLevel,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, logOptions))
	slog.SetDefault(logger)

	slog.Debug("Running repofetch", "args", args, "options", opts)

	token, err := getToken(opts)
	if err != nil {
		slog.Debug("Failed to get token", "error", err)
	}

	client, err := createGitHubClient(token)
	if err != nil {
		slog.Debug("Failed to create GitHub client", "error", err)
	}

	if err := fetchRepositories(client, args, opts); err != nil {
		slog.Debug("Failed to fetch repositories", "error", err)
		os.Exit(1)
	}
}

func getToken(opts options) (token string, err error) {
	if opts.token != "" {
		token = opts.token
	} else if opts.useGitHubAuth {
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

func parseArgsAndOptions() ([]string, options) {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
	}

	opts := options{}

	help := flag.Bool("help", false, "")
	flag.BoolVar(help, "h", false, "")
	flag.BoolVar(&opts.debug, "debug", false, "")
	flag.BoolVar(&opts.useGitHubAuth, "gh-auth", false, "")
	flag.BoolVar(&opts.bare, "bare", false, "")
	flag.StringVar(&opts.token, "token", "", "")
	flag.IntVar(&opts.concurrency, "concurrency", 10, "")
	flag.IntVar(&opts.depth, "depth", 0, "")

	flag.Parse()
	args := flag.Args()

	if *help || len(args) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	return args, opts
}

func fetchRepositories(client *gitkit.GitHubClient, paths []string, opts options) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	cloneOpts := gitkit.CloneOptions{
		Depth: opts.depth,
		Bare:  opts.bare,
	}

	var reposToClone []string

	for _, path := range paths {
		repos, err := client.GetRepositories(path)
		if err != nil {
			return fmt.Errorf("failed to list repositories for path %s: %w", path, err)
		}
		reposToClone = append(reposToClone, repos...)
	}

	sem := make(chan struct{}, opts.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []string

	for _, repoURL := range reposToClone {
		wg.Add(1)
		sem <- struct{}{}

		go func(repoURL string) {
			defer wg.Done()
			defer func() { <-sem }()

			_, err := client.CloneOrFetchRepo(repoURL, dir, &cloneOpts)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("failed to clone or fetch repo %s: %v", repoURL, err))
				mu.Unlock()
			}
		}(repoURL)
	}

	wg.Wait()
	close(sem)

	if len(errs) > 0 {
		return fmt.Errorf("encountered errors: \n%s", strings.Join(errs, "\n"))
	}

	return nil
}
