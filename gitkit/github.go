package gitkit

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v61/github"
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

type GitHubResult struct {
	GitCommand string
	RepoName   string
	RepoURL    string
	RepoPath   string
	Error      error
}

type GitHubOptions struct {
	Depth int
	Bare  bool
}

func (gc *GitHubClient) CloneOrFetchRepo(url string, progressWriter *io.Writer, opts *GitHubOptions) (*GitHubResult, error) {

	if err := isGitHubURL(url); err != nil {
		return nil, err
	}

	result, err := gc.FetchRepo(url, progressWriter, opts)
	if err != nil {
		if strings.Contains(err.Error(), git.ErrRepositoryNotExists.Error()) {
			return gc.CloneRepo(url, progressWriter, opts)
		}
	}
	return result, err
}

func (gc *GitHubClient) CloneRepo(url string, progressWriter *io.Writer, opts *GitHubOptions) (*GitHubResult, error) {
	var result GitHubResult

	if err := isGitHubURL(url); err != nil {
		return &result, err
	}

	owner, repoName, err := ExtractOwnerAndRepoName(url)
	if err != nil {
		return nil, err
	}

	localRepoPath, err := getLocalRepoPath(owner, *repoName)
	if err != nil {
		return nil, err
	}

	if !opts.Bare {
		localRepoPath = strings.TrimSuffix(localRepoPath, ".git")
	}

	auth := configureAuth(gc.token)

	cloneOptions := &git.CloneOptions{
		Auth:     auth,
		URL:      url,
		Progress: os.Stdout,
		Depth:    opts.Depth,
	}

	result = GitHubResult{
		RepoName:   *repoName,
		RepoURL:    url,
		RepoPath:   localRepoPath,
		GitCommand: "Clone",
	}

	if progressWriter != nil {
		cloneOptions.Progress = *progressWriter
	}

	fmt.Fprintf(cloneOptions.Progress, "Cloning '%s/%s' into '%s'\n", owner, *repoName, localRepoPath)
	slog.Debug("Cloning", "repo", url, "to", localRepoPath)

	_, err = git.PlainClone(localRepoPath, opts.Bare, cloneOptions)
	if err != nil {
		result.Error = fmt.Errorf("error cloning repo: '%s/%s': %w", owner, *repoName, err)
	}

	return &result, result.Error
}

func (gc *GitHubClient) FetchRepo(url string, progressWriter *io.Writer, opts *GitHubOptions) (*GitHubResult, error) {
	var result GitHubResult

	if err := isGitHubURL(url); err != nil {
		return &result, err
	}

	owner, repoName, err := ExtractOwnerAndRepoName(url)
	if err != nil {
		return &result, err
	}

	localRepoPath, err := getLocalRepoPath(owner, *repoName)
	if err != nil {
		return &result, err
	}

	if !opts.Bare {
		url = strings.TrimSuffix(url, ".git")
		localRepoPath = strings.TrimSuffix(localRepoPath, ".git")
	}

	auth := configureAuth(gc.token)

	fetchOptions := &git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
		Progress:   os.Stdout,
		Prune:      true,
		Depth:      opts.Depth,
	}

	result = GitHubResult{
		RepoName:   *repoName,
		RepoURL:    url,
		RepoPath:   localRepoPath,
		GitCommand: "Fetch",
	}

	if progressWriter != nil {
		fetchOptions.Progress = *progressWriter
	}

	repo, err := git.PlainOpen(localRepoPath)
	if err != nil {
		return &result, fmt.Errorf("unable to fetch '%s/%s': %v", owner, *repoName, err)
	}

	fmt.Fprintf(fetchOptions.Progress, "Repository '%s/%s' exists. Fetching updates...\n", owner, *repoName)
	slog.Debug("Fetching updates for", "repo", localRepoPath)

	remote, err := repo.Remote("origin")
	if err != nil {
		result.Error = fmt.Errorf("error retrieving remote 'origin': %v", err)
		return &result, result.Error
	}

	err = remote.Fetch(fetchOptions)
	if err != nil {
		if err != git.NoErrAlreadyUpToDate && err.Error() != "remote repository is empty" {
			result.Error = fmt.Errorf("error fetching repo '%s:': %v", *repoName, err)
			return &result, result.Error
		}
	}

	return &result, nil
}

func (gc *GitHubClient) GetRepositories(url string) ([]string, error) {
	if err := isGitHubURL(url); err != nil {
		return nil, err
	}

	owner, repoName, err := ExtractOwnerAndRepoName(url)
	if err != nil {
		return nil, fmt.Errorf("invalid URL '%s': %w", url, err)
	}

	orgInfo, res, err := gc.client.Organizations.Get(context.Background(), owner)
	isOrg := res != nil && res.StatusCode == 200
	if err != nil && (res == nil || res.StatusCode != 404) {
		return nil, fmt.Errorf("error fetching organization info: %w", err)
	}

	userInfo, res, err := gc.client.Users.Get(context.Background(), owner)
	isUser := res != nil && res.StatusCode == 200
	if err != nil && (res == nil || res.StatusCode != 404) {
		return nil, fmt.Errorf("error fetching user info: %w", err)
	}

	if !(isOrg || isUser) {
		return nil, fmt.Errorf("no user or organization named '%s'", owner)
	}

	if isOrg {
		if strings.ToLower(owner) != strings.ToLower(*orgInfo.Login) {
			return nil, fmt.Errorf("actual '%s' and requested '%s' org differ in more than casing", *orgInfo.Login, owner)
		}
		owner = *orgInfo.Login
	} else {
		if strings.ToLower(owner) != strings.ToLower(*userInfo.Login) {
			return nil, fmt.Errorf("actual '%s' and requested '%s' user differ in more than casing", *orgInfo.Login, owner)
		}
		owner = *userInfo.Login
	}

	var repoURLs []string

	if repoName == nil { // all repos
		var repos []*github.Repository
		if isOrg {
			repos, err = gc.ListAllReposForOrg(owner)
		} else {
			repos, err = gc.ListAllReposForUser(owner)
		}
		if err != nil {
			return nil, fmt.Errorf("error listing repositories: %w", err)
		}

		for _, repo := range repos {
			repoURLs = append(repoURLs, fmt.Sprintf("https://github.com/%s/%s.git", owner, repo.GetName()))
		}
	} else { // single repo
		if *repoName == "" {
			return nil, fmt.Errorf("repository name must not be empty")
		}

		repoInfo, found, err := gc.GetGitHubRepo(owner, *repoName)
		if err != nil {
			return nil, fmt.Errorf("error getting repo '%s/%s': %w", owner, *repoName, err)
		}
		if !found {
			return nil, fmt.Errorf("repository '%s/%s' not found", owner, *repoName)
		}
		if !strings.EqualFold(*repoInfo.Name, *repoName) {
			return nil, fmt.Errorf("actual '%s' and requested '%s' repo differ in more than casing", *repoInfo.Name, *repoName)
		}

		repoURLs = append(repoURLs, fmt.Sprintf("https://github.com/%s/%s.git", owner, *repoName))
	}

	return repoURLs, nil
}

func ExtractOwnerAndRepoName(url string) (owner string, repoName *string, err error) {
	if err := isGitHubURL(url); err != nil {
		return "", nil, err
	}

	url = strings.TrimSuffix(url, ".git")
	userOrOrg := strings.Trim(strings.TrimPrefix(url, "https://github.com/"), "/")

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

func isGitHubURL(url string) error {
	if !strings.HasPrefix(url, "https://github.com/") {
		return fmt.Errorf("invalid URL '%s'; must be prefixed with 'https://github.com/'", url)
	}
	return nil
}

func getLocalRepoPath(owner, repoName string) (string, error) {
	localBasePath, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(localBasePath, owner, repoName), nil
}

func configureAuth(token *string) transport.AuthMethod {
	if token != nil {
		return &http.BasicAuth{
			Username: "token",
			Password: *token,
		}
	}
	return nil
}
