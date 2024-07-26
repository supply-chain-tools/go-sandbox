package gitkit

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v61/github"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type GitHubClient struct {
	token  *string
	client *github.Client
}

func NewAuthenticatedGitHubClient(token string) *GitHubClient {
	client := github.NewClient(nil).WithAuthToken(token)
	return &GitHubClient{
		client: client,
		token:  &token,
	}
}

func NewGitHubClient() *GitHubClient {
	client := github.NewClient(nil)
	return &GitHubClient{
		client: client,
		token:  nil,
	}
}

func (gc *GitHubClient) GetGitHubOrganization(org string) (info *github.Organization, found bool, err error) {
	result, res, err := gc.client.Organizations.Get(context.Background(), org)
	if err != nil {
		if res != nil && res.StatusCode == 404 {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("unable to get GitHub org '%s': %w", org, err)
	}

	return result, true, nil
}

func (gc *GitHubClient) GetGitHubUser(user string) (info *github.User, found bool, err error) {
	result, res, err := gc.client.Users.Get(context.Background(), user)
	if err != nil {
		if res != nil && res.StatusCode == 404 {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("unable to get GitHub user '%s': %w", user, err)
	}

	return result, true, nil
}

func (gc *GitHubClient) GetGitHubRepo(owner string, repoName string) (info *github.Repository, found bool, err error) {
	result, res, err := gc.client.Repositories.Get(context.Background(), owner, repoName)
	if err != nil {
		if res != nil && res.StatusCode == 404 {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("unable to get GitHub repo '%s/%s': %w", owner, repoName, err)
	}

	return result, true, nil
}

func (gc *GitHubClient) ListAllReposForOrg(org string) ([]*github.Repository, error) {
	page := 1
	options := &github.RepositoryListByOrgOptions{
		Type:        "all",
		ListOptions: github.ListOptions{PerPage: 100, Page: page},
	}

	repos := make([]*github.Repository, 0)
	for {
		resultRepos, res, err := gc.client.Repositories.ListByOrg(context.Background(), org, options)
		if err != nil {
			return nil, fmt.Errorf("unable to list GitHub repos for org '%s': %w", org, err)
		}

		if res.StatusCode != 200 {
			return nil, fmt.Errorf("unable to list GitHub repos for org '%s': status code not 200", org)
		}

		repos = append(repos, resultRepos...)

		if res.NextPage == 0 {
			break
		}

		page += 1
		options.Page = page
	}

	return repos, nil
}

func (gc *GitHubClient) ListAllReposForUser(user string) ([]*github.Repository, error) {
	page := 1
	options := &github.RepositoryListByUserOptions{
		Type:        "owner",
		ListOptions: github.ListOptions{PerPage: 100, Page: page},
	}

	repos := make([]*github.Repository, 0)
	for {
		resultRepos, res, err := gc.client.Repositories.ListByUser(context.Background(), user, options)
		if err != nil {
			return nil, fmt.Errorf("unable to list GitHub repos for user '%s': %w", user, err)
		}

		if res.StatusCode != 200 {
			return nil, fmt.Errorf("unable to list GitHub repos for user '%s': status code not 200", user)
		}

		repos = append(repos, resultRepos...)

		if res.NextPage == 0 {
			break
		}

		page += 1
		options.Page = page
	}

	return repos, nil
}

func (gc *GitHubClient) CloneOrFetchGitHubToPath(org string, repoName string, path string, depth int) error {
	fmt.Printf("Cloning '%s/%s'\n", org, repoName)

	var auth transport.AuthMethod = nil
	if gc.token != nil {
		auth = &http.BasicAuth{
			Username: "token",
			Password: *gc.token,
		}
	}

	// Use max depth if zero. See https://git-scm.com/docs/shallow
	if depth == 0 {
		depth = 2147483647
	}

	cloneOptions := &git.CloneOptions{
		Auth:     auth,
		URL:      "https://github.com/" + org + "/" + repoName,
		Depth:    depth,
		Progress: os.Stdout,
	}

	_, err := git.PlainClone(path, false, cloneOptions)
	if err != nil {
		errorString := err.Error()
		if errorString == "repository already exists" {
			fmt.Printf("Updating\n")
			repo, err := git.PlainOpen(path)
			if err != nil {
				return fmt.Errorf("unable to open git repository '%s': %w", path, err)
			}

			err = repo.Fetch(&git.FetchOptions{
				RemoteName: "origin",
				Auth:       auth,
				Progress:   os.Stdout,
				Prune:      true,
				Depth:      depth,
			})
			if err != nil && err.Error() != "already up-to-date" && err.Error() != "remote repository is empty" {
				return fmt.Errorf("unable to fetch repo '%s': %w", path, err)
			}
		} else if errorString != "remote repository is empty" {
			// FIXME better tracking of failed repos
			// https://stackoverflow.com/a/57733557
			log.Printf("warning skipping repo '%s/%s' due to error '%s'", org, repoName, errorString)
		}
	}

	return nil
}

func (gc *GitHubClient) CloneOrFetchAllRepos(owner string, repoName *string, localBasePath string, depth int) error {
	orgInfo, isOrg, err := gc.GetGitHubOrganization(owner)
	if err != nil {
		return err
	}

	userInfo, isUser, err := gc.GetGitHubUser(owner)
	if err != nil {
		return err
	}

	if !(isOrg || isUser) {
		return fmt.Errorf("no user or org named %s", owner)
	}

	if isOrg {
		if strings.ToLower(owner) != strings.ToLower(*orgInfo.Login) {
			return fmt.Errorf("actual '%s' and requested '%s' org differ in more than casing", *orgInfo.Login, owner)
		}
		owner = *orgInfo.Login
	} else {
		if strings.ToLower(owner) != strings.ToLower(*userInfo.Login) {
			return fmt.Errorf("actual '%s' and requested '%s' user differ in more than casing", *orgInfo.Login, owner)
		}
		owner = *userInfo.Login
	}

	ownerPath, err := createDirectoryIfNotExists(localBasePath, owner)
	if err != nil {
		return err
	}

	if repoName == nil {
		var allRepos []*github.Repository
		var ownerType string
		if isOrg {
			ownerType = "organization"
			allRepos, err = gc.ListAllReposForOrg(owner)
			if err != nil {
				return err
			}
		} else {
			ownerType = "user"
			allRepos, err = gc.ListAllReposForUser(owner)
			if err != nil {
				return err
			}
		}

		slog.Debug("cloning/fetching",
			"owner", owner,
			"ownerType", ownerType,
			"numberOfRepos", len(allRepos))

		for _, r := range allRepos {
			repoPath := filepath.Join(ownerPath, r.GetName())
			err = gc.CloneOrFetchGitHubToPath(owner, r.GetName(), repoPath, depth)
			if err != nil {
				return err
			}
		}
	} else {
		if *repoName == "" {
			return fmt.Errorf("repo name must not be empty string")
		}

		repoInfo, found, err := gc.GetGitHubRepo(owner, *repoName)
		if err != nil {
			return err
		}

		if !found {
			return fmt.Errorf("repository '%s/%s' not found\n", owner, *repoName)
		}

		if strings.ToLower(*repoInfo.Name) != strings.ToLower(*repoName) {
			return fmt.Errorf("actual '%s' and requested '%s' repo differ in more than casing", *repoInfo.Name, *repoName)
		}
		*repoName = *repoInfo.Name

		var ownerType string
		if isOrg {
			ownerType = "organization"
		} else {
			ownerType = "user"
		}

		slog.Debug("cloning/fetching",
			"owner", owner,
			"ownerType", ownerType,
			"repoName", repoName)

		repoPath := filepath.Join(ownerPath, *repoName)
		err = gc.CloneOrFetchGitHubToPath(owner, *repoName, repoPath, depth)
		if err != nil {
			return err
		}
	}

	return nil
}

func createDirectoryIfNotExists(basePath string, relativePath string) (string, error) {
	path := filepath.Join(basePath, relativePath)
	stat, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			return "", err
		}
		return path, nil
	} else if err != nil {
		return "", err
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("path '%s' exists but is not a directory", path)
	}

	return path, nil
}

func ExtractOwnerAndRepoName(input string) (owner string, repoName *string, err error) {
	var userOrOrg string
	const httpsGithubPrefix = "https://github.com/"
	const githubPrefix = "github.com/"
	if strings.HasPrefix(input, httpsGithubPrefix) {
		userOrOrg = strings.TrimPrefix(input, httpsGithubPrefix)
	} else if strings.HasPrefix(input, githubPrefix) {
		userOrOrg = strings.TrimPrefix(input, githubPrefix)
	} else {
		return "", nil, fmt.Errorf("invalid target '%s'; must be prefixed with 'github.com/'", userOrOrg)
	}

	userOrOrg = strings.Trim(userOrOrg, "/")
	if userOrOrg == "" {
		return "", nil, fmt.Errorf("'owner' or 'owner/repo' must be specified")
	}

	parts := strings.Split(userOrOrg, "/")

	if len(parts) < 1 || len(parts) > 2 {
		return "", nil, fmt.Errorf("expected an 'owner' or 'owner/repo', got %d parts in '%s' instead", len(parts), userOrOrg)
	}

	owner = parts[0]
	repoName = nil

	if len(parts) > 1 {
		repoName = &parts[1]
	}

	return owner, repoName, nil
}
