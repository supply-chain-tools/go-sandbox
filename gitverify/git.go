package gitverify

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
)

type SignatureType string

const (
	SignatureTypeGPG     SignatureType = "gpg"
	SignatureTypeSSH     SignatureType = "ssh"
	SignatureTypeNone    SignatureType = "none"
	SignatureTypeSMime   SignatureType = "smime"
	SignatureTypeUnknown SignatureType = "unknown"
	namespaceSSH         string        = "git"
)

type CommitData struct {
	SignatureType                   SignatureType
	Ignore                          bool
	VerifiedToNotHaveContentChanges bool
}

func InferForgeOrgAndRepo(repo *git.Repository) (forge string, org string, repoName string) {
	remote, err := repo.Remote("origin")
	if err != nil {
		log.Fatal(err)
	}
	urls := remote.Config().URLs
	if len(urls) != 1 {
		log.Fatal("Expected exactly one remote url")
	}

	org, repoName, err = getGitHubOrgRepo(urls[0])
	if err != nil {
		log.Fatal(err)
	}

	return gitHubForgeId, org, repoName
}

func getGitHubOrgRepo(url string) (org string, repoName string, err error) {
	const httpsPrefix = "https://github.com/"
	const sshPrefix = "git@github.com:"

	if !strings.HasPrefix(url, httpsPrefix) && !strings.HasPrefix(url, sshPrefix) {
		return "", "", fmt.Errorf("GitHub URL does not start with 'https://github.com/' or 'git@github.com:': %s", url)
	}

	var suffix string
	if strings.HasPrefix(url, httpsPrefix) {
		suffix = url[len(httpsPrefix):]
	} else {
		suffix = url[len(sshPrefix):]
	}

	suffix = strings.TrimSuffix(suffix, ".git")
	parts := strings.Split(suffix, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected URL format: %s", url)
	}

	org = parts[0]
	repoName = parts[1]

	return org, repoName, nil
}

func ignoreCommitAndParents(commit *object.Commit, commitMap map[plumbing.Hash]*CommitData, state *gitkit.RepoState) error {
	queue := []*object.Commit{commit}

	for {
		if len(queue) == 0 {
			break
		}

		current := queue[0]
		queue = queue[1:]

		c, found := commitMap[current.Hash]
		if found && c.Ignore {
			continue
		}

		for _, parentHash := range current.ParentHashes {
			parent, found := state.CommitMap[parentHash]
			if !found {
				return fmt.Errorf("failed to get parent commit %s", parentHash)
			}

			queue = append(queue, parent)
		}

		signatureType, err := inferSignatureType(current.PGPSignature)
		if err != nil {
			return err
		}

		commitMap[current.Hash] = &CommitData{
			SignatureType: signatureType,
			Ignore:        true,
		}
	}

	return nil
}

func inferSignatureType(signature string) (SignatureType, error) {
	if strings.HasPrefix(signature, "-----BEGIN SSH SIGNATURE-----") {
		return SignatureTypeSSH, nil
	} else if strings.HasPrefix(signature, "-----BEGIN PGP SIGNATURE-----") {
		return SignatureTypeGPG, nil
	} else if signature == "" {
		return SignatureTypeNone, nil
	} else {
		return SignatureTypeUnknown, fmt.Errorf("unknown signature type: '%s'", signature)
	}
}

func gitDiff(a string, b string) (string, error) {
	match, err := regexp.MatchString(hexSHA1Regex, a)
	if err != nil {
		return "", err
	}
	if match == false {
		return "", fmt.Errorf("expected a hash, got '%s'", a)
	}

	match, err = regexp.MatchString(hexSHA1Regex, b)
	if err != nil {
		return "", err
	}
	if match == false {
		return "", fmt.Errorf("expected a hash, got '%s'", b)
	}

	var stdout bytes.Buffer
	command := []string{"git", "diff", a + "..." + b}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = &stdout
	err = cmd.Run()

	if err != nil {
		return "", err
	}

	return stdout.String(), err
}

func verifyMergeCommitNoContentChanges(commit *object.Commit) error {
	if len(commit.ParentHashes) != 2 {
		return fmt.Errorf("expected 2 parent hashes, got %d", len(commit.ParentHashes))
	}

	a := commit.ParentHashes[0].String()
	b := commit.ParentHashes[1].String()
	treeHash, err := gitMergeTree(a, b)
	if err != nil {
		return err
	}

	if commit.TreeHash.String() != treeHash {
		return fmt.Errorf("expected tree hash '%s', got '%s'", commit.TreeHash.String(), treeHash)
	}

	return nil
}

func gitMergeTree(a string, b string) (string, error) {
	match, err := regexp.MatchString(hexSHA1Regex, a)
	if err != nil {
		return "", err
	}
	if match == false {
		return "", fmt.Errorf("expected a hash, got '%s'", a)
	}

	match, err = regexp.MatchString(hexSHA1Regex, b)
	if err != nil {
		return "", err
	}
	if match == false {
		return "", fmt.Errorf("expected a hash, got '%s'", b)
	}

	var buffer bytes.Buffer
	command := []string{"git", "merge-tree", a, b}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = &buffer
	cmd.Stdout = &buffer

	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return "", fmt.Errorf("git merge-tree failed with exit code %d: %s", exitError.ExitCode(), buffer.String())
		} else {
			return "", fmt.Errorf("git merge-tree failed: output=%s, err=%w", buffer.String(), err)
		}
	}

	result := strings.TrimRight(buffer.String(), "\r\n")

	match, err = regexp.MatchString(hexSHA1Regex, result)
	if err != nil {
		return "", err
	}
	if match == false {
		return "", fmt.Errorf("expected a hash to be returned from merge-tree, got '%s'", result)
	}

	return result, nil
}

func gitMergeBase(a string, b string) (string, error) {
	match, err := regexp.MatchString(hexSHA1Regex, a)
	if err != nil {
		return "", err
	}
	if match == false {
		return "", fmt.Errorf("expected a hash, got '%s'", a)
	}

	match, err = regexp.MatchString(hexSHA1Regex, b)
	if err != nil {
		return "", err
	}
	if match == false {
		return "", fmt.Errorf("expected a hash, got '%s'", b)
	}

	var buffer bytes.Buffer
	command := []string{"git", "merge-base", a, b}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = &buffer
	cmd.Stdout = &buffer

	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return "", fmt.Errorf("git merge-base failed with exit code %d: %s", exitError.ExitCode(), buffer.String())
		} else {
			return "", fmt.Errorf("git merge-base failed: output=%s, err=%w", buffer.String(), err)
		}
	}

	result := strings.TrimRight(buffer.String(), "\r\n")

	match, err = regexp.MatchString(hexSHA1Regex, result)
	if err != nil {
		return "", err
	}
	if match == false {
		return "", fmt.Errorf("expected a hash to be returned from merge-base, got '%s'", result)
	}

	return result, nil
}
