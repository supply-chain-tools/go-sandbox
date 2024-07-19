package gitsearch

import (
	"encoding/json"
	"fmt"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"github.com/supply-chain-tools/go-sandbox/search"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func SearchLocalRepos(config Config) {
	setupStart := time.Now()

	compiledSearch := search.New(config.SearchTerms(), config.SearchParameters())

	dorkSearches := make([]*dorkSearch, 0)
	anySearchTerms := make([]string, 0)
	for _, dork := range config.Dorks() {
		if dork.PathType() == PathTypeAnything {
			anySearchTerms = append(anySearchTerms, dork.SearchTerm())
		} else {
			dorkSearches = append(dorkSearches, &dorkSearch{
				search:   search.New([]string{dork.SearchTerm()}, config.SearchParameters()),
				path:     dork.Path(),
				pathType: dork.PathType(),
			})
		}
	}
	if len(anySearchTerms) > 0 {
		dorkSearches = append(dorkSearches, &dorkSearch{
			search:   search.New(anySearchTerms, config.SearchParameters()),
			path:     "",
			pathType: PathTypeAnything,
		})
	}

	var numVariations, numNodes int
	var trieSizeInBytes uint64
	if config.DisplayStats() {
		numVariations, numNodes, trieSizeInBytes = compiledSearch.Size()
	}

	if config.OutputSearchStrings() {
		for _, s := range compiledSearch.Variations() {
			fmt.Println(s)
		}
		os.Exit(0)
	}

	repos := make([]gitkit.Repository, 0)

	includeReposSet := hashset.New[string]()
	for _, k := range config.IncludeRepos() {
		includeReposSet.Add(k)
	}

	excludeReposSet := hashset.New[string]()
	for _, k := range config.ExcludeRepos() {
		excludeReposSet.Add(k)
	}

	pathFilter := newPathFilter(config.IncludePaths(), config.ExcludePaths())

	for _, repo := range config.Repos() {
		cmp := repo.OrganizationName() + "/" + repo.RepositoryName()
		if repo.OrganizationName() == "" {
			cmp = repo.RepositoryName()
		}

		if excludeReposSet.Contains(cmp) {
			continue
		}

		if includeReposSet.Size() > 0 && !includeReposSet.Contains(cmp) {
			continue
		}

		repos = append(repos, repo)
	}

	if len(repos) == 0 {
		log.Fatal("no repos to search")
	}

	red := "\033[31m"
	green := "\033[32m"
	yellow := "\033[33m"
	blue := "\033[34m"
	magneta := "\033[35m"
	underline := "\033[4m"
	reset := "\033[0m"
	trimmed := red + "..." + reset

	if !config.DisplayColor() {
		red = ""
		magneta = ""
		yellow = ""
		green = ""
		underline = ""
		reset = ""
		blue = ""
		trimmed = ""
	}

	excludeSet := hashset.New[string]()
	for _, excludeKeyword := range config.ExcludeWords() {
		excludeSet.Add(strings.ToLower(excludeKeyword))
	}

	newSearcher := func(repo gitkit.Repository) gitkit.Searcher[search.Result] {
		return newGitSearchSearcher(repo, compiledSearch, dorkSearches, pathFilter)
	}

	setupElapsed := time.Since(setupStart)
	start := time.Now()
	resultsChannel, workerStats := gitkit.Search(repos, newSearcher, config.Mode(), config.Concurrency())

	filesMatched := 0
	matches := 0
	firstResult := true
	for workerResult := range resultsChannel {
		lastPath := ""
		for _, result := range workerResult.RepoResults {
			filteredMatches := make([]search.Result, 0)
			for _, r := range result.Results {
				candidate := strings.ToLower(r.MatchedText())
				if !excludeSet.Contains(candidate) {
					filteredMatches = append(filteredMatches, r)
				}
			}

			if len(filteredMatches) == 0 {
				continue
			}

			repo := ""
			if workerResult.Repo.RepositoryName() != "" {
				if workerResult.Repo.OrganizationName() != "" {
					repo = workerResult.Repo.OrganizationName() + "/" + workerResult.Repo.RepositoryName() + "/"
				} else {
					repo = workerResult.Repo.RepositoryName() + "/"
				}
			}

			filesMatched++
			for _, r := range filteredMatches {
				if config.OutputJsonLines() {
					resultOutput := ResultOutput{
						MatchedText:   r.MatchedText(),
						SearchTerm:    r.MatchedKeyword().Original(),
						ContextBefore: r.ContextBefore(),
						ContextAfter:  r.ContextAfter(),
						LineNumber:    r.LineNumber(),
						Column:        r.StartOfWord() - r.StartOfLine() + 1,
						Path:          result.Path,
						Repo:          workerResult.Repo.RepositoryName(),
						Org:           workerResult.Repo.OrganizationName(),
					}

					serializedOutput, err := json.Marshal(&resultOutput)
					if err != nil {
						log.Fatal(err)
					}
					fmt.Println(string(serializedOutput))
				} else {
					lineNumber := ""
					if config.DisplayLineNumber() {
						lineNumber = yellow + strconv.FormatUint(uint64(r.LineNumber()), 10) + reset + ":"
					}

					column := ""
					if config.DisplayColumn() {
						column = yellow + strconv.FormatUint(uint64(r.StartOfWord()-r.StartOfLine()+1), 10) + reset + ":"
					}

					matchedKeyword := ""
					if config.DisplayKeyword() {
						matchedKeyword = yellow + r.MatchedKeyword().Original() + reset + ":"
					}

					var repoAndFile string
					if config.HideRepoAndFilename() {
						repoAndFile = ""
					} else {
						repoAndFile = red + repo + magneta + result.Path + reset + ":"
					}

					var match string

					pre := r.StartOfMatch() - r.StartOfWord()
					post := r.EndOfWord() - r.EndOfMatch()
					if config.OnlyMatching() {
						match = green + r.MatchedText()[:pre] + underline + r.MatchedText()[pre:len(r.MatchedText())-post] + reset + green + r.MatchedText()[len(r.MatchedText())-post:]
					} else {
						trimmedLeft := ""
						if r.TrimmedLeft() {
							trimmedLeft = trimmed
						}

						trimmedRight := ""
						if r.TrimmedRight() {
							trimmedRight = trimmed
						}

						match = trimmedLeft + r.ContextBefore() + green + r.MatchedText()[:pre] + underline +
							r.MatchedText()[pre:len(r.MatchedText())-post] + reset + green + r.MatchedText()[len(r.MatchedText())-post:] +
							reset + r.ContextAfter() + trimmedRight
					}

					if config.DisplayHeading() {
						if lastPath != result.Path {
							if !firstResult {
								fmt.Println("")
							}
							fmt.Printf("%s\n", repoAndFile)
							lastPath = result.Path
						}
						repoAndFile = ""
						firstResult = false
					}
					fmt.Printf("%s%s%s%s%s\n", repoAndFile, lineNumber, column, matchedKeyword, match)

					if result.Matches != nil {
						for _, branchResult := range result.Matches[r] {
							sr := ""
							onHead := " "
							if branchResult.OnTip() {
								onHead = "^"
							}
							if len(branchResult.FirstCommit()) == 1 && branchResult.LastCommit() == branchResult.FirstCommit()[0] {
								last := branchResult.LastCommit().Committer.When.Format(time.DateOnly)
								lastHash := branchResult.LastCommit().Hash.String()[:6]

								sr = fmt.Sprintf("%s%s%s[%s:%s]%s", blue, onHead, branchResult.Name(), last, lastHash, reset)
							} else {
								last := branchResult.LastCommit().Committer.When.Format(time.DateOnly)
								lastHash := branchResult.LastCommit().Hash.String()[:6]

								firsts := make([]string, 0)
								for _, first := range branchResult.FirstCommit() {
									firstTime := first.Committer.When.Format(time.DateOnly)
									firstHash := first.Hash.String()[:6]
									firsts = append(firsts, firstTime+":"+firstHash)
								}

								sr = fmt.Sprintf("%s%s%s[%s:%s <- %s]%s", blue, onHead, branchResult.Name(), last, lastHash, strings.Join(firsts, ","), reset)
							}

							fmt.Printf("  %s\n", sr)
						}
						dangling, hasDangling := result.DanglingCommits[r]
						if hasDangling {
							first := dangling.Committer.When.Format(time.DateOnly)
							firstHash := dangling.Hash.String()[:6]
							fmt.Printf("  dangling %s[%s:%s]%s\n", blue, first, firstHash, reset)
						}

						tags, hasTags := result.Tags[r]
						if hasTags {
							names := make([]string, 0)
							for _, tag := range tags {
								names = append(names, tag.Name)
							}

							fmt.Printf("  tags %s%s%s\n", blue, strings.Join(names, ","), reset)
						}
					}
				}

				matches++
			}
		}
	}
	elapsed := time.Since(start)

	totalSize := workerStats.TotalFileSize()
	numFiles := workerStats.NumberOfFiles()

	if config.DisplayStats() {
		fmt.Printf("matches: %d in %d files\n", matches, filesMatched)
		fmt.Printf("keywords: %d with %d typosquatting variations\n", len(config.SearchTerms()), numVariations)
		fmt.Printf("time: %.2fs search, %.3fs setup, %.3fs analysis, %.3fs data load, %.3fs list files\n",
			elapsed.Seconds(),
			setupElapsed.Seconds(),
			float64(workerStats.QueryTime())/1000000000.0/float64(config.Concurrency()),
			float64(workerStats.DataLoadTime())/1000000000.0/float64(config.Concurrency()),
			float64(workerStats.ListFilesTime())/1000000000.0/float64(config.Concurrency()))
		fmt.Printf("throughput: %.0f MB/s, %.0f files/s, %.0f repos/s\n",
			float64(totalSize)/1000000.0/elapsed.Seconds(),
			float64(numFiles)/elapsed.Seconds(),
			float64(len(repos))/elapsed.Seconds())
		fmt.Printf("total: %.0f MB, %d files, %d repos\n", float64(totalSize)/1000000.0, numFiles, len(repos))
		fmt.Printf("trie: %d nodes, %.0f MB\n", numNodes, float64(trieSizeInBytes)/1000000.0)
		fmt.Printf("concurrency: %d\n", config.Concurrency())
	}
}
