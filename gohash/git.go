package gohash

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/sumdb/dirhash"
	"io"
	"log"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const tagRegex = "^v(?P<Major>[0-9]+)\\.(?P<Minor>[0-9]+)\\.(?P<Patch>[0-9]+)$"

func GitDirHashAll(repo *git.Repository, hash plumbing.Hash) ([]*H1, error) {
	return GitDirHashAllVersion(repo, hash, nil)
}

func GitDirHashAllVersion(repo *git.Repository, hash plumbing.Hash, versionOverride *string) ([]*H1, error) {
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	modDirectories := make([]string, 0)
	goDotMod := "go.mod"
	err = tree.Files().ForEach(func(file *object.File) error {
		name := file.Name

		if strings.HasSuffix(name, goDotMod) {
			modDirectories = append(modDirectories, strings.TrimSuffix(name, goDotMod))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(modDirectories)

	result := make([]*H1, 0, len(modDirectories))
	for _, modDirectory := range modDirectories {
		r, err := GitDirHash(repo, hash, modDirectory, versionOverride)
		if err != nil {
			return nil, err
		}

		result = append(result, r)
	}

	return result, nil
}

func IsSemanticVersion(version string) (bool, error) {
	match, err := regexp.MatchString(tagRegex, version)
	if err != nil {
		return false, err
	}

	return match, nil
}

func GitDirHash(repo *git.Repository, hash plumbing.Hash, prefix string, versionOverride *string) (*H1, error) {
	modulePath, moduleVersion := modulePathAndVersion(repo, hash, prefix)

	if versionOverride != nil {
		moduleVersion = *versionOverride
	}

	dependency := fmt.Sprintf("%s@%s", modulePath, moduleVersion)
	directoryHash, err := gitDirectoryHash(repo, hash, dependency, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to hash directory: %w", err)
	}

	modHash, err := gitGoModHash(repo, hash, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to hash go.mod: %w", err)
	}

	h1 := &H1{
		Path:          modulePath,
		Version:       moduleVersion,
		DirectoryHash: directoryHash,
		GoModHash:     modHash,
		Directory:     prefix,
	}

	return h1, nil
}

func gitDirectoryHash(repo *git.Repository, hash plumbing.Hash, dependency string, prefix string) (string, error) {
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return "", err
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", err
	}

	files, err := listTree(tree, dependency, prefix)
	if err != nil {
		return "", err
	}

	filesWithPrefix := make([]string, 0)
	for filename := range files {
		filesWithPrefix = append(filesWithPrefix, filename)
	}

	fileOpen := func(name string) (io.ReadCloser, error) {
		f, found := files[name]
		if !found {
			return nil, fmt.Errorf("errr %s: %w", name, err)
		}
		return f.Reader()
	}

	return dirhash.Hash1(filesWithPrefix, fileOpen)
}

func gitGoModHash(repo *git.Repository, hash plumbing.Hash, prefix string) (string, error) {
	commit, err := repo.CommitObject(hash)
	if err != nil {
		return "", err
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", err
	}

	osOpen := func(name string) (io.ReadCloser, error) {
		p := path.Join(prefix, name)
		f, err := tree.File(p)
		if err != nil {
			return nil, err
		}

		return f.Reader()
	}

	goModPath := "go.mod"
	modHash, err := dirhash.Hash1([]string{goModPath}, osOpen)
	if err != nil {
		return "", fmt.Errorf("failed to get hash of go.mod: %w", err)
	}

	return modHash, err
}

func listTree(tree *object.Tree, dependency string, prefix string) (map[string]*object.File, error) {
	fileMap := make(map[string]*object.File)

	modDirectories := make([]string, 0)
	goDotMod := "go.mod"
	goDotModPath := path.Join(prefix, goDotMod)

	// https://go.dev/ref/mod#vcs-license
	license := "LICENSE"
	localLicense := path.Join(prefix, license)
	hasLocalLicense := false

	err := tree.Files().ForEach(func(file *object.File) error {
		name := file.Name

		if name == localLicense {
			hasLocalLicense = true
		}

		fileMap[name] = file
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, goDotMod) && name != goDotModPath {
			modDirectories = append(modDirectories, strings.TrimSuffix(name, goDotMod))
		}

		return nil
	})

	filteredFileMap := make(map[string]*object.File)
	for name := range fileMap {
		include := true

		if !strings.HasPrefix(name, prefix) {
			if !hasLocalLicense && name == license {
				licensePath := dependency + "/" + name
				filteredFileMap[licensePath] = fileMap[name]
			} else {
				continue
			}
		}

		for _, directory := range modDirectories {
			if strings.HasPrefix(name, directory) {
				include = false
				break
			}
		}

		if include {
			p := dependency + "/" + strings.TrimPrefix(name, prefix)
			filteredFileMap[p] = fileMap[name]
		}
	}

	if err != nil {
		return nil, err
	}

	return filteredFileMap, nil
}

func modulePathAndVersion(repo *git.Repository, targetHash plumbing.Hash, prefix string) (string, string) {
	modulePath, err := getNameFromModule(repo, targetHash, prefix)
	if err != nil {
		log.Fatal(err)
	}

	goHash := targetHash.String()[0:12]

	// https://go.dev/ref/mod#pseudo-versions
	goTag, previousTagFound, tagOnCurrent, err := getNewestTag(repo, targetHash, prefix)
	if err != nil {
		log.Fatal(err)
	}

	commit, err := repo.CommitObject(targetHash)
	if err != nil {
		log.Fatal(err)
	}

	// TODO author or commiter date?
	goTime := commit.Committer.When.UTC().Format("20060102150405")

	var moduleVersion string
	if previousTagFound {
		if tagOnCurrent {
			moduleVersion = goTag
		} else {
			r := regexp.MustCompile(tagRegex)
			parts := r.FindStringSubmatch(goTag)

			patchVersion, err := strconv.Atoi(parts[3])
			if err != nil {
				log.Fatal(err)
			}

			goTag = "v" + parts[1] + "." + parts[2] + "." + strconv.Itoa(patchVersion+1)

			moduleVersion = fmt.Sprintf("%s-0.%s-%s", goTag, goTime, goHash)
		}
	} else {
		moduleVersion = fmt.Sprintf("v0.0.0-%s-%s", goTime, goHash)
	}

	return modulePath, moduleVersion
}

func getNewestTag(repo *git.Repository, hash plumbing.Hash, prefix string) (string, bool, bool, error) {
	// TODO figure out proper way to find latest tag

	repoState := gitkit.LoadRepoState(repo)
	tagMap := make(map[plumbing.Hash][]string)

	tags, err := repo.Tags()
	if err != nil {
		return "", false, false, err
	}

	err = tags.ForEach(func(tag *plumbing.Reference) error {
		entry := strings.TrimPrefix(tag.Name().String(), "refs/tags/")

		existing, found := tagMap[tag.Hash()]
		if !found {
			tagMap[tag.Hash()] = []string{entry}
		} else {
			existing = append(existing, entry)
			tagMap[tag.Hash()] = existing
		}
		return nil
	})
	if err != nil {
		return "", false, false, err
	}

	hashes := make([]plumbing.Hash, 0)
	hashes = append(hashes, hash)

	visited := make(map[plumbing.Hash]bool)
	visited[hash] = true

	onCurrent := true

	for {
		if len(hashes) == 0 {
			break
		}

		h := hashes[0]
		hashes = hashes[1:]

		commit, err := repo.CommitObject(h)
		if err != nil {
			return "", false, false, err
		}

		tags, found := repoState.TargetToTagMap[commit.Hash]
		if found {
			if len(tags) > 1 {
				return "", false, false, fmt.Errorf("multiple tags not supported, found for commit %s", commit.Hash.String())
			}

			tagCandidate := tags[0].Name

			if strings.HasPrefix(tagCandidate, prefix) {
				tagCandidate = strings.TrimPrefix(tagCandidate, prefix)
				match, err := regexp.MatchString(tagRegex, tagCandidate)
				if err != nil {
					return "", false, false, err
				}

				if !match {
					return "", false, false, fmt.Errorf("unsupported tag format for tag '%s'", tagCandidate)
				}

				return tagCandidate, true, onCurrent, nil
			}
		}

		tags2, found2 := tagMap[commit.Hash]
		if found2 {
			if len(tags2) > 1 {
				return "", false, false, fmt.Errorf("multiple tags not supported, found for commit %s", commit.Hash.String())
			}

			tagCandidate := tags2[0]

			if strings.HasPrefix(tagCandidate, prefix) {
				tagCandidate = strings.TrimPrefix(tagCandidate, prefix)
				match, err := regexp.MatchString(tagRegex, tagCandidate)
				if err != nil {
					return "", false, false, err
				}

				if !match {
					return "", false, false, fmt.Errorf("unsupported tag format for tag '%s'", tagCandidate)
				}

				return tagCandidate, true, onCurrent, nil
			}
		}

		onCurrent = false
		for i, parentHash := range commit.ParentHashes {
			if i < 1 {
				_, alreadyAdded := visited[parentHash]

				if !alreadyAdded {
					hashes = append(hashes, parentHash)
					visited[parentHash] = true
				}
			}
		}
	}

	return "", false, false, nil
}

func getNameFromModule(repo *git.Repository, hash plumbing.Hash, prefix string) (string, error) {
	commit, err := repo.CommitObject(hash)
	if err != nil {
		log.Fatal(err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", err
	}

	goModPath := path.Join(prefix, "go.mod")
	f, err := tree.File(goModPath)
	if err != nil {
		return "", err
	}

	r, err := f.Reader()
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	file, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return "", err
	}

	return file.Module.Mod.Path, nil
}
