package scanfail

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v72/github"
)

type disabledError struct {
	repoStatus RepoStatus
}

func (e *disabledError) Error() string {
	return fmt.Sprintf("%s", e.repoStatus)
}

func ReadFileAsString(name string) (string, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("error occured while reading file '%s': %s", name, err.Error())
	}

	return strings.TrimSuffix(string(data), "\n"), nil
}

func GetClient(appIdFileName string, privateKeyFileName string) (*github.Client, error) {
	appID, err := AppID(appIdFileName)
	if err != nil {
		return nil, err
	}

	privateKey, err := PrivateKey(privateKeyFileName)
	if err != nil {
		return nil, err
	}
	itr, err := ghinstallation.NewAppsTransport(http.DefaultTransport, appID, privateKey)
	if err != nil {
		return nil, err
	}

	return github.NewClient(&http.Client{Transport: itr}), nil
}

func GetClientWithInstallationId(appIdFileName string, privateKeyFileName string, installationId int64) (*github.Client, error) {
	appID, err := AppID(appIdFileName)
	if err != nil {
		return nil, err
	}

	privateKey, err := PrivateKey(privateKeyFileName)
	if err != nil {
		return nil, err
	}
	itr, err := ghinstallation.New(http.DefaultTransport, appID, installationId, privateKey)
	if err != nil {
		return nil, err
	}

	return github.NewClient(&http.Client{Transport: itr}), nil
}

func AppID(appIdFileName string) (int64, error) {
	s, err := ReadFileAsString(appIdFileName)
	if err != nil {
		return 0, err
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

func PrivateKey(privateKeyFileName string) ([]byte, error) {
	result, err := ReadFileAsString(privateKeyFileName)
	if err != nil {
		return nil, err
	}
	return []byte(result), nil
}

func GetInstallationId(appIdFileName string, privateKeyFileName string) (int64, error) {
	client, err := GetClient(appIdFileName, privateKeyFileName)
	if err != nil {
		return 0, err
	}

	r, res, err := client.Apps.ListInstallations(context.Background(), nil)
	if err != nil {
		return 0, err
	}
	if res.StatusCode != 200 {
		return 0, fmt.Errorf("unable to get GitHub repo installation: status code was %d but expected 200", res.StatusCode)
	}

	if len(r) != 1 {
		return 0, fmt.Errorf("unable to get GitHub app installation")
	}

	return *r[0].ID, nil
}

type Client struct {
	ghClient     *github.Client
	organization string
}

func NewClient(appIdFileName string, privateKeyFileName string, organization string) (*Client, error) {
	installationId, err := GetInstallationId(appIdFileName, privateKeyFileName)
	if err != nil {
		return nil, err
	}

	client, err := GetClientWithInstallationId(appIdFileName, privateKeyFileName, installationId)
	if err != nil {
		return nil, err
	}

	return &Client{
		ghClient:     client,
		organization: organization,
	}, nil
}

func (c *Client) GetRepoLanguages(repoName string) ([]string, error) {
	r, res, err := c.ghClient.Repositories.ListLanguages(context.Background(), c.organization, repoName)
	err = checkErrorCondition(err, res, repoName, c.organization, "repo languages")
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	for key, _ := range r {
		result = append(result, key)
	}
	return result, nil
}

func (c *Client) GetRepo(repoName string) (*github.Repository, error) {
	r, res, err := c.ghClient.Repositories.Get(context.Background(), c.organization, repoName)
	err = checkErrorCondition(err, res, repoName, c.organization, "repo")
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) GetCodeQLDatabases(repoName string) ([]*github.CodeQLDatabase, error) {
	r, res, err := c.ghClient.CodeScanning.ListCodeQLDatabases(context.Background(), c.organization, repoName)
	err = checkErrorCondition(err, res, repoName, c.organization, "CodeQLDatabases")
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) GetRepoCodeScanningConfiguration(repoName string) (*github.DefaultSetupConfiguration, error) {
	r, res, err := c.ghClient.CodeScanning.GetDefaultSetupConfiguration(context.Background(), c.organization, repoName)

	if err != nil {
		if strings.Contains(err.Error(), "Code scanning is not enabled for this repository. Please enable code scanning in the repository settings.") {
			return nil, &disabledError{repoStatus: CODESCAN_DISABLED}
		} else if strings.Contains(err.Error(), "Advanced Security must be enabled for this repository to use code scanning.") {
			return nil, &disabledError{repoStatus: ADS_DISABLED}
		}
	}

	err = checkErrorCondition(err, res, repoName, c.organization, "code scanning configuration")
	if err != nil {
		return nil, err
	}

	return r, nil
}
func GetRepoArchiveStatus(c *Client, repoName string) (bool, error) {
	repo, err := c.GetRepo(repoName)
	if err != nil {
		return false, fmt.Errorf("error getting repo: %w", err)
	}

	return repo.GetArchived(), nil
}

func checkErrorCondition(err error, res *github.Response, repoName string, organizationName string, functionName string) error {
	if err != nil {
		if res != nil {
			return fmt.Errorf("unable to list %s in %s repo for org '%s' status code: %d", functionName, repoName, organizationName, res.StatusCode)
		} else {
			return fmt.Errorf("unable to list %s in %s repo for org '%s'", functionName, repoName, organizationName)
		}
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("unable to list %s in %s repo for org '%s': status code not 200", functionName, repoName, organizationName)
	}
	return nil
}
