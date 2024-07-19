package gitsearch

import (
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/search"
)

const defaultConcurrency = 9

type Config interface {
	SearchTerms() []string
	SearchParameters() search.Parameters
	Mode() gitkit.Mode
	Repos() []gitkit.Repository
	Concurrency() int
	ExcludeWords() []string
	IncludeRepos() []string
	ExcludeRepos() []string
	IncludePaths() []string
	ExcludePaths() []string
	DisplayStats() bool
	DisplayHeading() bool
	OutputJsonLines() bool
	DisplayLineNumber() bool
	DisplayColumn() bool
	DisplayKeyword() bool
	DisplayColor() bool
	HideRepoAndFilename() bool
	OnlyMatching() bool
	OutputSearchStrings() bool
	Dorks() []Dork
}

type config struct {
	searchTerms         []string
	excludeWords        []string
	includeRepos        []string
	excludeRepos        []string
	includePaths        []string
	excludePaths        []string
	searchParameters    search.Parameters
	repos               []gitkit.Repository
	mode                gitkit.Mode
	displayStats        bool
	concurrency         int
	displayHeading      bool
	outputJsonLines     bool
	displayLineNumber   bool
	displayColumn       bool
	displayKeyword      bool
	displayColor        bool
	hideRepoAndFilename bool
	onlyMatching        bool
	outputSearchStrings bool
	dorks               []Dork
}

func NewConfig(searchTerms []string,
	searchParameters search.Parameters,
	mode gitkit.Mode,
	repos []gitkit.Repository,
	concurrency int,
	excludeWords []string,
	includeRepos []string,
	excludeRepos []string,
	includePaths []string,
	excludePaths []string,
	displayStats bool,
	displayHeading bool,
	displayLineNumber bool,
	displayColumn bool,
	displayKeyword bool,
	displayColor bool,
	hideRepoAndFilename bool,
	onlyMatching bool,
	outputSearchStrings bool,
	outputJsonLines bool,
	dorks []Dork) Config {

	if concurrency < 1 {
		concurrency = defaultConcurrency
	}

	return &config{
		searchTerms:         searchTerms,
		searchParameters:    searchParameters,
		mode:                mode,
		repos:               repos,
		concurrency:         concurrency,
		excludeWords:        excludeWords,
		includeRepos:        includeRepos,
		excludeRepos:        excludeRepos,
		includePaths:        includePaths,
		excludePaths:        excludePaths,
		displayStats:        displayStats,
		displayHeading:      displayHeading,
		displayLineNumber:   displayLineNumber,
		displayColumn:       displayColumn,
		displayKeyword:      displayKeyword,
		displayColor:        displayColor,
		hideRepoAndFilename: hideRepoAndFilename,
		onlyMatching:        onlyMatching,
		outputSearchStrings: outputSearchStrings,
		outputJsonLines:     outputJsonLines,
		dorks:               dorks,
	}
}

func (c *config) SearchTerms() []string {
	return c.searchTerms
}

func (c *config) ExcludeWords() []string {
	return c.excludeWords
}

func (c *config) IncludeRepos() []string {
	return c.includeRepos
}

func (c *config) ExcludeRepos() []string {
	return c.excludeRepos
}

func (c *config) IncludePaths() []string {
	return c.includePaths
}

func (c *config) ExcludePaths() []string {
	return c.excludePaths
}

func (c *config) SearchParameters() search.Parameters {
	return c.searchParameters
}

func (c *config) Repos() []gitkit.Repository {
	return c.repos
}

func (c *config) Mode() gitkit.Mode {
	return c.mode
}

func (c *config) DisplayStats() bool {
	return c.displayStats
}

func (c *config) Concurrency() int {
	return c.concurrency
}

func (c *config) DisplayHeading() bool {
	return c.displayHeading
}

func (c *config) OutputJsonLines() bool {
	return c.outputJsonLines
}

func (c *config) DisplayLineNumber() bool {
	return c.displayLineNumber
}

func (c *config) DisplayColumn() bool {
	return c.displayColumn
}

func (c *config) DisplayKeyword() bool {
	return c.displayKeyword
}

func (c *config) DisplayColor() bool {
	return c.displayColor
}

func (c *config) HideRepoAndFilename() bool {
	return c.hideRepoAndFilename
}

func (c *config) OnlyMatching() bool {
	return c.onlyMatching
}

func (c *config) OutputSearchStrings() bool {
	return c.outputSearchStrings
}

func (c *config) Dorks() []Dork {
	return c.dorks
}

type Dork interface {
	Path() string
	PathType() PathType
	SearchTerm() string
}

type dork struct {
	path       string
	pathType   PathType
	searchTerm string
}

func NewDork(path string, pathType PathType, searchTerm string) Dork {
	return &dork{
		path:       path,
		pathType:   pathType,
		searchTerm: searchTerm,
	}
}

func (d *dork) Path() string {
	return d.path
}

func (d *dork) PathType() PathType {
	return d.pathType
}

func (d *dork) SearchTerm() string {
	return d.searchTerm
}
