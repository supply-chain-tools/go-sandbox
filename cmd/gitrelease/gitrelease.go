package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/gitrelease"
	"github.com/supply-chain-tools/go-sandbox/gitverify"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const usage = `Usage:
    gitrelease [options]

Options:
    --keyfile      Path to SSH private key to use
    --init         Initialise if no other tags should exist
    --debug        Enable debug logging
    -h, --help     Show help message`

const (
	tagRegex     = "v[0-9]+\\.[0-9]+\\.[0-9]+"
	hashRegex    = "[0-9a-f]{40}"
	keyfileRegex = "(~/|../)?[a-zA-Z0-9-_/.]+"
	sshNamespace = "gitrelease"
)

func main() {
	optionsAndArgs, err := parseOptionsAndArgs()
	if err != nil {
		print("Failed to parse input: ", err.Error(), "\n")
		os.Exit(1)
	}

	tagName := optionsAndArgs.tagName
	hash := optionsAndArgs.commitHash
	keyfilePath := optionsAndArgs.keyfilePath
	countersignPath := optionsAndArgs.countersignPath

	repo, err := openRepoFromCwd()
	if err != nil {
		print("Unable to open repo: ", err.Error(), "\n")
		os.Exit(1)
	}

	previousTag, previousTagFound, err := gitrelease.FindPreviousTag(repo)
	if err != nil {
		print("Unable to find previous tag ", tagName, ": ", err.Error(), "\n")
		os.Exit(1)
	}
	if !previousTagFound && !optionsAndArgs.init {
		print("No previous tag found, use --init to initialize.\n")
		os.Exit(1)
	}

	if previousTagFound && optionsAndArgs.init {
		print("--init set, but there are existing tags")
		os.Exit(1)
	}

	_, orgName, repoName := gitverify.InferForgeOrgAndRepo(repo)
	repoPath := "https://github.com/" + orgName + "/" + repoName

	tagMetadata, err := gitrelease.CreatePreviousTagPayload(repo, tagName, hash, previousTag, repoPath)
	if err != nil {
		print("Unable to create previous tag payload: ", err.Error(), "\n")
		os.Exit(1)
	}

	command, err := createTag(repo, tagName, hash, tagMetadata, countersignPath != "")
	if err != nil {
		print("Unable to create tag ", tagName, ": ", err.Error(), "\n")
		os.Exit(1)
	}

	tag, err := getTag(repo, tagName)
	if err != nil {
		print("Unable to get tag ", tagName, ": ", err.Error(), "\n")
		os.Exit(1)
	}

	tagLink, err := gitrelease.CreateTagLink(repo, hash, tag, repoPath, command, tagMetadata)
	if err != nil {
		print("Unable to create tag.link: ", err.Error(), "\n")
		os.Exit(1)
	}

	var existingEnvelope *gitrelease.Envelope = nil
	if countersignPath != "" {
		tagLink, existingEnvelope, err = validateCountersignFile(countersignPath, tagLink)
		if err != nil {
			print("Unable to validate countersign file: ", err.Error(), "\n")
			os.Exit(1)
		}
	}

	err = displayTagLinkAndExitIfUserDoesNotWantToProceed(tagLink)
	if err != nil {
		print("Unable to proceed: ", err.Error(), "\n")
		os.Exit(1)
	}

	payload, signature, err := serializeAndSign(tagLink, keyfilePath)
	if err != nil {
		print("Failed to sign data: ", err.Error(), "\n")
		os.Exit(1)
	}

	var envelope *gitrelease.Envelope = nil
	if countersignPath == "" {
		envelope = gitrelease.CreateEnvelope(payload, signature)
	} else {
		envelope = existingEnvelope
		envelope.Signatures = append(envelope.Signatures,
			gitrelease.Signature{
				Sig: signature,
			},
		)
	}

	cwd, err := os.Getwd()
	if err != nil {
		print("Failed to get cwd: ", err.Error(), "\n")
		os.Exit(1)
	}

	err = writeFile(envelope, cwd)
	if err != nil {
		print("Unable to write output file: ", err.Error(), "\n")
		os.Exit(1)
	}
}

type OptionsAndArgs struct {
	tagName         string
	commitHash      plumbing.Hash
	keyfilePath     string
	countersignPath string
	init            bool
}

func parseOptionsAndArgs() (*OptionsAndArgs, error) {
	flag.Usage = func() {
		fmt.Println(usage)
	}

	flags := flag.NewFlagSet("all", flag.ExitOnError)
	var help, h, debugMode, init bool
	var keyfilePath, countersignPath string
	flags.StringVar(&keyfilePath, "keyfile", "", "")
	flags.StringVar(&countersignPath, "countersign", "", "")
	flags.BoolVar(&help, "help", false, "")
	flags.BoolVar(&h, "h", false, "")
	flags.BoolVar(&debugMode, "debug", false, "")
	flags.BoolVar(&init, "init", false, "")

	err := flags.Parse(os.Args[1:])
	if err != nil || help || h {
		fmt.Println(usage)
		os.Exit(0)
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if debugMode {
		opts.Level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	if len(flags.Args()) < 2 {
		return nil, fmt.Errorf("not enough arguments")
	}

	if len(flags.Args()) > 2 {
		return nil, fmt.Errorf("too many arguments")
	}

	tagName := flags.Args()[0]
	match, _ := regexp.MatchString(tagRegex, tagName)
	if !match {
		return nil, fmt.Errorf("invalid tag '%s': it must be on the form 'vX.Y.Z'", tagName)
	}

	if len(flags.Args()[1]) != 40 {
		return nil, fmt.Errorf("hash must be 40 characters long")
	}

	inputHash := flags.Args()[1]
	match, _ = regexp.MatchString(hashRegex, inputHash)
	if !match {
		return nil, fmt.Errorf("invalid commit '%s': it must be hex encoded", inputHash)
	}

	if keyfilePath == "" {
		return nil, fmt.Errorf("--keyfile must be specified")
	}

	match, _ = regexp.MatchString(keyfileRegex, keyfilePath)
	if !match {
		return nil, fmt.Errorf("invalid character in keyfile path '%s'", keyfilePath)
	}

	if strings.HasPrefix(keyfilePath, "-") || strings.Contains(keyfilePath, "--") {
		return nil, fmt.Errorf("invalid format of keyfile path '%s'", keyfilePath)
	}

	keyfilePath, err = filepath.Abs(keyfilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to get absolute path of '%s': %w", keyfilePath, err)
	}

	_, err = os.Stat(keyfilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to verify keyfile': %w", err)
	}

	commitHash := plumbing.NewHash(flags.Args()[1])

	return &OptionsAndArgs{
		tagName:         tagName,
		commitHash:      commitHash,
		keyfilePath:     keyfilePath,
		countersignPath: countersignPath,
		init:            init,
	}, nil
}

func openRepoFromCwd() (*git.Repository, error) {
	basePath, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	repoDir, found, err := gitkit.GetRootPathOfLocalGitRepo(basePath)
	if err != nil {
		return nil, fmt.Errorf("unable infer git root from %s: %w", basePath, err)
	}

	if !found {
		return nil, fmt.Errorf("not in a git repo %s", basePath)
	}

	repo, err := gitkit.OpenRepoInLocalPath(repoDir)
	if err != nil {
		return nil, fmt.Errorf("unable to open repo %s: %w", repoDir, err)
	}

	return repo, nil
}

func createTag(repo *git.Repository, tag string, commitHash plumbing.Hash, previousTagPayload *gitrelease.TagMetadata, validateRun bool) ([]string, error) {
	match, _ := regexp.MatchString(tagRegex, tag)
	if !match {
		return nil, fmt.Errorf("invalid tag '%s': it must be on the form 'vX.Y.Z'", tag)
	}

	c, err := repo.CommitObject(commitHash)
	if err != nil || c.Hash != commitHash {
		return nil, fmt.Errorf("unable to find commit %s", commitHash.String())
	}

	_, err = repo.Tag(tag)
	if validateRun {
		if err != nil {
			return nil, err
		}
	} else {
		if err != nil {
			// do nothing, FIXME different errors
		} else {
			return nil, fmt.Errorf("tag %s already exists", tag)
		}
	}

	p, err := json.Marshal(previousTagPayload)
	if err != nil {
		return nil, err
	}
	payload := base64.StdEncoding.EncodeToString(p)

	command := []string{"git", "tag", "-s", "-m", fmt.Sprintf("Release %s\n\nTag-metadata: %s", tag, payload), tag, commitHash.String()}

	if !validateRun {
		fmt.Printf("Executing command: %s\n", command)
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Run()
		if err != nil {
			return nil, err
		}
	}

	return command, nil
}

func getTag(repo *git.Repository, tagName string) (*object.Tag, error) {
	tagRef, err := repo.Tag(tagName)
	if err != nil {
		return nil, err
	}

	tag, err := repo.TagObject(tagRef.Hash())
	if err != nil {
		return nil, err
	}

	return tag, nil
}

func displayTagLinkAndExitIfUserDoesNotWantToProceed(tagLink *gitrelease.TagLink) error {
	data, err := json.MarshalIndent(tagLink, "  ", "  ")
	if err != nil {
		return err
	}
	fmt.Println("tag.link:")
	fmt.Println("  " + string(data))

	fmt.Printf("Sign and write [Y/n]? ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	if !(response == "Y\n" || response == "y\n" || response == "\n") {
		return fmt.Errorf("user aborted")
	}

	return nil
}

func validateCountersignFile(countersignPath string, tagLink *gitrelease.TagLink) (*gitrelease.TagLink, *gitrelease.Envelope, error) {
	data, err := os.ReadFile(countersignPath)
	if err != nil {
		return nil, nil, err
	}

	envelope := gitrelease.Envelope{}
	err = json.Unmarshal(data, &envelope)
	if err != nil {
		return nil, nil, err
	}

	existing, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return nil, nil, err
	}

	existingTagLink := gitrelease.TagLink{}
	err = json.Unmarshal(existing, &existingTagLink)
	if err != nil {
		return nil, nil, err
	}

	subject, err := json.Marshal(tagLink.Subject)
	if err != nil {
		return nil, nil, err
	}

	existingSubject, err := json.Marshal(existingTagLink.Subject)
	if err != nil {
		return nil, nil, err
	}

	if !bytes.Equal(existingSubject, subject) {
		return nil, nil, fmt.Errorf("countersign and computed subject do not match")
	}

	// FIXME verify the rest of the file

	return &existingTagLink, &envelope, err
}

func serializeAndSign(tagLink *gitrelease.TagLink, keyfilePath string) (payload string, signature string, err error) {
	serialized, err := json.Marshal(tagLink)
	if err != nil {
		return "", "", err
	}

	preAuthenticationEncoding, err := gitrelease.PreAuthenticationEncoding(serialized)
	if err != nil {
		return "", "", fmt.Errorf("unable to create signature payload: %w", err)
	}

	payload = base64.StdEncoding.EncodeToString(serialized)

	tmpFile, err := os.CreateTemp("", "tag.link")
	if err != nil {
		return "", "", fmt.Errorf("unable to create temporary file: %w", err)
	}

	_, err = tmpFile.Write(preAuthenticationEncoding)
	if err != nil {
		return "", "", fmt.Errorf("unable to write to temporary file: %w", err)
	}
	err = tmpFile.Sync()
	if err != nil {
		return "", "", fmt.Errorf("unable to sync temporary file: %w", err)
	}

	command := []string{"ssh-keygen", "-Y", "sign", "-n", sshNamespace, "-f", keyfilePath, tmpFile.Name()}

	fmt.Printf("Executing command: %s\n", command)
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return "", "", err
	}

	signatureFile := tmpFile.Name() + ".sig"
	sig, err := os.ReadFile(signatureFile)
	if err != nil {
		return "", "", fmt.Errorf("unable to read signature from %s: %w", signatureFile, err)
	}

	signature = base64.StdEncoding.EncodeToString(sig)

	err = os.Remove(tmpFile.Name())
	if err != nil {
		return "", "", fmt.Errorf("unable to delete temporary file %s: %w", tmpFile.Name(), err)
	}

	err = os.Remove(tmpFile.Name() + ".sig")
	if err != nil {
		return "", "", fmt.Errorf("unable to delete temporary signature %s: %w", signatureFile, err)
	}

	// FIXME verify signature

	return payload, signature, nil
}

func writeFile(envelope *gitrelease.Envelope, path string) error {
	data, err := json.Marshal(*envelope)
	if err != nil {
		return err
	}

	location := filepath.Join(path, "tag.link.intoto.jsonl")
	err = os.WriteFile(location, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
