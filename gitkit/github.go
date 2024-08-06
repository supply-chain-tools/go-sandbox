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

type CloneResult struct {
	RepoName  string
	RepoURL   string
	LocalPath string
	Status    string
	Error     error
}

type CloneOptions struct {
	Depth int
	Bare  bool
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

func (gc *GitHubClient) GetRepositories(url string) ([]string, error) {
	url = strings.TrimSuffix(url, ".git")

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
			return nil, fmt.Errorf("error fetching repository '%s/%s': %w", owner, *repoName, err)
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

func (gc *GitHubClient) CloneOrFetchRepo(repoURL string, localBasePath string, opts *CloneOptions, progressWriter *io.Writer) (*CloneResult, error) {
	var result CloneResult

	if progressWriter == nil {
		progressWriter = new(io.Writer)
		*progressWriter = os.Stdout
	}

	if !strings.HasPrefix(repoURL, "https://") && !strings.HasPrefix(repoURL, "http://") {
		repoURL = "https://" + repoURL
	}

	owner, repoName, err := ExtractOwnerAndRepoName(repoURL)
	if err != nil {
		return &result, fmt.Errorf("invalid URL '%s': %w", repoURL, err)
	}

	repoPath := filepath.Join(localBasePath, owner, *repoName)
	result = CloneResult{
		RepoName:  *repoName,
		RepoURL:   repoURL,
		LocalPath: repoPath,
	}

	var auth transport.AuthMethod = nil
	if gc.token != nil {
		auth = &http.BasicAuth{
			Username: "token",
			Password: *gc.token,
		}
	}

	cloneOptions := &git.CloneOptions{
		Auth:     auth,
		URL:      repoURL,
		Progress: *progressWriter,
		Depth:    opts.Depth,
	}

	fetchOptions := &git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
		Progress:   *progressWriter,
		Prune:      true,
		Depth:      opts.Depth,
	}

	repo, err := git.PlainOpen(repoPath)
	if err == nil {
		fmt.Fprintf(*progressWriter, "Repository '%s/%s' exists. Fetching updates...\n", owner, *repoName)
		slog.Debug("Fetching repository", "owner", owner, "repoName", *repoName, "path", repoPath)

		err = repo.Fetch(fetchOptions)
		if err != nil {
			if err != git.NoErrAlreadyUpToDate && err.Error() != "remote repository is empty" {
				result.Error = fmt.Errorf("unable to fetch repo '%s': %v", *repoName, err)
				slog.Debug(result.Error.Error())
				return &result, result.Error
			}
		}
		result.Status = "Fetched"
		slog.Debug("Successfully fetched", "owner", owner, "repoName", *repoName, "path", repoPath)
	} else if err == git.ErrRepositoryNotExists { // repo does not exist, clone
		fmt.Fprintf(*progressWriter, "Cloning '%s/%s' into '%s'\n", owner, *repoName, repoPath)
		slog.Debug("Cloning repository", "url", repoURL, "repoName", *repoName, "path", repoPath)

		if opts.Bare {
			_, err = git.PlainClone(repoPath+".git", true, cloneOptions)
		} else {
			_, err = git.PlainClone(repoPath, false, cloneOptions)
		}

		if err == nil {
			result.Status = "Cloned"
			slog.Debug("Successfully cloned", "url", repoURL, "repoName", *repoName, "path", repoPath)
		} else {
			result.Error = fmt.Errorf("error cloning repo '%s/%s': %v", owner, *repoName, err)
			slog.Debug(result.Error.Error())
			return &result, result.Error
		}
	} else {
		result.Error = fmt.Errorf("error accessing repository '%s/%s': %v", owner, *repoName, err)
		slog.Debug(result.Error.Error())
		return &result, result.Error
	}

	return &result, nil
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
		return "", nil, fmt.Errorf("invalid target '%s'; must start with '%s' or '%s", userOrOrg, httpsGithubPrefix, githubPrefix)
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
