package gitsearch

import (
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"log"
	"strings"
)

type PathType string

const (
	PathTypeExtension PathType = "extension"
	PathTypeDirectory PathType = "directory"
	PathTypeExact     PathType = "exact"
	PathTypeFilename  PathType = "filename"
	PathTypeAnything  PathType = "anything"
)

type pathFilter struct {
	hasFilters   bool
	includePaths hashset.Set[string]
	excludePaths hashset.Set[string]

	includeDirectories hashset.Set[string]
	excludeDirectories hashset.Set[string]

	includeExts hashset.Set[string]
	excludeExts hashset.Set[string]

	includeFiles hashset.Set[string]
	excludeFiles hashset.Set[string]
}

func newPathFilter(includePathsList []string, excludePathsList []string) *pathFilter {
	includeExts := hashset.New[string]()
	excludeExts := hashset.New[string]()
	includePaths := hashset.New[string]()
	excludePaths := hashset.New[string]()
	includeFiles := hashset.New[string]()
	excludeFiles := hashset.New[string]()
	includeDirectories := hashset.New[string]()
	excludeDirectories := hashset.New[string]()

	hasFilters := len(includePathsList) > 0 || len(excludePathsList) > 0

	for _, includePath := range includePathsList {
		if strings.HasPrefix(includePath, "/") {
			if strings.HasSuffix(includePath, "/") {
				includeDirectories.Add(strings.Trim(includePath, "/"))
				continue
			}

			includePaths.Add(strings.Trim(includePath, "/"))
			continue
		}

		if strings.HasPrefix(includePath, ".") {
			if strings.Contains(includePath[1:], ".") {
				log.Fatalf("include extension pattern can only contain a leading '.' (%s)", includePath)
			}
			includeExts.Add(includePath)
			continue
		}

		includeFiles.Add(includePath)
	}

	for _, excludePath := range excludePathsList {
		if strings.HasPrefix(excludePath, "/") {
			if strings.HasSuffix(excludePath, "/") {
				excludeDirectories.Add(strings.Trim(excludePath, "/"))
				continue
			}

			excludePaths.Add(strings.Trim(excludePath, "/"))
			continue
		}

		if strings.HasPrefix(excludePath, ".") {
			if strings.Contains(excludePath[1:], ".") {
				log.Fatalf("exclude extension pattern can only contain a leading '.' (%s)", excludePaths)
			}
			excludeExts.Add(excludePath)
			continue
		}

		excludeFiles.Add(excludePath)
	}

	return &pathFilter{
		hasFilters:         hasFilters,
		includeExts:        includeExts,
		excludeExts:        excludeExts,
		includeFiles:       includeFiles,
		excludeFiles:       excludeFiles,
		includePaths:       includePaths,
		excludePaths:       excludePaths,
		includeDirectories: includeDirectories,
		excludeDirectories: excludeDirectories,
	}
}
