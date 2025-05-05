package pr

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/supply-chain-tools/go-sandbox/githash"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/gitverify"
	"os"
	"os/exec"
	"strings"
)

func Create(tag string, commit plumbing.Hash, branch string, message string) error {
	repo, err := openRepoFromCwd()
	if err != nil {
		return err
	}

	gh := githash.NewGitHash(repo, sha512.New())
	objectSHA512, err := gh.CommitSum(commit)
	if err != nil {
		return err
	}
	objectSHA512Hex := hex.EncodeToString(objectSHA512)

	_, orgName, repoName := gitverify.InferForgeOrgAndRepo(repo)
	repoURI := wrapGitURL("https://github.com/" + orgName + "/" + repoName)

	sb := strings.Builder{}
	if message != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", message))
	}

	sb.WriteString("Type: pr\n")
	sb.WriteString(fmt.Sprintf("Base-repo: %s\n", repoURI))
	sb.WriteString(fmt.Sprintf("Base-branch: %s\n", branch))
	sb.WriteString(fmt.Sprintf("Object-sha512: %s\n", objectSHA512Hex))

	m := sb.String()
	fmt.Print(m)

	err = createTag(m, tag, commit.String())
	if err != nil {
		return err
	}

	return nil
}

func Approve(tag string, prHash plumbing.Hash, message string) error {
	repo, err := openRepoFromCwd()
	if err != nil {
		return err
	}

	gh := githash.NewGitHash(repo, sha512.New())
	objectSHA512, err := gh.TagSum(prHash)
	if err != nil {
		return err
	}
	objectSHA512Hex := hex.EncodeToString(objectSHA512)

	sb := strings.Builder{}
	if message != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", message))
	}

	sb.WriteString("Type: pr-approve\n")
	sb.WriteString(fmt.Sprintf("Object-sha512: %s\n", objectSHA512Hex))

	m := sb.String()
	fmt.Print(m)

	err = createTag(m, tag, prHash.String())
	if err != nil {
		return err
	}

	return nil
}

func Merge(prHash plumbing.Hash, message string) error {
	repo, err := openRepoFromCwd()
	if err != nil {
		return err
	}

	iter, err := repo.Tags()
	if err != nil {
		return err
	}

	sb := strings.Builder{}
	if message != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", message))
	}

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Hash() == prHash {
			return nil
		}

		obj, err := repo.TagObject(ref.Hash())
		if err != nil {
			return err
		}

		sb.WriteString(fmt.Sprintf("Approve-tag: object %s\n", ref.Hash().String()))
		sb.WriteString(" type tag\n")
		sb.WriteString(fmt.Sprintf(" tag %s\n", obj.Name))

		b := new(bytes.Buffer)
		err = obj.Tagger.Encode(b)
		if err != nil {
			return err
		}

		sb.WriteString(fmt.Sprintf(" tagger %s\n", b.String()))

		messageLines := strings.Split(obj.Message, "\n")
		for i, line := range messageLines {
			if i < len(messageLines)-1 {
				sb.WriteString(fmt.Sprintf("\n %s", line))
			} else {
				// FIXME
				sb.WriteString("\n")
			}
		}

		signatureLines := strings.Split(obj.PGPSignature, "\n")
		for i, line := range signatureLines {
			if i < len(signatureLines)-1 {
				sb.WriteString(fmt.Sprintf("\n %s", line))
			} else {
				// FIXME
				sb.WriteString("\n")
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	m := sb.String()
	fmt.Print(m)

	command := []string{"git", "merge", "-S", "-m", m, prHash.String()}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
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
func wrapGitURL(url string) string {
	// https://spdx.github.io/spdx-spec/v2.3/package-information/#77-package-download-location-field
	return "git+" + url + ".git"
}

func wrapGitURLTagOrCommit(url string, tagOrCommit string) string {
	// https://spdx.github.io/spdx-spec/v2.3/package-information/#77-package-download-location-field
	return "git+" + url + ".git@" + tagOrCommit
}

func createTag(message string, tag string, hash string) error {
	command := []string{"git", "tag", "-s", "-m", message, tag, hash}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
