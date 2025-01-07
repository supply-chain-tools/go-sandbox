package gitverify

import (
	"encoding/hex"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/supply-chain-tools/go-sandbox/githash"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"sort"
)

type ExemptTag struct {
	Ref  string  `json:"ref"`
	Hash Digests `json:"hash"`
}

func ComputeExemptTags(repo *git.Repository, state *gitkit.RepoState, gitHashSHA1 githash.GitHash, gitHashSHA512 githash.GitHash, useSHA512 bool) ([]ExemptTag, error) {
	tags, err := repo.Tags()
	if err != nil {
		return nil, err
	}

	result := make([]ExemptTag, 0)
	err = tags.ForEach(func(tag *plumbing.Reference) error {
		hashSHA1 := tag.Hash().String()

		var hexSHA512 *string = nil
		if useSHA512 {
			var hashSHA512 []byte = nil
			var err error

			t, found := state.TagMap[tag.Hash()]
			if found {
				// annotated tag
				hashSHA512, err = gitHashSHA512.TagSum(t.Hash)
				if err != nil {
					return err
				}
			} else {
				// lightweight tag
				hashSHA512, err = gitHashSHA512.CommitSum(tag.Hash())
				if err != nil {
					return err
				}
			}
			h := hex.EncodeToString(hashSHA512)
			hexSHA512 = &h
		}

		result = append(result, ExemptTag{
			Ref: tag.Name().String(),
			Hash: Digests{
				SHA1:   &hashSHA1,
				SHA512: hexSHA512,
			},
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Ref > result[j].Ref
	})
	return result, nil
}
