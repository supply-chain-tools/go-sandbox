package gitverify

import (
	"fmt"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/supply-chain-tools/go-sandbox/hashset"
)

func validateIdentityGPGCommit(commit *object.Commit, id identity, config *RepoConfig) error {
	if !config.allowGPGSignatures {
		return fmt.Errorf("GPG signatures not allowed: %s", commit.Hash.String())
	}

	if len(id.gpgPublicKeys) < 1 {
		return fmt.Errorf("GPG public key not found for commit %s", commit.Hash.String())
	}

	if len(id.gpgPublicKeys) > 1 {
		return fmt.Errorf("only one GPG key is currently supported got %d", len(id.gpgPublicKeys))
	}

	return validateGPGCommit(commit, id.gpgPublicKeys[0])
}

func validateGPGCommit(commit *object.Commit, key string) error {
	entity, err := commit.Verify(key)
	if err != nil {
		return fmt.Errorf("failed to verify commit %s: %w", commit.Hash.String(), err)
	}

	entityEmails := hashset.New[string]()
	for _, identity := range entity.Identities {
		entityEmails.Add(identity.UserId.Email)
	}

	if !entityEmails.Contains(commit.Committer.Email) {
		return fmt.Errorf("GPG key does not match committer email '%s' for commit %s", commit.Committer.Email, commit.Hash.String())
	}

	return nil
}

func validateIdentityGPGTag(tag *object.Tag, id identity, config *RepoConfig) error {
	if !config.allowGPGSignatures {
		return fmt.Errorf("GPG signatures not allowed: %s", tag.Name)
	}

	if len(id.gpgPublicKeys) < 1 {
		return fmt.Errorf("GPG public key not found for commit %s", tag.Name)
	}

	if len(id.gpgPublicKeys) > 1 {
		return fmt.Errorf("only one GPG key is currently supported got %d", len(id.gpgPublicKeys))
	}

	return validateGPGTag(tag, id.gpgPublicKeys[0])
}

func validateGPGTag(tag *object.Tag, key string) error {
	entity, err := tag.Verify(key)
	if err != nil {
		return fmt.Errorf("failed to verify tag %s: %w", tag.Hash.String(), err)
	}

	entityEmails := hashset.New[string]()
	for _, identity := range entity.Identities {
		entityEmails.Add(identity.UserId.Email)
	}

	if !entityEmails.Contains(tag.Tagger.Email) {
		return fmt.Errorf("GPG key does not match tagger email '%s' for tag %s", tag.Tagger.Email, tag.Hash.String())
	}

	return nil
}
