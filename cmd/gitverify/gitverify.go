package main

import (
	"crypto/sha1"
	"crypto/sha512"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/supply-chain-tools/go-sandbox/githash"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/gitverify"
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"log/slog"
	"os"
	"runtime/debug"
	"sort"
	"strings"
)

const usage = `Usage:
    gitverify [COMMAND] [OPTIONS]

COMMANDS
        verify
                Verify the state of a Git repository. This is also the default if no command is specified.
        after-candidates
                Generate a list of all commits that is not pointed to by other commits. The list can be
                used as the 'after' config.
        exempt-tags
                Generate a list of all the tags in the repository to be used for the 'exemptTags' config.

VERIFY OPTIONS
        --config-file
                Config file to use.
        --repository-uri
                URI to the repository in the config file.
        --commit
                Verify the commit.
        --tag
                Verify the tag and that it points to --commit.
        --branch
                Verify branch and ensure that --commit is on the branch.
        --verify-on-tip
                Verify that --commit is at the tip of --branch.
        --verify-on-head
                verify that HEAD points to the --commit. On by default.

AFTER-CANDIDATES OPTIONS
        --config-file
                Config file to use.
        --repository-uri
                URI to the repository in the config file.
        --sha512
                Output SHA-512 hashes in addition to SHA-1.

EXEMPT-TAGS OPTIONS
        --sha512
                Output SHA-512 hashes in addition to SHA-1.

GLOBAL OPTIONS
        --help, -h
                Show help
        --debug
                Output debug information.

Verify current repo
    $ gitverify

Verify current repo, specify config file and uri
    $ gitverify --config-file gitverify.json --repository-uri git+https://github.com/supply-chain-tools/go-sandbox.git

Verify repo and make sure a given commit and tag is present, that the tag points to the commit, that the commit
is on branch 'main' and that the commit is a descendant of 'after'
    $ gitverify --commit 1f46f2053221c040ce5bcba0239bc09214a37658 --tag v0.0.1 --branch main`

func main() {
	flag.Usage = func() {
		fmt.Println(usage)
	}

	command := "verify"
	if len(os.Args) > 1 {
		c := os.Args[1]
		if !strings.HasPrefix(c, "-") {
			command = os.Args[1]
		}
	}

	switch command {
	case "verify":
		opts, err := parseVerifyOptions(os.Args)
		if err != nil {
			print("failed to parse input: ", err.Error(), "\n")
			os.Exit(1)
		}

		err = verify(opts)
		if err != nil {
			print("verification failed: ", err.Error(), "\n")
			os.Exit(1)
		}
	case "after-candidates":
		opts, err := parseGenerateOptions(os.Args[2:])
		if err != nil {
			print("failed to parse input: ", err.Error(), "\n")
			os.Exit(1)
		}

		err = afterCandidates(opts)
		if err != nil {
			print("after-candidates failed: ", err.Error(), "\n")
		}
	case "exempt-tags":
		opts, err := parseGenerateOptions(os.Args[2:])
		if err != nil {
			print("failed to parse input: ", err.Error(), "\n")
			os.Exit(1)
		}

		result, err := exemptTags(opts)
		if err != nil {
			print("failed to get exempt tags: ", err.Error(), "\n")
			os.Exit(1)
		}
		fmt.Println(result)
	default:
		fmt.Printf("unknown command: %s\n", command)
		os.Exit(1)
	}
}

type VerifyOptions struct {
	repoDir         string
	validateOptions *gitverify.ValidateOptions
	configFilePath  string
	repoUri         string
	localState      bool
}

func parseVerifyOptions(osArgs []string) (*VerifyOptions, error) {
	flags := flag.NewFlagSet("all", flag.ExitOnError)
	var help, h, debugMode, verifyOnHEAD, verifyOnTip, localState, version bool
	var configFilePath, repoUri, commit, tag, branch string
	flags.BoolVar(&help, "help", false, "")
	flags.BoolVar(&h, "h", false, "")
	flags.BoolVar(&version, "version", false, "")
	flags.BoolVar(&debugMode, "debug", false, "")

	flags.StringVar(&configFilePath, "config-file", "", "")
	flags.StringVar(&repoUri, "repository-uri", "", "")
	flags.BoolVar(&localState, "local-state", true, "")

	flags.StringVar(&commit, "commit", "", "")
	flags.StringVar(&tag, "tag", "", "")
	flags.StringVar(&branch, "branch", "", "")
	flags.BoolVar(&verifyOnHEAD, "verify-on-head", true, "")
	flags.BoolVar(&verifyOnTip, "verify-on-tip", false, "")

	args := osArgs[1:]
	if len(osArgs) > 2 && !strings.HasPrefix(osArgs[1], "-") {
		args = osArgs[2:]
	}

	err := flags.Parse(args)
	if err != nil || help || h {
		fmt.Println(usage)
		os.Exit(0)
	}

	if len(flags.Args()) > 0 {
		return nil, fmt.Errorf("no arguments expected, got: %s", strings.Join(flags.Args(), ","))
	}

	if version {
		err := printVersion()
		if err != nil {
			print("failed to print version: ", err.Error(), "\n")
			os.Exit(1)
		}
		os.Exit(0)
	}

	configureLogger(debugMode)

	repoDir, err := getRepoDir()
	if err != nil {
		return nil, err
	}

	if tag != "" && commit == "" {
		return nil, fmt.Errorf("when using --tag, --commit must be specified")
	}

	if branch != "" && commit == "" {
		return nil, fmt.Errorf("when using --branch, --commit must be specified")
	}

	if commit != "" && tag == "" && branch == "" {
		return nil, fmt.Errorf("when using --commit, --branch or --tag must be specified")
	}

	if verifyOnTip && commit == "" {
		return nil, fmt.Errorf("when using --verify-on-tip, --commit must be specified")
	}

	if verifyOnTip && branch == "" {
		return nil, fmt.Errorf("when using --verify-on-tip, --branch must be specified")
	}

	validateOptions := &gitverify.ValidateOptions{
		Commit:       commit,
		Tag:          tag,
		Branch:       branch,
		VerifyOnHEAD: verifyOnHEAD,
		VerifyOnTip:  verifyOnTip,
	}

	if configFilePath != "" || repoUri != "" {
		if configFilePath == "" {
			return nil, fmt.Errorf("--config-file must be used with --repository-uri")
		}

		if repoUri == "" {
			return nil, fmt.Errorf("--repository-uri must be used with --config-file\n")
		}

		// TODO consider supporting local state
		localState = false
	}

	if commit != "" {
		// TODO consider supporting local state
		localState = false
	}

	return &VerifyOptions{
		repoDir:         repoDir,
		validateOptions: validateOptions,
		configFilePath:  configFilePath,
		repoUri:         repoUri,
		localState:      localState,
	}, nil
}

type GenerateOptions struct {
	repoDir        string
	useSHA512      bool
	configFilePath string
	repoUri        string
}

func parseGenerateOptions(args []string) (*GenerateOptions, error) {
	var debugMode, useSHA512, help, h bool
	var configFilePath, repoUri string
	flags := flag.NewFlagSet("generate", flag.ExitOnError)
	flags.BoolVar(&debugMode, "debug", false, "")
	flags.BoolVar(&useSHA512, "sha512", false, "")
	flags.StringVar(&configFilePath, "config-file", "", "")
	flags.StringVar(&repoUri, "repository-uri", "", "")

	flags.BoolVar(&help, "help", false, "")
	flags.BoolVar(&h, "h", false, "")

	err := flags.Parse(args)
	if err != nil || help || h {
		fmt.Println(usage)
		os.Exit(0)
	}

	if len(flags.Args()) > 0 {
		return nil, fmt.Errorf("no arguments expected, got: %s", strings.Join(flags.Args(), ","))
	}

	configureLogger(debugMode)

	repoDir, err := getRepoDir()
	if err != nil {
		return nil, err
	}

	return &GenerateOptions{
		repoDir:        repoDir,
		useSHA512:      useSHA512,
		configFilePath: configFilePath,
		repoUri:        repoUri,
	}, nil
}

func getRepoDir() (string, error) {
	basePath, err := os.Getwd()
	if err != nil {
		return "", err
	}

	repoDir, _, err := gitkit.GetRootPathOfLocalGitRepo(basePath)
	if err != nil {
		return "", err
	}

	return repoDir, nil
}

func configureLogger(debugMode bool) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if debugMode {
		opts.Level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)
}

func verify(opts *VerifyOptions) error {
	repoDir := opts.repoDir
	validateOptions := opts.validateOptions
	configFilePath := opts.configFilePath
	repoUri := opts.repoUri
	localState := opts.localState

	fmt.Println("validating...")

	repo, err := gitkit.OpenRepoInLocalPath(repoDir)
	if err != nil {
		return err
	}

	state := gitkit.LoadRepoState(repo)
	sha1Hash := githash.NewGitHashFromRepoState(state, sha1.New())
	sha512Hash := githash.NewGitHashFromRepoState(state, sha512.New())

	var localStatePath string

	var repoConfig *gitverify.RepoConfig
	repoConfig, repoUri, err = loadRepoConfig(repo, configFilePath, repoUri)
	if err != nil {
		return err
	}

	err = gitverify.Verify(repo, state, repoConfig, sha1Hash, sha512Hash, validateOptions)
	if err != nil {
		return err
	}

	if localState {
		if configFilePath == "" {
			forge, org, repoName := gitverify.InferForgeOrgAndRepo(repo)
			localStatePath, err = gitverify.GetLocalStatePath(forge, org, repoName)
			if err != nil {
				return err
			}
		}

		err = gitverify.VerifyLocalState(repo, state, repoConfig, repoUri, localStatePath, sha1Hash, sha512Hash)
		if err != nil {
			return fmt.Errorf("failed to verify local state: %w", err)
		}

		err = gitverify.SaveLocalState(repo, state, repoConfig, repoUri, localStatePath, sha1Hash, sha512Hash)
		if err != nil {
			return fmt.Errorf("failed to save local state: %w", err)
		}
	}

	fmt.Println("OK")
	return nil
}

func loadRepoConfig(repo *git.Repository, configFilePath string, inputRepoUri string) (config *gitverify.RepoConfig, repoUri string, err error) {
	repoUri = inputRepoUri
	if configFilePath == "" {
		forge, org, repoName := gitverify.InferForgeOrgAndRepo(repo)
		var err error
		configFilePath, err = gitverify.GetConfigPath(forge, org)
		if err != nil {
			return nil, "", err
		}

		repoUri = "git+https://" + forge + "/" + org + "/" + repoName + ".git"
	}

	parsedConfig, err := gitverify.LoadConfig(configFilePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load config: %w", err)
	}

	repoConfig, err := gitverify.LoadRepoConfig(parsedConfig, repoUri)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse config %s: %w", configFilePath, err)
	}

	return repoConfig, repoUri, nil
}

func afterCandidates(opts *GenerateOptions) error {
	repoDir := opts.repoDir
	useSHA512 := opts.useSHA512

	repo, err := gitkit.OpenRepoInLocalPath(repoDir)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	repoConfig, _, err := loadRepoConfig(repo, opts.configFilePath, opts.repoUri)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	candidates, err := gitverify.AfterCandidates(repo, repoConfig, useSHA512)
	if err != nil {
		return fmt.Errorf("failed to find after candidates: %w", err)
	}

	refs, err := repo.References()
	if err != nil {
		return fmt.Errorf("failed to list refs: %w", err)
	}

	refMap := make(map[plumbing.Hash][]string)
	err = refs.ForEach(func(reference *plumbing.Reference) error {
		ref, found := refMap[reference.Hash()]

		if found {
			refMap[reference.Hash()] = append(ref, reference.Name().String())
		} else {
			refMap[reference.Hash()] = []string{reference.Name().String()}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to process refs: %w", err)
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Branch != nil && candidates[j].Branch != nil {
			return *candidates[i].Branch < *candidates[j].Branch
		}

		if candidates[i].Branch != nil {
			return true
		}

		if candidates[j].Branch != nil {
			return false
		}

		return *candidates[i].SHA1 < *candidates[j].SHA1
	})

	for i, candidate := range candidates {
		refs, found := refMap[plumbing.NewHash(*candidate.SHA1)]
		if found {
			fmt.Printf("%s %s\n", *candidate.SHA1, strings.Join(refs, ","))

			if candidates[i].Branch == nil {
				allBranches := hashset.New[string]()
				var branchName string
				for _, ref := range refs {
					branchName, found = gitverify.BranchName(ref)
					if found {
						allBranches.Add(branchName)
					}
				}

				if allBranches.Size() == 1 {
					candidates[i].Branch = &allBranches.Values()[0]
				}
			}
		}
	}

	data, err := json.Marshal(candidates)
	if err != nil {
		return fmt.Errorf("failed to marshal candidates: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func exemptTags(opts *GenerateOptions) (string, error) {
	repoDir := opts.repoDir
	useSHA512 := opts.useSHA512

	repo, err := gitkit.OpenRepoInLocalPath(repoDir)
	if err != nil {
		return "", err
	}

	state := gitkit.LoadRepoState(repo)
	sha1Hash := githash.NewGitHashFromRepoState(state, sha1.New())
	sha512Hash := githash.NewGitHashFromRepoState(state, sha512.New())
	exemptTags, err := gitverify.ComputeExemptTags(repo, state, sha1Hash, sha512Hash, useSHA512)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(exemptTags)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func printVersion() error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("no version information")
	}

	for _, kv := range info.Settings {
		if strings.HasPrefix(kv.Key, "vcs") {
			fmt.Printf("%s: %s\n", kv.Key, kv.Value)
		}
	}

	return nil
}
