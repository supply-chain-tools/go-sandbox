package main

import (
	"flag"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/supply-chain-tools/go-sandbox/pr"
	"log/slog"
	"os"
	"regexp"
	"strings"
)

const usage = `TBD`

const (
	hashRegex = "[0-9a-f]{40}"
)

func main() {
	optionsAndArgs, err := parseOptionsAndArgs()
	if err != nil {
		print(err.Error(), "\n")
		os.Exit(1)
	}

	switch optionsAndArgs.command {
	case create:
		err = pr.Create(optionsAndArgs.tag, optionsAndArgs.hash, optionsAndArgs.branch, optionsAndArgs.message)
		if err != nil {
			print(err.Error(), "\n")
			os.Exit(1)
		}
	case approve:
		err = pr.Approve(optionsAndArgs.tag, optionsAndArgs.hash, optionsAndArgs.message)
		if err != nil {
			print(err.Error(), "\n")
			os.Exit(1)
		}
	case merge:
		err = pr.Merge(optionsAndArgs.hash, optionsAndArgs.message)
		if err != nil {
			print(err.Error(), "\n")
			os.Exit(1)
		}
	default:
		print("Unknown command: ", optionsAndArgs.command, "\n")
		os.Exit(1)
	}
}

type optionsAndArgs struct {
	command Command
	hash    plumbing.Hash
	branch  string
	message string
	tag     string
}

type Command string

const (
	create  Command = "create"
	approve Command = "approve"
	merge   Command = "merge"
)

func parseOptionsAndArgs() (*optionsAndArgs, error) {
	flag.Usage = func() {
		fmt.Println(usage)
	}

	flags := flag.NewFlagSet("all", flag.ExitOnError)
	var help, h, debugMode bool
	var hashInput, branchInput, messageInput, tagInput string

	flags.BoolVar(&help, "help", false, "")
	flags.BoolVar(&h, "h", false, "")
	flags.BoolVar(&debugMode, "debug", false, "")
	flags.StringVar(&hashInput, "hash", "", "")
	flags.StringVar(&branchInput, "branch", "", "")
	flags.StringVar(&messageInput, "message", "", "")
	flags.StringVar(&tagInput, "tag", "", "")

	err := flags.Parse(os.Args[1:])
	if err != nil || help || h {
		fmt.Println(usage)
		os.Exit(0)
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if debugMode {
		opts.Level = slog.LevelDebug
	}

	if len(flags.Args()) == 0 {
		print(usage)
		os.Exit(1)
	}

	if len(flags.Args()) != 1 {
		return nil, fmt.Errorf("only one command expected, got {%s}", strings.Join(flags.Args(), ", "))
	}

	commandString := flags.Args()[0]
	command, err := getCommand(commandString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %s", commandString)
	}

	match, _ := regexp.MatchString(hashRegex, hashInput)
	if !match {
		return nil, fmt.Errorf("invalid commit '%s': it must be hex encoded", hashInput)
	}
	hash := plumbing.NewHash(hashInput)

	if command == create || command == approve {
		// FIXME verify tag characters
		if tagInput == "" {
			return nil, fmt.Errorf("a tag must be specified")
		}
	}

	if command == create {
		// FIXME verify branch characters
		if branchInput == "" {
			return nil, fmt.Errorf("a branch must be specified")
		}
	}

	// FIXME verify message characters

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	return &optionsAndArgs{
		command: command,
		hash:    hash,
		branch:  branchInput,
		message: messageInput,
		tag:     tagInput,
	}, nil
}

func getCommand(commandString string) (Command, error) {
	switch commandString {
	case string(create):
		return create, nil
	case string(approve):
		return approve, nil
	case string(merge):
		return merge, nil
	default:
		return "", fmt.Errorf("unknown command: %s", commandString)
	}
}
