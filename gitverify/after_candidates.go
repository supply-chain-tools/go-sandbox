package gitverify

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/supply-chain-tools/go-sandbox/githash"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/hashset"
)

func AfterCandidates(repo *git.Repository, useSHA256 bool) ([]After, error) {
	state := gitkit.LoadRepoState(repo)
	sha256Hash := githash.NewGitHashFromRepoState(state, sha256.New())

	pointedTo := hashset.New[plumbing.Hash]()

	for _, commit := range state.CommitMap {
		for _, parent := range commit.ParentHashes {
			pointedTo.Add(parent)
		}
	}

	candidates := make([]After, 0)
	for _, commit := range state.CommitMap {
		if !pointedTo.Contains(commit.Hash) {

			sha1 := commit.Hash.String()

			var hexSHA256 *string = nil
			if useSHA256 {
				sha256, err := sha256Hash.CommitSum(commit.Hash)
				if err != nil {
					return nil, err
				}

				h := hex.EncodeToString(sha256)
				hexSHA256 = &h
			}

			candidates = append(candidates, After{
				SHA1:   &sha1,
				SHA256: hexSHA256,
			})
		}
	}

	return candidates, nil
}
