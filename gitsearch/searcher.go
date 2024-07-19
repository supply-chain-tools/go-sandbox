package gitsearch

import (
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/search"
	"path/filepath"
	"strings"
)

type gitSearchSearcher struct {
	search       search.Search
	dorkSearches []*dorkSearch
	pathFilter   *pathFilter
	subPath      *string
	blobResults  map[plumbing.Hash][]search.Result
}

type dorkSearch struct {
	search   search.Search
	path     string
	pathType PathType
}

func newGitSearchSearcher(repo gitkit.Repository, search search.Search, dorkSearches []*dorkSearch, pathFilter *pathFilter) *gitSearchSearcher {
	return &gitSearchSearcher{
		search:       search,
		dorkSearches: dorkSearches,
		pathFilter:   pathFilter,
		subPath:      repo.SubPath(),
	}
}

func (gs *gitSearchSearcher) Process(hash *plumbing.Hash, loadData func() []byte, path string) []search.Result {
	if !shouldProcess(path, gs.pathFilter, gs.subPath) {
		return []search.Result{}
	}

	if len(gs.dorkSearches) == 0 {
		found := false
		var blobResults []search.Result

		if hash != nil {
			blobResults, found = gs.blobResults[*hash]
		}

		if found {
			return blobResults
		} else {
			r, err := gs.search.Match(loadData())
			if err != nil {
				return []search.Result{}
			} else {
				return r
			}
		}
	} else {
		searches := convertToSearch(path, gs.dorkSearches)
		results := make([]search.Result, 0)

		if len(searches) > 0 {
			data := loadData()
			for _, compiledSearch := range searches {
				r, err := compiledSearch.Match(data)
				if err != nil {
					// ignore
				}
				results = append(results, r...)
			}
		}

		return results
	}
}

func shouldProcess(localPath string, pathFilter *pathFilter, subPath *string) bool {
	if subPath != nil && !strings.HasPrefix(localPath, *subPath) {
		return false
	}

	if pathFilter.includePaths.Size() > 0 || pathFilter.excludePaths.Size() > 0 {
		if pathFilter.excludePaths.Contains(localPath) {
			return false
		}

		if pathFilter.includePaths.Size() > 0 && !pathFilter.includePaths.Contains(localPath) {
			return false
		}
	}

	if pathFilter.includeDirectories.Size() > 0 || pathFilter.excludeDirectories.Size() > 0 {
		directory := filepath.Dir(localPath)
		if pathFilter.excludeDirectories.Contains(directory) {
			return false
		}

		if pathFilter.includeDirectories.Size() > 0 && !pathFilter.includeDirectories.Contains(directory) {
			return false
		}
	}

	if pathFilter.includeFiles.Size() > 0 || pathFilter.excludeFiles.Size() > 0 {
		filename := filepath.Base(localPath)
		if pathFilter.excludeFiles.Contains(filename) {
			return false
		}

		if pathFilter.includeFiles.Size() > 0 && !pathFilter.includeFiles.Contains(filename) {
			return false
		}
	}

	if pathFilter.includeExts.Size() > 0 || pathFilter.excludeExts.Size() > 0 {
		ext := filepath.Ext(localPath)
		if pathFilter.excludeExts.Contains(ext) {
			return false
		}

		if pathFilter.includeExts.Size() > 0 && !pathFilter.includeExts.Contains(ext) {
			return false
		}
	}

	return true
}

func convertToSearch(localPath string, dorkSearches []*dorkSearch) []search.Search {
	searches := make([]search.Search, 0)
	for _, dorkSearch := range dorkSearches {
		switch dorkSearch.pathType {
		case PathTypeExact:
			if localPath != dorkSearch.path {
				continue
			}
		case PathTypeDirectory:
			directory := filepath.Dir(localPath)
			if directory != dorkSearch.path {
				continue
			}
		case PathTypeFilename:
			filename := filepath.Base(localPath)
			if filename != dorkSearch.path {
				continue
			}
		case PathTypeExtension:
			ext := filepath.Ext(localPath)
			if ext != dorkSearch.path {
				continue
			}
		case PathTypeAnything:
			// do nothing
		}
		searches = append(searches, dorkSearch.search)
	}

	return searches
}
