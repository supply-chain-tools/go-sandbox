package gitverify

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/supply-chain-tools/go-sandbox/githash"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"regexp"
	"strings"
)

type ValidateOptions struct {
	Commit       string
	Tag          string
	Branch       string
	VerifyOnHEAD bool
	VerifyOnTip  bool
}

const hexSHA1Regex = "^[a-f0-9]{40}$"
const hexSHA256Regex = "^[a-f0-9]{64}$"

func Verify(repo *git.Repository, state *gitkit.RepoState, repoConfig *RepoConfig, gitHashSHA1 githash.GitHash, gitHashSHA256 githash.GitHash, opts *ValidateOptions) error {
	commitMetadata, err := computeCommitMetadata(state, repoConfig, gitHashSHA1, gitHashSHA256)
	if err != nil {
		return err
	}

	if opts != nil && opts.Commit != "" {
		matched, err := regexp.MatchString(hexSHA1Regex, opts.Commit)
		if err != nil {
			return err
		}

		if !matched {
			return fmt.Errorf("target commit must be a 40 character hex, not '%s'", opts.Commit)
		}

		err = validateOpts(opts, repo, state, commitMetadata, repoConfig, gitHashSHA1, gitHashSHA256)
		if err != nil {
			return err
		}
	} else {
		for _, commit := range state.CommitMap {
			err := validateCommit(commit, commitMetadata, repoConfig)
			if err != nil {
				return err
			}
		}

		err = validateTags(repo, state, repoConfig, gitHashSHA1, gitHashSHA256)
		if err != nil {
			return err
		}

		err = validateProtectedBranches(repo, state, commitMetadata, repoConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateCommit(commit *object.Commit, commitMetadata map[plumbing.Hash]*CommitData, repoConfig *RepoConfig) error {
	metadata, found := commitMetadata[commit.Hash]
	if !found {
		return fmt.Errorf("commit not processed: %s", commit.Hash)
	}

	if metadata.Ignore {
		return nil
	}

	email := commit.Committer.Email

	if repoConfig.forge != nil {
		if repoConfig.forge.email == email {
			err := validateGPGCommit(commit, repoConfig.forge.gpgPublicKey)
			if err != nil {
				return err
			}

			if !repoConfig.forge.allowMergeCommits && !repoConfig.forge.allowContentCommits {
				return fmt.Errorf("forge is not allowed to make commits: %s", commit.Hash.String())
			}

			_, found := repoConfig.maintainerOrContributorEmails[commit.Author.Email]
			if !found {
				_, found := repoConfig.maintainerOrContributorForgeEmails[commit.Author.Email]
				if !found {
					return fmt.Errorf("author email '%s' not found for forge commit: %s", commit.Author.Email, commit.Hash.String())
				}
			}

			if !repoConfig.forge.allowMergeCommits && len(commit.ParentHashes) > 1 {
				return fmt.Errorf("up to one parent hash supported for forge: %s", commit.Hash.String())
			}

			if repoConfig.forge.allowMergeCommits && !repoConfig.forge.allowContentCommits {
				err := verifyMergeCommitNoContentChanges(commit)
				if err != nil {
					return fmt.Errorf("failed to verify forge merge commit %s to not have content changes: %s", commit.Hash.String(), err)
				}

				metadata.VerifiedToNotHaveContentChanges = true
			}

			return nil
		}
	}

	id, found := repoConfig.maintainerOrContributorEmails[email]
	if !found {
		return fmt.Errorf("no maintainer with email '%s' for commit %s", email, commit.Hash)
	}

	switch metadata.SignatureType {
	case SignatureTypeSSH:
		content := buildContent(commit)
		err := validateSSH(content, commit.PGPSignature, id, repoConfig)
		if err != nil {
			return fmt.Errorf("failed to validate commit %s: %w", commit.Hash.String(), err)
		}
	case SignatureTypeGPG:
		err := validateIdentityGPGCommit(commit, id, repoConfig)
		if err != nil {
			return err
		}
	case SignatureTypeNone:
		return fmt.Errorf("unsigned commit: %s", commit.Hash.String())
	default:
		return fmt.Errorf("unknown signature type for commit: %s", commit.Hash.String())
	}

	return nil
}

func validateOpts(opts *ValidateOptions, repo *git.Repository, state *gitkit.RepoState, commitMetadata map[plumbing.Hash]*CommitData, config *RepoConfig, gitHashSHA1 githash.GitHash, gitHashSHA256 githash.GitHash) error {
	head, err := repo.Head()
	if err != nil {
		return err
	}

	headHash := head.Hash()

	c, found := state.CommitMap[plumbing.NewHash(opts.Commit)]
	if !found {
		return fmt.Errorf("target commit '%s' not found", opts.Commit)
	}

	err = validateCommit(c, commitMetadata, config)
	if err != nil {
		return err
	}

	targetHash := c.Hash

	if opts.VerifyOnHEAD {
		if c.Hash != headHash {
			return fmt.Errorf("HEAD does not point to the target commit %s", opts.Commit)
		}
	}

	// Verify that all commits properly signed
	visited := hashset.New[plumbing.Hash]()
	visited.Add(c.Hash)
	queue := []*object.Commit{c}

	for {
		if len(queue) == 0 {
			break
		}

		current := queue[0]
		queue = queue[1:]

		for _, parentHash := range current.ParentHashes {
			if !visited.Contains(parentHash) {
				parent, found := state.CommitMap[parentHash]
				if !found {
					return fmt.Errorf("target parent hash not found: %s", parentHash)
				}

				if !commitMetadata[parent.Hash].Ignore {
					err = validateCommit(parent, commitMetadata, config)
					if err != nil {
						return err
					}

					queue = append(queue, parent)
					visited.Add(parentHash)
				}
			}
		}
	}

	// Verify that commit is connected to after (otherwise the commits might not be in the right repository)
	// The commit can be connected to after by being a descendant of an ignored commit or by being an ignored commit
	if !commitMetadata[c.Hash].Ignore {
		queue = []*object.Commit{c}

		for {
			if len(queue) == 0 {
				return fmt.Errorf("commit '%s' not connected to after", targetHash.String())
			}

			current := queue[0]
			queue = queue[1:]

			if len(current.ParentHashes) > 0 {
				parentHash := current.ParentHashes[0]
				if commitMetadata[parentHash].Ignore {
					break
				}

				parent, found := state.CommitMap[parentHash]
				if !found {
					return fmt.Errorf("target parent hash not found: %s", parentHash)
				}
				queue = append(queue, parent)
			}
		}
	}

	var tagHash *plumbing.Hash = nil
	if opts.Tag != "" {
		tags, err := repo.Tags()
		if err != nil {
			return err
		}

		tagFound := false
		err = tags.ForEach(func(tag *plumbing.Reference) error {
			entry := strings.TrimPrefix(tag.Name().String(), "refs/tags/")

			if entry == opts.Tag {
				err := validateTag(tag, state, config, gitHashSHA1, gitHashSHA256)
				if err != nil {
					return err
				}

				tagFound = true
				if tagFound {
					t, found := state.TagMap[tag.Hash()]
					if found {
						// annotated tag
						tagHash = &t.Target

						if opts.VerifyOnHEAD {
							if t.Target != headHash {
								return fmt.Errorf("HEAD does not point to the same commit %s as target tag '%s'", t.Target.String(), opts.Tag)
							}
						}
					} else {
						// lightweight tag
						t := tag.Hash()
						tagHash = &t

						if opts.VerifyOnHEAD {
							if tag.Hash() != headHash {
								return fmt.Errorf("HEAD does not point to the same commit %s as target tag '%s'", tag.Hash().String(), opts.Tag)
							}
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		if !tagFound {
			return fmt.Errorf("target tag '%s' not found", opts.Tag)
		}
	}

	if tagHash != nil {
		if targetHash != *tagHash {
			return fmt.Errorf("target tag '%s' does not point to target commit '%s' ", opts.Tag, opts.Commit)
		}
	}

	if opts.Branch != "" {
		remotes, err := repo.References()
		if err != nil {
			return err
		}

		branchFound := false
		err = remotes.ForEach(func(reference *plumbing.Reference) error {
			prefix := "refs/remotes/origin/"
			if strings.HasPrefix(reference.Name().String(), prefix) {
				branchName := reference.Name().String()[len(prefix):]
				if branchName == opts.Branch {
					branchFound = true

					c, found := state.CommitMap[reference.Hash()]
					if !found {
						return fmt.Errorf("commit '%s' not found", reference.Hash().String())
					}

					err = validateCommit(c, commitMetadata, config)
					if err != nil {
						return err
					}

					if opts.VerifyOnTip {
						if targetHash != c.Hash {
							return fmt.Errorf("target commit %s does not point to the tip of branch '%s'", targetHash.String(), opts.Branch)
						}
					} else {
						// Verify that targetHash is on the branch
						queue := []*object.Commit{c}

						for {
							if len(queue) == 0 {
								return fmt.Errorf("target commit %s is not on target branch '%s'", opts.Commit, opts.Branch)
							}

							current := queue[0]
							queue = queue[1:]

							if current.Hash == targetHash {
								break
							}

							if len(current.ParentHashes) > 0 {
								parentHash := current.ParentHashes[0]
								if !visited.Contains(parentHash) {
									parent, found := state.CommitMap[parentHash]
									if !found {
										return fmt.Errorf("target parent hash not found: %s", parentHash)
									}

									err = validateCommit(parent, commitMetadata, config)
									if err != nil {
										return err
									}

									queue = append(queue, parent)
								}
							}
						}
					}

				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		if !branchFound {
			return fmt.Errorf("target branch '%s' not found", opts.Branch)
		}
	}

	return nil
}

func validateProtectedBranches(repo *git.Repository, state *gitkit.RepoState, commitMetadata map[plumbing.Hash]*CommitData, config *RepoConfig) error {
	remotes, err := repo.References()
	if err != nil {
		return err
	}

	err = remotes.ForEach(func(reference *plumbing.Reference) error {
		isProtected, branchName := isProtected(reference, config)

		if isProtected {
			targetAfter, found := config.branchToSHA1[branchName]
			if !found {
				return fmt.Errorf("protected branch '%s' without matching after branch", branchName)
			}

			current, found := state.CommitMap[reference.Hash()]
			if !found {
				return fmt.Errorf("did not find commit %s", reference.Hash().String())
			}

			for {
				if current.Hash == targetAfter {
					break
				}

				if config.requireMergeCommits {
					if len(current.ParentHashes) != 2 {
						return fmt.Errorf("requireMergeCommits is set, but commit %s on protected branch has %d parents", current.Hash.String(), len(current.ParentHashes))
					}

				}

				if len(current.ParentHashes) == 2 {
					_, found := config.maintainerEmails[current.Committer.Email]
					if !found {
						if config.forge != nil && current.Committer.Email == config.forge.email {
							_, found = config.maintainerEmails[current.Author.Email]
							if !found {
								_, found = config.maintainerForgeEmails[current.Author.Email]
							}
						}

						if !found {
							return fmt.Errorf("merge commit %s made by %s which is not a maintainer", current.Hash.String(), current.Committer.Email)
						}
					}

					metadata := commitMetadata[current.Hash]
					if !metadata.VerifiedToNotHaveContentChanges {
						err := verifyMergeCommitNoContentChanges(current)
						if err != nil {
							return fmt.Errorf("failed to verify protected merge commit %s to not have content changes: %s", current.Hash.String(), err)
						}

						metadata.VerifiedToNotHaveContentChanges = true
					}

					if config.requireUpToDate {
						mergeBase, err := gitMergeBase(current.ParentHashes[0].String(), current.ParentHashes[1].String())
						if err != nil {
							return fmt.Errorf("failed to find merge base for parent commits of %s: %w", current.Hash.String(), err)
						}

						if mergeBase != current.ParentHashes[0].String() {
							return fmt.Errorf("second parent of %s is not up to date with first", current.Hash.String())
						}
					}
				}

				if len(current.ParentHashes) == 0 {
					return fmt.Errorf("protected branch %s is not a decendant of after", reference.Name().String())
				}

				current, found = state.CommitMap[current.ParentHashes[0]]
				if !found {
					return fmt.Errorf("did not find commit %s", reference.Hash().String())
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func validateTags(repo *git.Repository, state *gitkit.RepoState, repoConfig *RepoConfig, gitHashSHA1 githash.GitHash, gitHashSHA256 githash.GitHash) error {
	tags, err := repo.Tags()
	if err != nil {
		return err
	}

	err = tags.ForEach(func(tag *plumbing.Reference) error {
		return validateTag(tag, state, repoConfig, gitHashSHA1, gitHashSHA256)
	})
	if err != nil {
		return err
	}

	return nil
}

func validateTag(tag *plumbing.Reference, state *gitkit.RepoState, repoConfig *RepoConfig, gitHashSHA1 githash.GitHash, gitHashSHA256 githash.GitHash) error {
	isExempted := false

	tagHash, found := repoConfig.exemptedTags[tag.Name().String()]
	if found {
		if tagHash != tag.Hash().String() {
			return fmt.Errorf("wrong hash.sha1 for exempted tag '%s', got %s, expected %s", tag.Name().String(), tag.Hash().String(), tagHash)
		}
		isExempted = true
	}

	t, isAnnotatedTag := state.TagMap[tag.Hash()]

	tagHashSHA256, found := repoConfig.exemptedTagsSHA256[tag.Name().String()]
	if found {
		var sha256Hash []byte
		var err error
		if isAnnotatedTag {
			sha256Hash, err = gitHashSHA256.TagSum(t.Hash)
			if err != nil {
				return err
			}
		} else {
			sha256Hash, err = gitHashSHA256.CommitSum(tag.Hash())
			if err != nil {
				return err
			}
		}

		h := hex.EncodeToString(sha256Hash)
		if tagHashSHA256 != h {
			return fmt.Errorf("wrong hash.sha256 for exempted tag '%s', got %s, expected %s", tag.Name().String(), h, tagHashSHA256)
		}
		isExempted = true
	}

	entry := strings.TrimPrefix(tag.Name().String(), "refs/tags/")
	if isAnnotatedTag {
		if entry != t.Name {
			return fmt.Errorf("tag ref '%s' does not match name '%s'", entry, t.Name)
		}

		if !isExempted {
			signatureType, err := inferSignatureType(t.PGPSignature)
			if err != nil {
				return err
			}

			id, found := repoConfig.maintainerEmails[t.Tagger.Email]
			if !found {
				return fmt.Errorf("no maintainer with email '%s' for tag %s", t.Tagger.Email, t.Name)
			}

			switch signatureType {
			case SignatureTypeSSH:
				content, err := tagContent(t)
				if err != nil {
					return err
				}
				err = validateSSH(content, t.PGPSignature, id, repoConfig)
				if err != nil {
					return fmt.Errorf("failed to validate tag %s: %w", t.Name, err)
				}
			case SignatureTypeGPG:
				err := validateIdentityGPGTag(t, id, repoConfig)
				if err != nil {
					return err
				}
			case SignatureTypeNone:
				if !repoConfig.requireSignedTags {
					return fmt.Errorf("unsigned annotated tag: %s", t.Name)
				}
			default:
				return fmt.Errorf("unknown signature type for tag: %s", t.Name)
			}
		}
	} else {
		if !isExempted {
			if repoConfig.requireSignedTags {
				return fmt.Errorf("tag '%s' is lightweight, but signing is required", tag.Name())
			}
		}
	}

	return nil
}

func tagContent(tag *object.Tag) (string, error) {
	sb := strings.Builder{}

	sb.WriteString("object " + tag.Target.String() + "\n")
	sb.WriteString("type commit\n")
	sb.WriteString("tag " + tag.Name + "\n")
	sb.WriteString(fmt.Sprintf("tagger %s <%s> %d %s\n", tag.Tagger.Name, tag.Tagger.Email, tag.Tagger.When.Unix(), tag.Tagger.When.Format("-0700")))

	sb.WriteString("\n")
	sb.WriteString(tag.Message)

	return sb.String(), nil
}

func computeCommitMetadata(state *gitkit.RepoState, repoConfig *RepoConfig, gitHashSHA1 githash.GitHash, gitHashSHA256 githash.GitHash) (map[plumbing.Hash]*CommitData, error) {
	commitMap := make(map[plumbing.Hash]*CommitData)

	foundAfterSHA1 := hashset.New[plumbing.Hash]()
	foundAfterSHA256 := hashset.New[[32]byte]()

	for hash, commit := range state.CommitMap {
		if len(commit.ParentHashes) > 2 {
			return nil, fmt.Errorf("up to two parents are allowed, commit '%s' has %d", hash.String(), len(commit.ParentHashes))
		}

		verifiedSHA1, err := gitHashSHA1.CommitSum(hash)
		if err != nil {
			return nil, err
		}

		if !bytes.Equal(verifiedSHA1, hash[:]) {
			return nil, fmt.Errorf("failed to verify hash %s", hash)
		}

		var verifiedSHA256 [32]byte
		var sha256WasVerified = false
		if repoConfig.afterSHA256.Size() > 0 {
			v, err := gitHashSHA256.CommitSum(hash)
			if err != nil {
				return nil, err
			}

			if len(v) != 32 {
				return nil, fmt.Errorf("expected hash to be 32, got %d", len(verifiedSHA256))
			}

			copy(verifiedSHA256[:], v[:32])
			sha256WasVerified = true
		}

		matchedAfterSHA256 := false
		if sha256WasVerified {
			if repoConfig.afterSHA256.Contains(verifiedSHA256) {
				matchedAfterSHA256 = true
				foundAfterSHA256.Add(verifiedSHA256)
			}
		}

		matchedAfterSHA1 := false
		if repoConfig.afterSHA1.Size() > 0 {
			if repoConfig.afterSHA1.Contains(hash) {
				matchedAfterSHA1 = true
				foundAfterSHA1.Add(hash)
			}
		}

		matchedAfter := false

		_, found := repoConfig.afterSHA1ToSHA256[hash]
		if found {
			// Both SHA-1 and SHA-256 specified, check that they are the same
			if matchedAfterSHA1 != matchedAfterSHA256 {
				return nil, fmt.Errorf("matched after SHA-1 or SHA-256 but not both")
			}

			matchedAfter = matchedAfterSHA1
		} else {
			// Otherwise it's enough that one matched
			matchedAfter = matchedAfterSHA1 || matchedAfterSHA256
		}

		if matchedAfter {
			if !repoConfig.afterSHA1.Contains(hash) {
				// This was matched using SHA-256, add it to SHA-1 as well
				repoConfig.afterSHA1.Add(hash)

				// Use branches from SHA-256
				branch := repoConfig.sha256ToBranch[verifiedSHA256]
				repoConfig.sha1ToBranch[hash] = branch
				repoConfig.branchToSHA1[branch] = hash
			}
		}

		_, found = commitMap[hash]
		if found {
			continue
		}

		if matchedAfter {
			err := ignoreCommitAndParents(commit, commitMap, state)
			if err != nil {
				return nil, err
			}
		} else {
			signatureType, err := inferSignatureType(commit.PGPSignature)
			if err != nil {
				return nil, err
			}

			commitMap[hash] = &CommitData{
				SignatureType: signatureType,
			}
		}
	}

	afterSHA1Diff := repoConfig.afterSHA1.Difference(foundAfterSHA1)
	if afterSHA1Diff.Size() > 0 {
		missingHashes := make([]string, 0)
		for _, k := range afterSHA1Diff.Values() {
			missingHashes = append(missingHashes, k.String())
		}
		return nil, fmt.Errorf("after SHA-1 commit(s) not found in repo: %s", strings.Join(missingHashes, ","))
	}

	afterSHA256Diff := repoConfig.afterSHA256.Difference(foundAfterSHA256)
	if afterSHA256Diff.Size() > 0 {
		missingHashes := make([]string, 0)
		for _, k := range afterSHA256Diff.Values() {
			missingHashes = append(missingHashes, hex.EncodeToString(k[:]))
		}
		return nil, fmt.Errorf("after SHA-256 commit(s) not found in repo: %s", strings.Join(missingHashes, ","))
	}

	return commitMap, nil
}

func buildContent(commit *object.Commit) string {
	sb := strings.Builder{}
	sb.WriteString("tree " + commit.TreeHash.String() + "\n")

	for _, parent := range commit.ParentHashes {
		sb.WriteString("parent " + parent.String() + "\n")
	}

	// TODO verify for UTC
	sb.WriteString(fmt.Sprintf("author %s <%s> %d %s\n", commit.Author.Name, commit.Author.Email, commit.Author.When.Unix(), commit.Author.When.Format("-0700")))
	sb.WriteString(fmt.Sprintf("committer %s <%s> %d %s\n", commit.Committer.Name, commit.Committer.Email, commit.Committer.When.Unix(), commit.Committer.When.Format("-0700")))
	sb.WriteString("\n")
	sb.WriteString(commit.Message)
	return sb.String()
}

func isProtected(reference *plumbing.Reference, config *RepoConfig) (bool, string) {
	isProtected := false
	var branchName string
	if strings.HasPrefix(reference.Name().String(), "refs/remotes/") {
		parts := strings.Split(reference.Name().Short(), "/")
		branchName = strings.Join(parts[1:], "/")
		if config.protectedBranches.Contains(branchName) {
			isProtected = true
		}
	} else if strings.HasPrefix(reference.Name().String(), "refs/heads/") {
		branchName = reference.Name().Short()
		if config.protectedBranches.Contains(branchName) {
			isProtected = true
		}
	}

	return isProtected, branchName
}

func BranchName(ref string) (string, bool) {
	found := false
	var branchName string

	remotesPrefix := "refs/remotes/"
	headsPrefix := "refs/heads/"
	if strings.HasPrefix(ref, remotesPrefix) {
		suffix := strings.TrimPrefix(ref, remotesPrefix)
		parts := strings.Split(suffix, "/")
		branchName = strings.Join(parts[1:], "/")
		found = true
	} else if strings.HasPrefix(ref, headsPrefix) {
		branchName = strings.TrimPrefix(ref, headsPrefix)
		found = true
	}

	return branchName, found
}
