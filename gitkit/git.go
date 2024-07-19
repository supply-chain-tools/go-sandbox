package gitkit

import (
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func OpenRepoInLocalPath(path string) (*git.Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open git repository '%s': %w", path, err)
	}

	return repo, nil
}

func GetRootPathOfLocalGitRepo(startPath string) (rootPath string, found bool, err error) {
	// https://stackoverflow.com/a/65499840

	path := startPath

	for {
		if path == "/" {
			return "", false, nil
		}

		stat, err := os.Stat(filepath.Join(path, ".git"))

		if err != nil && errors.Is(err, os.ErrNotExist) {
			// skip
		} else if err != nil {
			return "", false, err
		} else {
			if stat.IsDir() {
				return path, true, nil
			}
		}

		slog.Debug("looking for .git in", "path", path)
		path = filepath.Dir(path)
	}
}

func ListRemoteReferences(repo *git.Repository) ([]*plumbing.Reference, error) {
	remotes := make([]*plumbing.Reference, 0)
	refs, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("unable to list remote refernces: %w", err)
	}

	err = refs.ForEach(func(branch *plumbing.Reference) error {
		if strings.HasPrefix(branch.Name().String(), "refs/remotes/origin") {
			remotes = append(remotes, branch)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list remote refernces: %w", err)
	}

	return remotes, nil
}
