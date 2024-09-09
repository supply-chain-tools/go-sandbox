package gitverify

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/supply-chain-tools/go-sandbox/githash"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"os"
	"path/filepath"
)

type LocalState struct {
	Tags     []ExemptTag `json:"tags"`
	Branches []ExemptTag `json:"branches"`
}

func GetLocalStatePath(forge string, org string, repoName string) (string, error) {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDirectory, ".config", "gitverify", forge, org, repoName, "local.json"), nil
}

func SaveLocalState(repo *git.Repository, state *gitkit.RepoState, repoConfig *RepoConfig, repoUri string, localPath string, gitHashSHA1 githash.GitHash, gitHashSHA256 githash.GitHash) error {
	// TODO time of check, time of save issues
	tags, err := ComputeExemptTags(repo, state, gitHashSHA1, gitHashSHA256, true)
	if err != nil {
		return err
	}

	protectedBranches, err := computeProtectedBranches(repo, repoConfig, gitHashSHA1, gitHashSHA256)
	if err != nil {
		return err
	}

	localState := &LocalState{
		Tags:     tags,
		Branches: protectedBranches,
	}

	data, err := json.Marshal(localState)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(localPath), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.WriteFile(localPath, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func VerifyLocalState(repo *git.Repository, state *gitkit.RepoState, repoConfig *RepoConfig, repoUri string, localPath string, gitHashSHA1 githash.GitHash, gitHashSHA256 githash.GitHash) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		} else {
			return err
		}
	}

	localState := LocalState{}
	err = json.Unmarshal(data, &localState)
	if err != nil {
		return err
	}

	newTags, err := ComputeExemptTags(repo, state, gitHashSHA1, gitHashSHA256, true)
	if err != nil {
		return err
	}

	newTagMap := make(map[string]ExemptTag)
	for _, tag := range newTags {
		_, found := newTagMap[tag.Ref]
		if found {
			return fmt.Errorf("duplicate tag '%s'", tag.Ref)
		}

		newTagMap[tag.Ref] = tag
	}

	for _, tag := range localState.Tags {
		newTag, found := newTagMap[tag.Ref]
		if found {
			if newTag.Hash.SHA1 == nil || tag.Hash.SHA1 == nil {
				return fmt.Errorf("tag SHA-1 hashes must be set")
			}

			if newTag.Hash.SHA256 == nil || tag.Hash.SHA256 == nil {
				return fmt.Errorf("tag SHA-256 hashes must be set")
			}

			if *newTag.Hash.SHA1 != *tag.Hash.SHA1 {
				return fmt.Errorf("tag '%s' hash has changed from %s to %s", tag.Ref, *tag.Hash.SHA1, *newTag.Hash.SHA1)
			}

			if *newTag.Hash.SHA256 != *tag.Hash.SHA256 {
				return fmt.Errorf("tag '%s' SHA-256 hash has changed from %s to %s", tag.Ref, *tag.Hash.SHA256, *newTag.Hash.SHA256)
			}
		} else {
			return fmt.Errorf("tag '%s' has been deleted, was %s", tag.Ref, *tag.Hash.SHA1)
		}
	}

	newProtectedBranches, err := computeProtectedBranches(repo, repoConfig, gitHashSHA1, gitHashSHA256)
	if err != nil {
		return err
	}

	newProtectedBranchesMap := make(map[string]ExemptTag)
	for _, branch := range newProtectedBranches {
		_, found := newProtectedBranchesMap[branch.Ref]
		if found {
			return fmt.Errorf("duplicate branch '%s'", branch.Ref)
		}

		newProtectedBranchesMap[branch.Ref] = branch
	}

	for _, branch := range localState.Branches {
		newBranch, found := newProtectedBranchesMap[branch.Ref]
		if found {
			if newBranch.Hash.SHA1 == nil || branch.Hash.SHA1 == nil {
				return fmt.Errorf("branch hashes must be set")
			}

			if newBranch.Hash.SHA256 == nil || branch.Hash.SHA256 == nil {
				return fmt.Errorf("branch SHA-256 hashes must be set")
			}

			c, found := state.CommitMap[plumbing.NewHash(*newBranch.Hash.SHA1)]
			if !found {
				return fmt.Errorf("target commit '%s' not found for %s", *newBranch.Hash.SHA1, branch.Ref)
			}

			visited := hashset.New[plumbing.Hash]()
			visited.Add(c.Hash)
			queue := []*object.Commit{c}

			for {
				if len(queue) == 0 {
					return fmt.Errorf("new state of %s is not a descendant of the local state", branch.Ref)
				}

				current := queue[0]
				queue = queue[1:]

				if current.Hash.String() == *branch.Hash.SHA1 {
					hashSHA256, err := gitHashSHA256.CommitSum(current.Hash)
					if err != nil {
						return err
					}

					if hex.EncodeToString(hashSHA256) != *branch.Hash.SHA256 {
						return fmt.Errorf("SHA-256 does not match SHA-1 for %s", branch.Ref)
					}

					break
				}

				if len(current.ParentHashes) > 0 {
					// Rather than looking for any ancestor, only look at the first parent recursively.
					// This assumes that changes are either merged into the protected branch or commited to the
					// protected branch directly. That the stored commit occur some other place is not considered
					// sufficient.
					parentHash := current.ParentHashes[0]
					if !visited.Contains(parentHash) {
						parent, found := state.CommitMap[parentHash]
						if !found {
							return fmt.Errorf("target parent hash not found: %s", parentHash)
						}

						queue = append(queue, parent)
						visited.Add(parentHash)
					}
				}
			}

			if *branch.Hash.SHA1 != *newBranch.Hash.SHA1 {
				fmt.Printf("%s: git log -p --full-diff %s...%s\n", branch.Ref, *branch.Hash.SHA1, *newBranch.Hash.SHA1)
			}
		} else {
			return fmt.Errorf("protected branch '%s' has been deleted, was %s", branch.Ref, *branch.Hash.SHA1)
		}
	}

	return nil
}

func computeProtectedBranches(repo *git.Repository, config *RepoConfig, gitHashSHA1 githash.GitHash, gitHashSHA256 githash.GitHash) ([]ExemptTag, error) {
	remotes, err := repo.References()
	if err != nil {
		return nil, err
	}

	result := make([]ExemptTag, 0)
	err = remotes.ForEach(func(reference *plumbing.Reference) error {
		isProtected, _ := isProtected(reference, config)

		if isProtected {

			hashSHA1 := reference.Hash().String()
			h, err := gitHashSHA256.CommitSum(reference.Hash())
			if err != nil {
				return err
			}
			hashSHA256 := hex.EncodeToString(h)

			result = append(result, ExemptTag{
				Ref: reference.Name().String(),
				Hash: Digests{
					SHA1:   &hashSHA1,
					SHA256: &hashSHA256,
				},
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
