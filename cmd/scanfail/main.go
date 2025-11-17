package main

import (
	"flag"
	"fmt"
	"github.com/supply-chain-tools/go-sandbox/scanfail"
	"os"
)

const usage = `Usage:
scanfail [options]
options:
--private-key File with PEM encoded private key for your GitHub App
--app-id      File with GitHub App ID
--org-name    Name of organization you want to scan
--repos       File containing list of repositories
--help        Show this help`

func main() {
	appIdFileName, privateKeyFileName, orgName, reposFileName, err := processOpts()
	if err != nil {
		print("Failed to process command line optionsq: ", err.Error(), "\n")
		os.Exit(1)
	}

	if _, err := os.Stat("./results"); os.IsNotExist(err) {
		err := os.MkdirAll("./results", 0755)
		if err != nil {
			print("Error creating results directory: %v\n", err.Error(), "\n")
			os.Exit(1)
		}
	}

	client, err := scanfail.NewClient(appIdFileName, privateKeyFileName, orgName)
	if err != nil {
		print("Error creating client: ", err.Error(), "\n")
		os.Exit(1)
	}
	RepoNames, err := scanfail.LoadRepoNames(reposFileName)
	if err != nil {
		print("Error loading repo names: ", err.Error(), "\n")
		os.Exit(1)
	}

	_, err = scanfail.CheckRepoConfigMisConfigurations(client, RepoNames, reposFileName)
	if err != nil {
		print("Error checking repo config: ", err.Error(), "\n")
		os.Exit(1)
	}

}

func processOpts() (appIdFileName string, privateKeyFileName string, orgName string, reposFileName string, err error) {

	flag.Usage = func() {
		fmt.Println(usage)
	}

	flags := flag.NewFlagSet("all", flag.ExitOnError)

	flags.StringVar(&privateKeyFileName, "private-key", "", "File with PEM encoded private key for your GitHub App")
	flags.StringVar(&appIdFileName, "app-id", "", "File with GitHub App ID")
	flags.StringVar(&orgName, "org-name", "", "Name of organization you want to scan")
	flags.StringVar(&reposFileName, "repos", "", "File containing list of repositories")
	help := flags.Bool("help", false, "Show help message")
	err = flags.Parse(os.Args[1:])
	if err != nil {
		err = fmt.Errorf("failed to parse flags: %s", err.Error())
		return
	}

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if privateKeyFileName == "" {
		err = fmt.Errorf("--private-key is required")
		return
	}

	if appIdFileName == "" {
		err = fmt.Errorf("--app-id is required")
		return
	}

	if orgName == "" {
		err = fmt.Errorf("--org-name is required")
		return
	}

	if reposFileName == "" {
		err = fmt.Errorf("--repos is required")
		return
	}

	return
}
