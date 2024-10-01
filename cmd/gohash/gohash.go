package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/gohash"
	"log/slog"
	"os"
)

const usage = `Usage:
    gohash [options] <version>

Options:
    --debug       Enable debug logging
    -h, --help     Show help message`

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

	repo, err := loadRepoFromCwd()
	if err != nil {
		print("Failed to load repo: ", err.Error(), "\n")
		os.Exit(1)
	}

	var target plumbing.Hash
	if len(flags.Args()) == 0 {
		head, err := repo.Head()
		if err != nil {
			print("Failed to get HEAD commit: ", err.Error(), "\n")
			os.Exit(1)
		}

		target = head.Hash()
	} else {
		input := flags.Args()[0]
		isSemantic, err := gohash.IsSemanticVersion(input)
		if err != nil {
			print("Failed to check semantic version: ", err.Error(), "\n")
			os.Exit(1)
		}

		if isSemantic {
			tag, err := repo.Tag(input)
			if err != nil {
				print("Failed to get tag '", input, "': ", err.Error(), "\n")
				os.Exit(1)
			}

			tagObject, err := repo.TagObject(tag.Hash())
			if err != nil {
				if errors.Is(err, plumbing.ErrObjectNotFound) {
					target = tag.Hash()
				} else {
					print("Failed to get tag object '", input, "': ", err.Error(), "\n")
					os.Exit(1)
				}
			} else {
				target = tagObject.Target
			}
		} else {
			if len(input) != 40 {
				print("Input must be a semantic versioned tag or a 40 character hash, not: ", input, "\n")
				os.Exit(1)
			}
			target = plumbing.NewHash(input)
		}
	}

	h1s, err := gohash.GitDirHashAll(repo, target)
	if err != nil {
		print("Failed to get git dir hash: ", err.Error(), "\n")
		os.Exit(1)
	}

	fmt.Printf("== this is experimental and should not be relied on yet ==\n")
	for _, h1 := range h1s {
		fmt.Printf("# https://sum.golang.org/lookup/%s@%s\n", h1.Path, h1.Version)
		fmt.Printf("%s %s %s\n", h1.Path, h1.Version, h1.DirectoryHash)
		fmt.Printf("%s %s/go.mod %s\n", h1.Path, h1.Version, h1.GoModHash)
	}
}

func loadRepoFromCwd() (*git.Repository, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	repoDir, _, err := gitkit.GetRootPathOfLocalGitRepo(cwd)
	if err != nil {
		return nil, err
	}

	repo, err := gitkit.OpenRepoInLocalPath(repoDir)
	if err != nil {
		return nil, err
	}

	return repo, nil
}
