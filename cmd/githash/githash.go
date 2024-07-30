package main

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/supply-chain-tools/go-sandbox/githash"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
	"hash"
	"log"
	"log/slog"
	"os"
)

const usage = `Usage:
    githash [options] <commit>

Options:
    -a             Algorithm: sha1, sha256 (default), sha512, sha3-256, sha3-512, blake2b
    -t             Type: commit (default), hash, blob
    --debug        Enable debug logging
    -h, --help     Show help message`

func main() {
	algorithm, objectType, targetHash := processOptionsAndArgs()

	repo, err := loadRepoFromCwd()
	if err != nil {
		log.Fatal(err)
	}

	if targetHash == nil {
		head, err := repo.Head()
		slog.Debug("getting commit from HEAD")
		if err != nil {
			log.Fatal(err)
		}
		h := plumbing.NewHash(head.Hash().String())
		targetHash = &h
	}

	slog.Debug("running githash",
		"targetHash", hex.EncodeToString((*targetHash)[:]))

	verifyTargetHashOrExit(repo, targetHash, objectType)

	gitHash := githash.NewGitHash(repo, algorithm)
	result, err := hashSum(gitHash, targetHash, objectType)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(hex.EncodeToString(result))
}

func processOptionsAndArgs() (algorithm hash.Hash, objectType githash.ObjectType, targetHash *plumbing.Hash) {
	flag.Usage = func() {
		fmt.Println(usage)
	}

	flags := flag.NewFlagSet("all", flag.ExitOnError)
	var help, h, debugMode bool
	var algorithmString, typeString string
	flags.StringVar(&algorithmString, "a", "sha256", "")
	flags.StringVar(&typeString, "t", "commit", "")
	flags.BoolVar(&help, "help", false, "")
	flags.BoolVar(&h, "h", false, "")
	flags.BoolVar(&debugMode, "debug", false, "")

	err := flags.Parse(os.Args[1:])
	if err != nil || help || h {
		fmt.Println(usage)
		os.Exit(1)
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if debugMode {
		opts.Level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	switch algorithmString {
	case "sha1":
		algorithm = sha1.New()
	case "sha256":
		algorithm = sha256.New()
	case "sha512":
		algorithm = sha512.New()
	case "sha3-256":
		algorithm = sha3.New256()
	case "sha3-512":
		algorithm = sha3.New512()
	case "blake2b":
		algorithm, err = blake2b.New(64, nil)
		if err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Printf("Unsupported hash algorithm: %s\n", algorithmString)
		os.Exit(1)
	}

	switch typeString {
	case "commit":
		objectType = githash.CommitObject
	case "tree":
		objectType = githash.TreeObject
	case "blob":
		objectType = githash.BlobObject
	default:
		fmt.Printf("Unsupported type: %s\n", typeString)
		os.Exit(1)
	}

	if len(flags.Args()) > 1 {
		fmt.Printf("Only one argument expected\n")
		os.Exit(1)
	}

	if len(flags.Args()) == 1 {
		hashCandidate := flags.Args()[0]
		if len(hashCandidate) != 40 {
			fmt.Printf("Hash must be 40 characters: got %d\n", len(hashCandidate))
			os.Exit(1)
		}

		h := plumbing.NewHash(hashCandidate)
		targetHash = &h
	}

	return algorithm, objectType, targetHash
}

func loadRepoFromCwd() (*git.Repository, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	repoDir, _, err := gitkit.GetRootPathOfLocalGitRepo(cwd)
	if err != nil {
		return nil, err
	}

	repo, err := gitkit.OpenRepoInLocalPath(repoDir)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func hashSum(gitHash githash.GitHash, targetHash *plumbing.Hash, objectType githash.ObjectType) ([]byte, error) {
	var result []byte
	var err error

	switch objectType {
	case githash.CommitObject:
		result, err = gitHash.CommitSum(*targetHash)
	case githash.TreeObject:
		result, err = gitHash.TreeSum(*targetHash)
	case githash.BlobObject:
		result, err = gitHash.BlobSum(*targetHash)
	}

	return result, err
}

// verifyOrExit verifies that the computed sha1 matches the target hash.
// When testing has been improved this can be removed.
func verifyTargetHashOrExit(repo *git.Repository, targetHash *plumbing.Hash, objectType githash.ObjectType) {
	verificationGitHash := githash.NewGitHash(repo, sha1.New())

	verificationHash, err := hashSum(verificationGitHash, targetHash, objectType)
	if err != nil {
		log.Fatal(err)
	}

	if !bytes.Equal(verificationHash, (*targetHash)[:]) {
		fmt.Printf("Internal error: verification hash %s does not match target hash %s\n", hex.EncodeToString(verificationHash), hex.EncodeToString((*targetHash)[:]))
		os.Exit(1)
	}
}
