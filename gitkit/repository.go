package gitkit

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type Repository interface {
	OrganizationName() string
	RepositoryName() string
	LocalRootPath() string
	SubPath() *string
}

type repo struct {
	org      string
	repoName string
	path     string
	subPath  *string
}

func NewRepo(org string, repoName string, path string, subPath *string) Repository {
	return &repo{
		org:      org,
		repoName: repoName,
		path:     path,
		subPath:  subPath,
	}
}

func (r *repo) OrganizationName() string {
	return r.org
}

func (r *repo) RepositoryName() string {
	return r.repoName
}

func (r *repo) LocalRootPath() string {
	return r.path
}

func (r *repo) SubPath() *string {
	return r.subPath
}

func InferReposFromPath(path string) ([]Repository, error) {
	type searchPath struct {
		repoPath          string
		subPath           string
		orgDirectoryPath  string
		orgsDirectoryPath string
	}

	sp := &searchPath{}
	repoDir, inGitRepoDirectory, err := GetRootPathOfLocalGitRepo(path)

	if err != nil {
		return nil, err
	}

	if !inGitRepoDirectory {
		inOrgDirectory, err := isOrgDirectory(path)
		if err != nil {
			return nil, err
		}

		if !inOrgDirectory {
			inOrgsDirectory, err := isOrgsDirectory(path)
			if err != nil {
				return nil, err
			}

			if !inOrgsDirectory {
				fmt.Println("no search directory specified")
				os.Exit(1)
			} else {
				sp.orgsDirectoryPath = path
				slog.Debug("searching git orgs", "path", path)
			}
		} else {
			sp.orgDirectoryPath = path
			slog.Debug("searching git org", "path", path)
		}
	} else {
		sp.repoPath = repoDir
		sp.subPath = path
		slog.Debug("searching git directory", "path", repoDir, "subPath", path)
	}

	// TODO Verify that all directories are repos
	repos := make([]Repository, 0)
	if sp.orgsDirectoryPath != "" {
		localOrgs, err := getDirectories(sp.orgsDirectoryPath)
		if err != nil {
			return nil, fmt.Errorf("unable to infer local repos from path %s (%w)", path, err)
		}
		for _, org := range localOrgs {
			orgPath := filepath.Join(sp.orgsDirectoryPath, org)
			localRepos, err := getDirectories(orgPath)
			if err != nil {
				return nil, fmt.Errorf("unable to infer local repos from path %s (%w)", path, err)
			}

			for _, repoName := range localRepos {
				path := filepath.Join(orgPath, repoName)
				repos = append(repos, NewRepo(org, repoName, path, nil))
			}
		}
	} else {
		if sp.orgDirectoryPath != "" {
			localRepos, err := getDirectories(sp.orgDirectoryPath)
			if err != nil {
				return nil, fmt.Errorf("unable to infer local repos from path %s (%w)", path, err)
			}

			for _, repoName := range localRepos {
				path := filepath.Join(sp.orgDirectoryPath, repoName)
				repos = append(repos, NewRepo("", repoName, path, nil))
			}
		} else {
			if sp.repoPath != "" {
				if sp.repoPath != sp.subPath {
					local := sp.subPath[len(sp.repoPath)+1:]
					repos = append(repos, NewRepo("", "", sp.repoPath, &local))
				} else {
					repos = append(repos, NewRepo("", "", sp.repoPath, nil))
				}
			}
		}
	}

	return repos, nil
}

func GetAllReposForAllOrgs(orgsDir string) ([]Repository, error) {
	repos := make([]Repository, 0)

	localOrgs, err := getDirectories(orgsDir)
	if err != nil {
		return nil, fmt.Errorf("unable to get all repos in org '%s': %w", orgsDir, err)
	}

	for _, org := range localOrgs {
		localRepos, err := getDirectories(filepath.Join(orgsDir, org))
		if err != nil {
			return nil, fmt.Errorf("unable to get all repos in org '%s': %w", orgsDir, err)
		}

		for _, repoName := range localRepos {
			path := filepath.Join(orgsDir, org, repoName)
			repos = append(repos, NewRepo(org, repoName, path, nil))
		}
	}

	return repos, nil
}

func getDirectories(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("unable to get local orgnaizations '%s': %w", path, err)
	}

	organizations := make([]string, 0)
	for _, e := range entries {
		if e.Type().IsDir() {
			organizations = append(organizations, e.Name())
		}
	}

	return organizations, nil
}

func isOrgDirectory(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			stat, err := os.Stat(filepath.Join(path, entry.Name(), ".git"))
			if err != nil {
				// skip
			} else {
				if stat.IsDir() {
					return true, nil
				}
			}

			return false, nil
		}
	}

	return false, nil
}

func isOrgsDirectory(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			inOrgDirectory, err := isOrgDirectory(filepath.Join(path, entry.Name()))
			if err != nil {
				return false, err
			}

			if inOrgDirectory {
				return true, nil
			} else {
				return false, nil
			}
		}
	}

	return false, nil
}
