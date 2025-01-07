package gitverify

import (
	"crypto/sha512"
	"encoding/hex"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/supply-chain-tools/go-sandbox/githash"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"strings"
)

func AfterCandidates(repo *git.Repository, repoConfig *RepoConfig, useSHA512 bool) ([]After, error) {
	state := gitkit.LoadRepoState(repo)
	sha512Hash := githash.NewGitHashFromRepoState(state, sha512.New())

	pointedTo := hashset.New[plumbing.Hash]()

	for _, commit := range state.CommitMap {
		for _, parent := range commit.ParentHashes {
			pointedTo.Add(parent)
		}
	}

	remotes, err := repo.References()
	if err != nil {
		return nil, err
	}

	candidates := make([]After, 0)
	protectedHashes := hashset.New[plumbing.Hash]()

	err = remotes.ForEach(func(reference *plumbing.Reference) error {
		if !strings.HasPrefix(reference.Name().String(), "refs/remotes/origin/") {
			// local branches might not be up-to-date, so using remotes/origin
			return nil
		}

		isProtected, branchName := isProtected(reference, repoConfig)

		if isProtected {
			sha1 := reference.Hash().String()
			var hexSHA512 *string = nil

			if useSHA512 {
				sha2, err := sha512Hash.CommitSum(reference.Hash())
				if err != nil {
					return err
				}

				h := hex.EncodeToString(sha2)
				hexSHA512 = &h
			}

			candidates = append(candidates, After{
				SHA1:   &sha1,
				SHA512: hexSHA512,
				Branch: &branchName,
			})

			protectedHashes.Add(reference.Hash())
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, commit := range state.CommitMap {
		if !pointedTo.Contains(commit.Hash) {

			sha1 := commit.Hash.String()

			if protectedHashes.Contains(commit.Hash) {
				continue
			}

			var hexSHA512 *string = nil
			if useSHA512 {
				sha2, err := sha512Hash.CommitSum(commit.Hash)
				if err != nil {
					return nil, err
				}

				h := hex.EncodeToString(sha2)
				hexSHA512 = &h
			}

			candidates = append(candidates, After{
				SHA1:   &sha1,
				SHA512: hexSHA512,
			})
		}
	}

	return candidates, nil
}
