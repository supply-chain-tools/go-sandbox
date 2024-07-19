package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/gitsearch"
	"github.com/supply-chain-tools/go-sandbox/search"
	"log"
	"log/slog"
	"os"
	"strings"
)

const usage = `SYNOPSIS
    gitsearch [OPTIONS] [SEARCH_TERM...]

SEARCH OPTIONS
	--type=TYPE
		Type of search terms: 'package', 'url', 'domain', 'input' (default)
		The type will be used to match the word boundary as well as how normalization and
		typo variations are generated.

	--match=MATCH
		How search terms are matched: 'exact', 'typo', 'normalized' (default), 'all'
		Normalization depends on the type:
		  - All types will be lowercased.
		  - 'package' will normalize one or more consecutive '.', '-' and '_' into one '-'.

	--anchor-beginning
		Matches must start at a word boundary. What is considered a word depends on --type.

	--anchor-end
		Matches must end at a word boundary. What is considered a word depends on --type.

	--search-terms-file=FILE
		File containing '\n' separated search terms. Words only (regex is not supported).
		When using 'exact' matching the casing matters, otherwise the search term will be normalized.

	--exclude-words=WORD,...
		Exclude words from matching.

	--exclude-words-file=FILE
		File containing '\n' separated words to exclude from matching.

	--include-paths=PATTERN,...
		Comma-separated list of path PATTERNs to include:
		 - PATTERN with leading '/' are interpreted as absolute paths inside each repo.
		 - PATTERN with leading and trailing '/' are interpreted as absolute directories (not recursive) in each repo.
		 - PATTERN with leading '.' will match files ending with PATTERN (either extension or hidden files, only one dot allowed)
		 - Other PATTERNs will match filenames (including extension).

	--include-paths-file=FILE
		File containing '\n' separated paths to include.

	--exclude-paths=PATH,...
		Comma-separated list of paths to exclude.

	--exclude-paths-file=FILE
		File containing '\n' delimited paths to include.

	--include-repos=PATH,...
		List of repos to include. Relative to where the command is run.

	--include-repos-file=FILE
		File containing '\n' delimited repos to include. Relative to where the command is run.

	--exclude-repos=PATH,...
		List of repos to exclude. Relative to where the command is run.

	--exclude-repos-file=FILE
		File containing '\n' delimited repos to exclude. Relative to where the command is run.

	--all-history
		Search the entire history of each repo, including dangling commits and blobs.

	--all-branches
		Search the the tip of each branch.

	--dorks=PATH:SEARCH_TERM,...
		Each dork is a file path and search term separated by ':'

	--dorks-file=FILE
		File containing '\n' delimited dorks. Each dork is a file path and search term separated by ':'.

	--threads=NUM
		Set the number of threads to use.

OUTPUT OPTIONS
	--only-matching
		Only show the match, not the surrounding context.

	--no-path
		Don't show repo and filename path.

	--heading
		Output the file path before the matches rather than on each line.

	--show-search-term
		Show the search term that was matched.

	--line-number
		Show the line number that was matched.

	--column
		Show the column the match begins.

	--after-context=NUM
		Show NUM lines after match. Note that too long outputs are still truncated.

	--before-context=NUM
		Show NUM lines before match. Note that too long outputs are still truncated.

	--context=NUM	
		Show NUM lines before and after match. Note that too long outputs are still truncated.

	--context-columns=NUM
		Set the number of columns to add as context before and after a match.

	--color
		Display colors in output.

	--json
		Show results as JSON.

	--stats
		Add stats about the search to the end of the output.

OTHER OPTIONS
	--help, -h
		Print the help output.

	--debug
		Turn on debug logging.

	--output-search-strings
		Output the typo variations that was generated and exit.

Search for all occurrences of 'FooBar' (normalized)
    $ gitsearch FooBar

Search for exact match of 'FooBar' (including casing), don't match 'FooBar' as a substring of a longer word
    $ gitsearch --match exact --anchor-beginning --anchor-end FooBar

Search only for typos of 'FooBar'
    $ gitsearch --match typo FooBar`

func main() {
	flag.Usage = func() {
		fmt.Println(usage)
	}

	var (
		matchCharacters, characterSet, searchTermsFile, excludeWordsFile, excludeWordsList, excludeReposList,
		excludeReposFile, includeReposList, includeReposFile, excludePathsList, excludePathsFile, includePathsList,
		includePathsFile, dorksList, dorksFile string
		anchorBeginning, anchorEnd, debugMode, stats, displayLineNumber,
		hideRepoAndFilename, onlyMatching, displayColor, displayColumn, displayKeyword, help, h,
		outputSearchStrings, allHistory, allBranches, heading, outputJsonLines bool
		numThreads, numContext, linesBeforeContext, linesAfterContext, contextColumns int
	)
	flags := flag.NewFlagSet("all", flag.ExitOnError)
	flags.StringVar(&characterSet, "type", "input", "")
	flags.StringVar(&matchCharacters, "match", "normalized", "")

	flags.BoolVar(&anchorBeginning, "anchor-beginning", false, "")
	flags.BoolVar(&anchorEnd, "anchor-end", false, "")

	flags.StringVar(&searchTermsFile, "search-terms-file", "", "")

	flags.StringVar(&excludeWordsList, "exclude-words", "", "")
	flags.StringVar(&excludeWordsFile, "exclude-words-file", "", "")

	flags.StringVar(&includePathsList, "include-paths", "", "")
	flags.StringVar(&includePathsFile, "include-paths-file", "", "")

	flags.StringVar(&excludePathsList, "exclude-paths", "", "")
	flags.StringVar(&excludePathsFile, "exclude-paths-file", "", "")

	flags.StringVar(&includeReposList, "include-repos", "", "")
	flags.StringVar(&includeReposFile, "include-repos-file", "", "")

	flags.StringVar(&excludeReposList, "exclude-repos", "", "")
	flags.StringVar(&excludeReposFile, "exclude-repos-file", "", "")

	flags.BoolVar(&allHistory, "all-history", false, "")
	flags.BoolVar(&allBranches, "all-branches", false, "")

	flags.StringVar(&dorksList, "dorks", "", "")
	flags.StringVar(&dorksFile, "dorks-file", "", "")

	flags.IntVar(&numThreads, "threads", 0, "")

	flags.BoolVar(&onlyMatching, "only-matching", false, "")
	flags.BoolVar(&hideRepoAndFilename, "no-path", false, "")
	flags.BoolVar(&heading, "heading", false, "")
	flags.BoolVar(&outputJsonLines, "json", false, "")
	flags.BoolVar(&displayKeyword, "show-search-term", false, "")
	flags.BoolVar(&displayLineNumber, "line-number", false, "")
	flags.BoolVar(&displayColumn, "column", false, "")

	flags.IntVar(&numContext, "context", 0, "")
	flags.IntVar(&linesAfterContext, "after-context", 0, "")
	flags.IntVar(&linesBeforeContext, "before-context", 0, "")
	flags.IntVar(&contextColumns, "context-columns", 50, "")

	flags.BoolVar(&displayColor, "color", false, "")
	flags.BoolVar(&debugMode, "debug", false, "")
	flags.BoolVar(&stats, "stats", false, "")
	flags.BoolVar(&outputSearchStrings, "output-search-strings", false, "")

	flags.BoolVar(&help, "help", false, "")
	flags.BoolVar(&h, "h", false, "")

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

	searchTerms := flags.Args()

	var searchType search.Match
	switch matchCharacters {
	case "all":
		searchType = search.MatchNormalizedAndTypo
	case "exact":
		searchType = search.MatchExact
	case "typo":
		searchType = search.MatchTypoOnly
	case "normalized":
		searchType = search.MatchNormalized
	default:
		fmt.Println(usage)
		os.Exit(1)
	}

	var selectedCharacterSet search.CharacterSet
	switch characterSet {
	case "package":
		selectedCharacterSet = search.CharacterSetPackage
	case "url":
		selectedCharacterSet = search.CharacterSetUrl
	case "domain":
		selectedCharacterSet = search.CharacterSetDomain
	case "input":
		selectedCharacterSet = search.CharacterSetInput
	default:
		fmt.Println(usage)
		os.Exit(1)
	}

	slog.Debug("search options", "type", selectedCharacterSet,
		"match", searchType,
		"anchor-beginning", anchorBeginning,
		"anchor-end", anchorEnd,
		"all-history", allHistory,
		"all-branches", allBranches,
		"threads", numThreads)

	slog.Debug("output options", "only-matching", onlyMatching,
		"no-path", hideRepoAndFilename,
		"heading", heading,
		"show-search-term", displayKeyword,
		"line-number", displayLineNumber,
		"column", displayColumn,
		"context", numContext,
		"before-context", linesBeforeContext,
		"after-context", linesAfterContext,
		"color", displayColor,
		"json", outputJsonLines,
		"output-search-strings", outputSearchStrings)

	if allHistory && allBranches {
		fmt.Println("--all-history and --all-branches cannot be used together")
		os.Exit(1)
	}

	mode := gitkit.ModeAllFiles

	if allBranches {
		mode = gitkit.ModeAllBranches
	} else if allHistory {
		mode = gitkit.ModeAllHistory
	}

	file, err := os.Stdin.Stat()
	if err != nil {
		log.Fatal(err)
	}

	if !(file.Mode()&os.ModeNamedPipe == 0) {
		slog.Debug("reading keywords from stdin")
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			searchTerms = append(searchTerms, strings.Trim(scanner.Text(), "\n "))
		}
	}

	excludeWords := readFromList(excludeWordsList)
	if excludeWordsFile != "" {
		slog.Debug("exclude from file", "path", excludeWordsFile)
		excludeWords = append(excludeWords, readFromFile(excludeWordsFile)...)
	}

	includeRepos := readFromList(includeReposList)
	if includeReposFile != "" {
		slog.Debug("include repos", "path", includeReposFile)
		includeRepos = append(includeRepos, readFromFile(includeReposFile)...)
	}

	excludeRepos := readFromList(excludeReposList)
	if excludeReposFile != "" {
		slog.Debug("exclude repos", "path", excludeReposFile)
		excludeRepos = append(excludeRepos, readFromFile(excludeReposFile)...)
	}
	slog.Debug("repos", "include", strings.Join(includeRepos, ","),
		"exclude", strings.Join(excludeRepos, ","))

	includePaths := readFromList(includePathsList)
	if includePathsFile != "" {
		slog.Debug("include paths file", "path", includePathsFile)
		includePaths = append(includePaths, readFromFile(includePathsFile)...)
	}
	slog.Debug("include paths", "paths", strings.Join(includePaths, ","))

	excludePaths := readFromList(excludePathsList)
	if excludePathsFile != "" {
		slog.Debug("exclude paths file", "path", excludePathsFile)
		excludePaths = append(excludePaths, readFromFile(excludePathsFile)...)
	}
	slog.Debug("excluded paths", "paths", strings.Join(excludePaths, ","))

	inputDorks := readFromList(dorksList)
	if dorksFile != "" {
		slog.Debug("dorks file", "path", dorksFile)
		inputDorks = append(inputDorks, readFromFile(dorksFile)...)
	}
	slog.Debug("dorks", "input", strings.Join(inputDorks, ","))
	dorks := parseDorks(inputDorks)

	if searchTermsFile != "" {
		slog.Debug("include from file", "path", searchTermsFile)
		searchTerms = append(searchTerms, readFromFile(searchTermsFile)...)
	}

	if numThreads < 0 {
		fmt.Println("The number of threads must be a positive number or 0 (default)")
		os.Exit(1)
	}

	if numContext < 0 {
		fmt.Println("Context must be a non-negative number")
		os.Exit(1)
	}

	if linesBeforeContext < 0 {
		fmt.Println("Before context must be a non-negative number")
		os.Exit(1)
	}

	if linesAfterContext < 0 {
		fmt.Println("After context must be a non-negative number")
		os.Exit(1)
	}

	if numContext > 0 && linesBeforeContext > 0 {
		fmt.Println("Cannot set context and before context at the same time")
		os.Exit(1)
	}

	if numContext > 0 && linesAfterContext > 0 {
		fmt.Println("Cannot set context and after context at the same time")
		os.Exit(1)
	}

	if numContext > 0 {
		linesAfterContext = numContext
		linesBeforeContext = numContext
	}

	if contextColumns < 1 {
		fmt.Println("Context columns must be a positive number")
		os.Exit(1)
	}

	basePath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	candidateRepos, err := gitkit.InferReposFromPath(basePath)
	if err != nil {
		log.Fatal(err)
	}

	if len(searchTerms) > 0 && len(dorks) > 0 {
		fmt.Println("cannot search both dorks and regular terms")
		os.Exit(1)
	}

	if len(searchTerms) == 0 && len(dorks) == 0 {
		fmt.Println("no keywords to search for")
		os.Exit(1)
	} else {
		slog.Debug("searching", "include", strings.Join(searchTerms, ","),
			"exclude", strings.Join(excludeWords, ", "))
	}

	searchParameters := search.NewParameters(searchType, selectedCharacterSet)
	searchParameters.SetAnchorBeginning(anchorBeginning)
	searchParameters.SetAnchorEnd(anchorEnd)
	searchParameters.SetIncludeContextInResult(!onlyMatching)
	searchParameters.SetLinesBeforeContext(linesBeforeContext)
	searchParameters.SetLinesAfterContext(linesAfterContext)
	searchParameters.SetContextColumns(contextColumns)

	config := gitsearch.NewConfig(searchTerms,
		searchParameters,
		mode,
		candidateRepos,
		numThreads,
		excludeWords,
		includeRepos,
		excludeRepos,
		includePaths,
		excludePaths,
		stats,
		heading,
		displayLineNumber,
		displayColumn,
		displayKeyword,
		displayColor,
		hideRepoAndFilename,
		onlyMatching,
		outputSearchStrings,
		outputJsonLines,
		dorks)

	gitsearch.SearchLocalRepos(config)
}

func readFromList(list string) []string {
	if len(list) == 0 {
		return make([]string, 0)
	}
	return strings.Split(list, ",")
}

func parseDorks(dorks []string) []gitsearch.Dork {
	result := make([]gitsearch.Dork, 0)
	for _, dork := range dorks {
		parts := strings.Split(dork, ":")
		if len(parts) != 2 {
			fmt.Printf("dorks have to be on the form 'path:searchTerm', got '%s'\n", dork)
			os.Exit(1)
		}

		path := parts[0]
		searchTerm := parts[1]

		if searchTerm == "" {
			fmt.Printf("the search term cannot be empty '%s'\n", dork)
			os.Exit(1)
		}

		var pathType gitsearch.PathType
		if path != "" {
			if strings.HasPrefix(path, "/") {
				if strings.HasSuffix(path, "/") {
					pathType = gitsearch.PathTypeDirectory
				} else {
					pathType = gitsearch.PathTypeExact
				}
				path = strings.Trim(path, "/")
			} else if strings.HasPrefix(path, ".") {
				if strings.Contains(path[1:], ".") {
					fmt.Printf("path can only contain a leading '.' (%s)", path)
					os.Exit(1)
				}
				pathType = gitsearch.PathTypeExtension
			} else {
				pathType = gitsearch.PathTypeFilename
			}
		} else {
			pathType = gitsearch.PathTypeAnything
		}

		result = append(result, gitsearch.NewDork(path, pathType, searchTerm))
	}

	return result
}

func readFromFile(inputPath string) []string {
	path := inputPath

	if !strings.HasPrefix(inputPath, "/") {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		path = wd + "/" + inputPath
	}

	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	result := make([]string, 0)
	for scanner.Scan() {
		result = append(result, strings.Trim(scanner.Text(), "\n "))
	}

	return result
}
