package scanfail

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/supply-chain-tools/go-sandbox/hashset"
	"go.uber.org/ratelimit"
)

type RepoStatus string

const (
	PROPERLY_CONFIGURED RepoStatus = "PROPERLY_CONFIGURED"
	MISCONFIGURED       RepoStatus = "MISCONFIGURED"
	ADS_DISABLED        RepoStatus = "ADS_DISABLED"
	CODESCAN_DISABLED   RepoStatus = "CODESCAN_DISABLED"
)

type RepoResult struct {
	RepoName              string
	Status                RepoStatus
	IsArchived            bool
	IsUsingYaml           bool
	UnConfiguredLanguages []string
	ConfiguredLanguages   []string
}

func LoadRepoNames(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	result := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}

	return result, nil
}
func CheckRepoConfigMisConfigurations(c *Client, allRepos []string, resultFilename string) ([]string, error) {
	repoStatuses := make(map[string]*RepoResult)
	reposFailed := make([]string, 0)

	fmt.Println("Fetching repo names...")

	rl := ratelimit.New(30) // per second
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	for _, repoName := range allRepos {
		wg.Add(1)
		go func(repoName string) {
			defer wg.Done()
			rl.Take() //Wait for ratelimit to allow

			result, err := CheckRepo(c, repoName)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				fmt.Printf("Error occured while checking repo '%s': %s\n", repoName, err.Error())
				reposFailed = append(reposFailed, repoName)
				return
			}
			repoStatuses[repoName] = result

		}(repoName)
	}

	wg.Wait()

	misConfiguredRepositories := make([]string, 0)
	properlyConfiguredRepositories := make([]string, 0)
	adsDisabled := make([]string, 0)
	codeScanDisabled := make([]string, 0)
	reposUsingYaml := make([]string, 0)
	for repoName, result := range repoStatuses {
		switch result.Status {
		case ADS_DISABLED:
			adsDisabled = append(adsDisabled, repoName)
			break
		case CODESCAN_DISABLED:
			codeScanDisabled = append(codeScanDisabled, repoName)
			break
		case MISCONFIGURED:
			misConfiguredRepositories = append(misConfiguredRepositories, repoName)
			break
		case PROPERLY_CONFIGURED:
			properlyConfiguredRepositories = append(properlyConfiguredRepositories, repoName)
			break
		default:
			fmt.Printf("Unhandled status '%s' for repo '%s',struct was %#v\n", result.Status, repoName, result)
			break
		}
		if result.IsUsingYaml {
			reposUsingYaml = append(reposUsingYaml, repoName)
		}
	}

	filteredRepoStatuses := make(map[string]*RepoResult)
	for repoName, result := range repoStatuses {
		if result.Status != PROPERLY_CONFIGURED {
			filteredRepoStatuses[repoName] = result
		}
	}

	resultFilename = strings.TrimSuffix(filepath.Base(resultFilename), filepath.Ext(resultFilename))

	err := dumpObjectsToFile(filteredRepoStatuses, filepath.Join("results", resultFilename+"-misconfigured.json"))
	if err != nil {
		return nil, err
	}
	err = dumpObjectsToFile(repoStatuses, filepath.Join("results", resultFilename+".json"))
	if err != nil {
		return nil, err
	}
	err = dumpListToFile(reposFailed, "./results/failedrepositories.txt")
	if err != nil {
		return nil, err
	}

	showRepoStats(misConfiguredRepositories, properlyConfiguredRepositories, adsDisabled, codeScanDisabled, reposUsingYaml, reposFailed)
	return misConfiguredRepositories, nil
}
func CheckRepo(c *Client, repoName string) (*RepoResult, error) {
	check, isUsingYaml, unconfiguredLanguage, configuredLanguages, err := checkRepoConfig(c, repoName)

	repoArchiveStatus, err2 := GetRepoArchiveStatus(c, repoName)
	if err2 != nil {
		return nil, fmt.Errorf("error getting repo: %w", err)
	}

	if unconfiguredLanguage == nil {
		unconfiguredLanguage = make([]string, 0)
	}
	if configuredLanguages == nil {
		configuredLanguages = make([]string, 0)
	}
	result := &RepoResult{IsUsingYaml: isUsingYaml, Status: PROPERLY_CONFIGURED, RepoName: repoName, IsArchived: repoArchiveStatus, UnConfiguredLanguages: unconfiguredLanguage, ConfiguredLanguages: configuredLanguages}

	if err != nil {
		var targetErr *disabledError
		if errors.As(err, &targetErr) {
			result.Status = targetErr.repoStatus
		} else {
			return nil, fmt.Errorf("error checking repository '%s': %w", repoName, err)
		}
		return result, nil
	}

	if check == false {
		result.Status = MISCONFIGURED
		fmt.Println("Missconfigured repo:", repoName)
	}

	return result, nil
}
func checkRepoConfig(c *Client, repoName string) (bool, bool, []string, []string, error) {
	// this wil check if the given GitHub repo code scanning setting is configured properly.
	// it wil return true if it is configured correctly
	// if it is not configured properly it wil return false
	codeScanningConfiguration, err := c.GetRepoCodeScanningConfiguration(repoName)
	if err != nil {
		return false, false, nil, nil, fmt.Errorf("error getting code scanning configuration: %w", err)
	}

	defaultCodeScaningLanguages := hashset.New(normalizeLanguages(codeScanningConfiguration.Languages)...)

	databases, err := c.GetCodeQLDatabases(repoName)
	if err != nil {
		return false, false, nil, nil, fmt.Errorf("error getting code QL database languages: %w", err)
	}

	repoLanguages, err := c.GetRepoLanguages(repoName)
	if err != nil {
		return false, false, nil, nil, fmt.Errorf("error getting code QL database languages: %w", err)
	}

	supportedRepoLanguages := extractSupportedLanguages(repoLanguages)
	// Union these because Actions is not available in api that returns what languages is present in a repo
	supportedRepoLanguages = union(supportedRepoLanguages, defaultCodeScaningLanguages)
	codeQLDatabaseLanguages := hashset.New[string]()

	for i := 0; i < len(databases); i++ {
		d := databases[i].GetLanguage()
		codeQLDatabaseLanguages.Add(d)
	}
	isUsingYaml := codeScanningConfiguration.GetState() == "not-configured"

	if isUsingYaml {
		unconfiguredLanguages := supportedRepoLanguages.Difference(codeQLDatabaseLanguages)
		configuredCorrectly := unconfiguredLanguages.Size() == 0
		return configuredCorrectly, isUsingYaml, unconfiguredLanguages.Values(), codeQLDatabaseLanguages.Values(), nil
	} else {
		unconfiguredLanguages := supportedRepoLanguages.Difference(defaultCodeScaningLanguages)
		configuredCorrectly := unconfiguredLanguages.Size() == 0
		return configuredCorrectly, isUsingYaml, unconfiguredLanguages.Values(), defaultCodeScaningLanguages.Values(), nil
	}
}
func normalizeCodeCLLanguage(language string) string {
	switch language {
	case "java-kotlin":
		return "java"
	case "javascript-typescript":
		return "javascript"
	case "typescript":
		return "javascript"
	case "c-cpp":
		return "cpp"
	default:
		return language
	}
}
func normalizeRepoLanguage(language string) string {
	inputLanguage := strings.ToLower(language)
	switch inputLanguage {
	case "vue":
		return "javascript"
	case "html":
		return "javascript"
	case "typescript":
		return "javascript"
	default:
		return inputLanguage
	}
}
func normalizeLanguages(languages []string) []string {
	normalized := make([]string, 0)
	for _, lang := range languages {
		normalized = append(normalized, normalizeCodeCLLanguage(lang))
	}
	return normalized
}
func intersection(a hashset.Set[string], b hashset.Set[string]) hashset.Set[string] {
	result := hashset.New[string]()

	for _, item := range a.Values() {
		if b.Contains(item) {
			result.Add(item)
		}
	}

	return result
}
func union(a, b hashset.Set[string]) hashset.Set[string] {
	result := hashset.New[string]()
	for _, item := range a.Values() {
		result.Add(item)
	}
	for _, item := range b.Values() {
		result.Add(item)
	}
	return result
}
func extractSupportedLanguages(languagesInRepo []string) hashset.Set[string] {

	normalizedLanguagesInRepo := hashset.New[string]()
	for _, language := range languagesInRepo {
		normalized := normalizeRepoLanguage(language)
		normalizedLanguagesInRepo.Add(normalized)
	}

	languages := []string{"actions", "c-cpp", "csharp", "go", "java-kotlin", "javascript-typescript", "javascript", "python", "rust", "ruby", "typescript", "swift"}
	supportedLanguages := hashset.New(languages...)

	return intersection(supportedLanguages, normalizedLanguagesInRepo)

}

func dumpListToFile(list []string, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer file.Close()
	w := bufio.NewWriter(file)
	for _, repo := range list {
		w.WriteString(repo + "\n")
	}
	w.Flush()
	return nil
}
func dumpObjectsToFile(objects any, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	jsonResult, err := json.Marshal(objects)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(file)
	w.Write(jsonResult)
	w.Flush()
	return nil
}
func showRepoStats(misConfiguredRepositories []string, properlyConfiguredRepositories []string, adsDisabled []string, codeScanDisabled []string, reposUsingYaml []string, reposFailed []string) {
	fmt.Println("================================")
	fmt.Println("misconfiguredRepositories:", misConfiguredRepositories)
	fmt.Println("adsDisabledRepos:", adsDisabled)
	fmt.Println("codeScanDisabledRepos:", codeScanDisabled)
	fmt.Println("properlyConfiguredRepositories:", properlyConfiguredRepositories)
	fmt.Println("failed repository:", reposFailed)
	fmt.Println("================================")
	fmt.Println("Stats:")
	fmt.Println("Misconfigured repos:", len(misConfiguredRepositories))
	fmt.Println("properly configured repos:", len(properlyConfiguredRepositories))
	fmt.Println("Repos with Advanced Security disabled:", len(adsDisabled))
	fmt.Println("Repos with Code Scanning Disabled:", len(codeScanDisabled))
	fmt.Println("Repos configured with codeql.yaml:", len(reposUsingYaml))
	fmt.Println("Repos that failed to scan:", len(reposFailed))
}
