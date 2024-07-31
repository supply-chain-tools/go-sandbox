package githash

import (
	"encoding/hex"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"hash"
	"io"
	"strings"
)

type ObjectType string

var (
	CommitObject ObjectType = "commit"
	TreeObject   ObjectType = "tree"
	BlobObject   ObjectType = "blob"
)

type GitHash interface {
	CommitSum(commitHash plumbing.Hash) ([]byte, error)
	TreeSum(treeHash plumbing.Hash) ([]byte, error)
	BlobSum(blobHash plumbing.Hash) ([]byte, error)
}

type gitHash struct {
	repo      *git.Repository
	repoState *gitkit.RepoState
	hash      hash.Hash
	commitMap map[plumbing.Hash][]byte
	treeMap   map[plumbing.Hash][]byte
	blobMap   map[plumbing.Hash][]byte
}

func NewGitHash(repo *git.Repository, hash hash.Hash) GitHash {
	repoState := gitkit.LoadRepoState(repo)
	return &gitHash{
		repo:      repo,
		repoState: repoState,
		hash:      hash,
		commitMap: make(map[plumbing.Hash][]byte),
		treeMap:   make(map[plumbing.Hash][]byte),
		blobMap:   make(map[plumbing.Hash][]byte),
	}
}

func (gh *gitHash) CommitSum(commitHash plumbing.Hash) ([]byte, error) {
	h, found := gh.commitMap[commitHash]
	if found {
		return h, nil
	}

	commit, found := gh.repoState.CommitMap[commitHash]
	if !found {
		return nil, fmt.Errorf("commit %s not found", commitHash)
	}

	content, err := gh.commitContent(commit, true)
	if err != nil {
		return nil, err
	}

	h = objectHash([]byte(content), CommitObject, gh.hash)
	gh.commitMap[commitHash] = h

	return h, nil
}

func (gh *gitHash) commitContent(commit *object.Commit, includeSignature bool) (string, error) {
	sb := strings.Builder{}

	treeHash, found := gh.treeMap[commit.TreeHash]
	if !found {
		h, err := gh.TreeSum(commit.TreeHash)
		if err != nil {
			return "", err
		}
		treeHash = h
	}

	sb.WriteString("tree " + hex.EncodeToString(treeHash) + "\n")

	for _, parent := range commit.ParentHashes {
		parentHash, found := gh.commitMap[parent]
		if !found {
			h, err := gh.CommitSum(parent)
			if err != nil {
				return "", err
			}
			parentHash = h
		}

		sb.WriteString("parent " + hex.EncodeToString(parentHash) + "\n")
	}

	sb.WriteString(fmt.Sprintf("author %s <%s> %d %s\n", commit.Author.Name, commit.Author.Email, commit.Author.When.Unix(), commit.Author.When.Format("-0700")))
	sb.WriteString(fmt.Sprintf("committer %s <%s> %d %s\n", commit.Committer.Name, commit.Committer.Email, commit.Committer.When.Unix(), commit.Committer.When.Format("-0700")))

	if includeSignature {
		sb.WriteString(gpgSigString(commit))
	}

	sb.WriteString(commit.Message)

	result := sb.String()

	return result, nil
}

func gpgSigString(commit *object.Commit) string {
	sb := strings.Builder{}
	sb.WriteString("gpgsig")

	parts := strings.Split(commit.PGPSignature, "\n")

	for i := 0; i < len(parts)-1; i++ {
		sb.WriteString(" ")
		sb.WriteString(parts[i])
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	s := sb.String()

	return s
}

func (gh *gitHash) TreeSum(treeHash plumbing.Hash) ([]byte, error) {
	h, found := gh.treeMap[treeHash]
	if found {
		return h, nil
	}

	tree, found := gh.repoState.TreeMap[treeHash]
	if !found {
		return nil, fmt.Errorf("tree %s not found", treeHash)
	}

	entries := tree.Entries

	data := make([]byte, 0)
	for _, entry := range entries {
		var entryHash []byte
		var err error

		if entry.Mode == filemode.Dir {
			entryHash, err = gh.TreeSum(entry.Hash)
			if err != nil {
				return nil, err
			}
		} else if entry.Mode == filemode.Regular || entry.Mode == filemode.Executable || entry.Mode == filemode.Symlink {
			entryHash, err = gh.BlobSum(entry.Hash)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("entry mode %s not supported", entry.Mode)
		}

		m := removeLeadingZeros([]byte(entry.Mode.String()))
		data = append(data, m...)

		data = append(data, 32) // space
		data = append(data, []byte(entry.Name)...)
		data = append(data, 0) // null

		data = append(data, entryHash...)

	}
	h = objectHash(data, TreeObject, gh.hash)

	gh.treeMap[treeHash] = h

	return h, nil
}

func (gh *gitHash) BlobSum(treeHash plumbing.Hash) ([]byte, error) {
	h, found := gh.blobMap[treeHash]
	if found {
		return h, nil
	}

	blob, found := gh.repoState.BlobMap[treeHash]
	if !found {
		return nil, fmt.Errorf("blob %s not found", treeHash)
	}

	reader, err := blob.Reader()
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	h = objectHash(data, BlobObject, gh.hash)
	gh.blobMap[treeHash] = h

	return h, nil
}

func removeLeadingZeros(data []byte) []byte {
	for i := 0; i < len(data); i++ {
		if data[i] != '0' {
			return data[i:]
		}
	}

	return data
}

func objectHash(data []byte, objectType ObjectType, hash hash.Hash) []byte {
	header := fmt.Sprintf("%s %d\x00", objectType, len(data))
	data = append([]byte(header), data...)

	hash.Reset()
	hash.Write(data)
	return hash.Sum(nil)
}
