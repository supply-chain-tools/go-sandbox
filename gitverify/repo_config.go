package gitverify

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"golang.org/x/crypto/ssh"
	"regexp"
	"strings"
)

type RepoConfig struct {
	afterSHA1                          hashset.Set[plumbing.Hash]
	afterSHA256                        hashset.Set[[32]byte]
	sha1ToBranch                       map[plumbing.Hash]string
	branchToSHA1                       map[string]plumbing.Hash
	sha256ToBranch                     map[[32]byte]string
	afterSHA1ToSHA256                  map[plumbing.Hash][32]byte
	maintainerEmails                   map[string]identity
	contributorEmails                  map[string]identity
	maintainerOrContributorEmails      map[string]identity
	maintainerForgeEmails              map[string]identity
	contributorForgeEmails             map[string]identity
	maintainerOrContributorForgeEmails map[string]identity
	forge                              *forge
	allowSSHSignatures                 bool
	requireSSHUserPresent              bool
	requireSSHUserVerified             bool
	allowGPGSignatures                 bool
	requireSignedTags                  bool
	requireMergeCommits                bool
	requireUpToDate                    bool
	protectedBranches                  hashset.Set[string]
	exemptedTags                       map[string]string
	exemptedTagsSHA256                 map[string]string
}

type identity struct {
	email         string
	forgeUsername *string
	forgeUserId   *string
	sshPublicKeys map[string]*ssh.PublicKey
	gpgPublicKeys []string
}

type forge struct {
	email               string
	gpgPublicKey        string
	allowMergeCommits   bool
	allowContentCommits bool
}

func LoadRepoConfig(config *ParsedConfig, repoUri string) (*RepoConfig, error) {
	var repo *ParsedRepository = nil
	for _, r := range config.Repositories {
		if r.Uri == repoUri {
			repo = &r
		}
	}

	if repo == nil {
		return nil, fmt.Errorf("repository %s not found in config", repoUri)
	}

	if len(repo.Maintainers) == 0 {
		return nil, fmt.Errorf("no maintainers specified: %s", repoUri)
	}
	maintainerSet := hashset.New[string](repo.Maintainers...)
	contributorSet := hashset.New[string](repo.Contributors...)

	for _, m := range repo.Maintainers {
		if contributorSet.Contains(m) {
			return nil, fmt.Errorf("'%s' must be maintainer or contributor not both", m)
		}
	}

	allEmails := hashset.New[string]()
	maintainerEmails := make(map[string]identity)
	contributorEmails := make(map[string]identity)
	maintainerOrContributor := make(map[string]identity)

	allForgeEmails := hashset.New[string]()
	maintainerForgeEmails := make(map[string]identity)
	contributorForgeEmails := make(map[string]identity)
	maintainerOrContributorForgeEmails := make(map[string]identity)

	for _, i := range repo.Identities {
		sshPublicKeys := make(map[string]*ssh.PublicKey)
		for _, sshPublicKey := range i.SSHPublicKeys {
			parts := strings.Split(sshPublicKey, " ")
			rawKey, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				return nil, err
			}

			publicKey, err := ssh.ParsePublicKey(rawKey)
			if err != nil {
				return nil, err
			}

			// TODO check for duplicates
			sshPublicKeys[string(rawKey)] = &publicKey
		}

		identityEntry := identity{
			email:         i.Email,
			forgeUsername: i.ForgeUsername,
			forgeUserId:   i.ForgeUserId,
			sshPublicKeys: sshPublicKeys,
			gpgPublicKeys: i.GPGPublicKeys,
		}

		var forgeEmail = ""
		if config.ForgeId != nil && *config.ForgeId == gitHubForgeId && i.ForgeUsername != nil && i.ForgeUserId != nil {
			forgeEmail = gitHubUserEmail(*i.ForgeUserId, *i.ForgeUsername)

			if allForgeEmails.Contains(forgeEmail) {
				return nil, fmt.Errorf("duplicate forge email '%s' in repository %s", forgeEmail, repoUri)
			}
			allForgeEmails.Add(forgeEmail)
		}

		if allEmails.Contains(i.Email) {
			return nil, fmt.Errorf("duplicate email %s found in repository %s", i.Email, repoUri)
		}
		allEmails.Add(i.Email)

		if maintainerSet.Contains(i.Email) || contributorSet.Contains(i.Email) {
			maintainerOrContributor[i.Email] = identityEntry

			if forgeEmail != "" {
				maintainerOrContributorForgeEmails[forgeEmail] = identityEntry
			}
		}

		if maintainerSet.Contains(i.Email) {
			maintainerEmails[i.Email] = identityEntry

			if forgeEmail != "" {
				maintainerForgeEmails[forgeEmail] = identityEntry
			}
		}

		if contributorSet.Contains(i.Email) {
			contributorEmails[i.Email] = identityEntry

			if forgeEmail != "" {
				contributorForgeEmails[forgeEmail] = identityEntry
			}
		}

		for _, additionalEmail := range i.AdditionalEmails {
			if allEmails.Contains(additionalEmail) {
				return nil, fmt.Errorf("duplicate email '%s' found in repository '%s'", additionalEmail, repoUri)
			}

			allEmails.Add(additionalEmail)
			if maintainerSet.Contains(i.Email) {
				maintainerEmails[additionalEmail] = identityEntry
				maintainerOrContributor[i.Email] = identityEntry
			}

			if contributorSet.Contains(additionalEmail) {
				contributorEmails[i.Email] = identityEntry
				maintainerOrContributor[i.Email] = identityEntry
			}
		}
	}

	var f *forge
	if config.ForgeId != nil {
		if *config.ForgeId == gitHubForgeId {
			f = &forge{
				email:               gitHubEmail,
				gpgPublicKey:        gitHubKey,
				allowMergeCommits:   repo.ForgeRules.AllowMergeCommits,
				allowContentCommits: repo.ForgeRules.AllowContentCommits,
			}
		} else {
			return nil, fmt.Errorf("unsupported forge: %s", *config.ForgeId)
		}
	}

	exemptedTagMap := make(map[string]string)
	exemptedTagSHA256Map := make(map[string]string)
	for _, exemptTag := range repo.ExemptedTags {
		_, found := exemptedTagMap[exemptTag.Ref]
		if found {
			return nil, fmt.Errorf("duplicate extempted tag %s found in repository %s", exemptTag.Ref, repoUri)
		}

		_, found = exemptedTagSHA256Map[exemptTag.Ref]
		if found {
			return nil, fmt.Errorf("duplicate extempted SHA256 tag %s found in repository %s", exemptTag.Ref, repoUri)
		}

		if exemptTag.Hash.SHA1 == nil && exemptTag.Hash.SHA256 == nil {
			return nil, fmt.Errorf("at least one of hash.sha1 and hash.sha256 must be set for exempted tag %s", exemptTag.Ref)
		}

		if exemptTag.Hash.SHA1 != nil {
			match, err := regexp.MatchString(hexSHA1Regex, *exemptTag.Hash.SHA1)
			if err != nil {
				return nil, err
			}

			if !match {
				return nil, fmt.Errorf("SHA-1 hash for exempted tag must be 40 character hex, got %s", *exemptTag.Hash.SHA1)
			}
			exemptedTagMap[exemptTag.Ref] = *exemptTag.Hash.SHA1
		}

		if exemptTag.Hash.SHA256 != nil {
			match, err := regexp.MatchString(hexSHA256Regex, *exemptTag.Hash.SHA256)
			if err != nil {
				return nil, err
			}

			if !match {
				return nil, fmt.Errorf("hash.sha256 for exempted tag must be 64 character hex, got %s", *exemptTag.Hash.SHA256)
			}

			exemptedTagSHA256Map[exemptTag.Ref] = *exemptTag.Hash.SHA256
		}
	}

	protectedBranches := hashset.New[string](repo.ProtectedBranches...)

	var afterSHA1 = hashset.New[plumbing.Hash]()
	var afterSHA256 = hashset.New[[32]byte]()
	afterSHA1ToSHA256 := make(map[plumbing.Hash][32]byte)

	sha1ToBranch := make(map[plumbing.Hash]string)
	branchToSHA1 := make(map[string]plumbing.Hash)

	sha256ToBranch := make(map[[32]byte]string)

	for _, after := range repo.After {
		var sha1 plumbing.Hash
		if after.SHA1 != nil {
			sha1 = plumbing.NewHash(*after.SHA1)
			afterSHA1.Add(sha1)

			if after.Branch != nil {
				sha1ToBranch[sha1] = *after.Branch
				branchToSHA1[*after.Branch] = sha1
			}
		}

		var sha256 [32]byte
		if after.SHA256 != nil {
			h, err := hex.DecodeString(*after.SHA256)
			if err != nil {
				return nil, err
			}

			if len(h) != 32 {
				return nil, fmt.Errorf("SHA256 hash should be 32 bytes, got %d", len(h))
			}

			copy(sha256[:], h[:32])

			afterSHA256.Add(sha256)

			if after.Branch != nil {
				sha256ToBranch[sha256] = *after.Branch
			}
		}

		if after.SHA1 != nil && after.SHA256 != nil {
			afterSHA1ToSHA256[sha1] = sha256
		}
	}

	return &RepoConfig{
		afterSHA1:                          afterSHA1,
		afterSHA256:                        afterSHA256,
		sha1ToBranch:                       sha1ToBranch,
		branchToSHA1:                       branchToSHA1,
		sha256ToBranch:                     sha256ToBranch,
		afterSHA1ToSHA256:                  afterSHA1ToSHA256,
		maintainerEmails:                   maintainerEmails,
		contributorEmails:                  contributorEmails,
		maintainerOrContributorEmails:      maintainerOrContributor,
		maintainerForgeEmails:              maintainerForgeEmails,
		contributorForgeEmails:             contributorForgeEmails,
		maintainerOrContributorForgeEmails: maintainerOrContributorForgeEmails,
		forge:                              f,
		allowSSHSignatures:                 repo.Rules.AllowSSHSignatures,
		requireSSHUserPresent:              repo.Rules.RequireSSHUserPresent,
		requireSSHUserVerified:             repo.Rules.RequireSSHUserVerified,
		allowGPGSignatures:                 repo.Rules.AllowGPGSignatures,
		requireSignedTags:                  repo.Rules.RequireSignedTags,
		requireMergeCommits:                repo.Rules.RequireMergeCommits,
		requireUpToDate:                    repo.Rules.RequireUpToDate,
		exemptedTags:                       exemptedTagMap,
		exemptedTagsSHA256:                 exemptedTagSHA256Map,
		protectedBranches:                  protectedBranches,
	}, nil
}
