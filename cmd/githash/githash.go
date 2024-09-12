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
	"log/slog"
	"os"
)

const usage = `Usage:
    githash [options] <commit>

Options:
    -a, --algorithm    Algorithm: sha1, sha256 (default), sha512, sha3-256, sha3-512, blake2b
    -o, --object-type  Object type: commit (default), hash, blob, tag
    --debug            Enable debug logging
    -h, --help         Show help message`

func main() {
	algorithm, objectType, targetHash, err := processOptionsAndArgs()
	if err != nil {
		print("Failed to process command line options and arguments: ", err.Error(), "\n")
		os.Exit(1)
	}

	repo, err := loadRepoFromCwd()
	if err != nil {
		print("Failed to load repository: ", err.Error(), "\n")
		os.Exit(1)
	}

	if targetHash == nil {
		head, err := repo.Head()
		if err != nil {
			print("Failed to get HEAD commit: ", err.Error(), "\n")
			os.Exit(1)
		}
		slog.Debug("getting commit from HEAD")
		h := plumbing.NewHash(head.Hash().String())
		targetHash = &h
	}

	slog.Debug("running githash",
		"targetHash", hex.EncodeToString((*targetHash)[:]))

	err = verifyTargetHashOrExit(repo, targetHash, objectType)
	if err != nil {
		print("Failed to run integrity check: ", err.Error(), "\n")
		os.Exit(1)
	}

	gitHash := githash.NewGitHash(repo, algorithm)
	result, err := hashSum(gitHash, targetHash, objectType)
	if err != nil {
		print("Failed to get hash sum: ", err.Error(), "\n")
		os.Exit(1)
	}

	fmt.Println(hex.EncodeToString(result))
}

func processOptionsAndArgs() (algorithm hash.Hash, objectType githash.ObjectType, targetHash *plumbing.Hash, err error) {
	flag.Usage = func() {
		fmt.Println(usage)
	}

	flags := flag.NewFlagSet("all", flag.ExitOnError)
	var help, h, debugMode bool
	var algorithmString, algorithmStringShort, typeString, typeStringShort string

	const (
		defaultAlgorithm = "sha256"
		defaultType      = "commit"
	)

	flags.StringVar(&algorithmString, "algorithm", defaultAlgorithm, "")
	flags.StringVar(&algorithmStringShort, "a", defaultAlgorithm, "")
	flags.StringVar(&typeString, "object-type", defaultType, "")
	flags.StringVar(&typeStringShort, "o", defaultType, "")
	flags.BoolVar(&help, "help", false, "")
	flags.BoolVar(&h, "h", false, "")
	flags.BoolVar(&debugMode, "debug", false, "")

	err = flags.Parse(os.Args[1:])
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	if h || help {
		fmt.Println(usage)
		os.Exit(0)
	}

	if algorithmString != defaultAlgorithm && algorithmStringShort != defaultAlgorithm {
		return nil, "", nil, fmt.Errorf("both --algorithm and -a set, pick one")
	}

	if algorithmStringShort != defaultAlgorithm {
		algorithmString = algorithmStringShort
	}

	if typeString != defaultType && typeStringShort != defaultType {
		return nil, "", nil, fmt.Errorf("both --object-type and -o set, pick one")
	}

	if typeStringShort != defaultType {
		typeString = typeStringShort
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
			return nil, "", nil, fmt.Errorf("failed to initialize blake2b: %w", err)
		}
	default:
		return nil, "", nil, fmt.Errorf("unsupported hash algorithm: %s", algorithmString)
	}

	switch typeString {
	case "commit":
		objectType = githash.CommitObject
	case "tree":
		objectType = githash.TreeObject
	case "blob":
		objectType = githash.BlobObject
	case "tag":
		objectType = githash.TagObject
	default:
		return nil, "", nil, fmt.Errorf("unsupported object type: %s", typeString)
	}

	if len(flags.Args()) > 1 {
		return nil, "", nil, fmt.Errorf("only one argument expected, got %d", len(flags.Args()))
	}

	if len(flags.Args()) == 1 {
		hashCandidate := flags.Args()[0]
		if len(hashCandidate) != 40 {
			return nil, "", nil, fmt.Errorf("hash must be 40 characters: got %d\n", len(hashCandidate))
		}

		h := plumbing.NewHash(hashCandidate)
		targetHash = &h
	}

	return algorithm, objectType, targetHash, nil
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
	case githash.TagObject:
		result, err = gitHash.TagSum(*targetHash)
	}

	return result, err
}

// verifyOrExit verifies that the computed sha1 matches the target hash.
// When testing has been improved this can be removed.
func verifyTargetHashOrExit(repo *git.Repository, targetHash *plumbing.Hash, objectType githash.ObjectType) error {
	verificationGitHash := githash.NewGitHash(repo, sha1.New())

	verificationHash, err := hashSum(verificationGitHash, targetHash, objectType)
	if err != nil {
		return err
	}

	if !bytes.Equal(verificationHash, (*targetHash)[:]) {
		return fmt.Errorf("internal error: verification hash %s does not match target hash %s\n", hex.EncodeToString(verificationHash), hex.EncodeToString((*targetHash)[:]))
	}

	return nil
}
