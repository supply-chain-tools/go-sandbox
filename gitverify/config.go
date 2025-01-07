package gitverify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/supply-chain-tools/go-sandbox/hashset"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type Config struct {
	Type              string     `json:"_type"`
	Identities        []Identity `json:"identities"`
	Maintainers       []string   `json:"maintainers"`
	Contributors      []string   `json:"contributors"`
	Rules             *Rules     `json:"rules"`
	ProtectedBranches []string   `json:"protectedBranches"`

	ForgeId    *string     `json:"forgeId"`
	ForgeRules *ForgeRules `json:"forgeRules"`

	Repositories []Repository `json:"repositories"`
}

type Identity struct {
	Email            string   `json:"email"`
	AdditionalEmails []string `json:"additionalEmails"`
	GPGPublicKeys    []string `json:"gpgPublicKeys"`
	SSHPublicKeys    []string `json:"sshPublicKeys"`
	ForgeUsername    *string  `json:"forgeUsername"`
	ForgeUserId      *string  `json:"forgeUserId"`
}

type ForgeRules struct {
	AllowMergeCommits   bool `json:"allowMergeCommits"`
	AllowContentCommits bool `json:"allowContentCommits"`
}

type Rules struct {
	AllowSSHSignatures     *bool `json:"allowSshSignatures"`
	RequireSSHUserPresent  *bool `json:"requireSshUserPresent"`
	RequireSSHUserVerified *bool `json:"requireSshUserVerified"`

	AllowGPGSignatures *bool `json:"allowGpgSignatures"`

	RequireSignedTags   *bool `json:"RequireSignedTags"`
	RequireMergeCommits *bool `json:"requireMergeCommits"`
	RequireUpToDate     *bool `json:"requireUpToDate"`
}

type Repository struct {
	Uri   string  `json:"uri"`
	After []After `json:"after"`

	Identities        []Identity `json:"identities"`
	Maintainers       []string   `json:"maintainers"`
	Contributors      []string   `json:"contributors"`
	Rules             *Rules     `json:"rules"`
	ProtectedBranches []string   `json:"protectedBranches"`

	ForgeRules *ForgeRules `json:"forgeRules"`

	ExemptTags []ExemptTag `json:"exemptTags"`
}

type Digests struct {
	SHA1   *string `json:"sha1,omitempty"`
	SHA512 *string `json:"sha512,omitempty"`
}

type After struct {
	SHA1   *string `json:"sha1,omitempty"`
	SHA512 *string `json:"sha512,omitempty"`
	Branch *string `json:"branch,omitempty"`
}

type ParsedConfig struct {
	ForgeId      *string
	Repositories []ParsedRepository
}

type ParsedRepository struct {
	Uri   string
	After []After

	Identities        []Identity
	Maintainers       []string
	Contributors      []string
	Rules             ParsedRules
	ProtectedBranches []string

	ForgeRules   *ForgeRules
	ExemptedTags []ExemptTag
}

type ParsedRules struct {
	AllowSSHSignatures     bool
	RequireSSHUserPresent  bool
	RequireSSHUserVerified bool

	AllowGPGSignatures bool

	RequireSignedTags   bool
	RequireMergeCommits bool
	RequireUpToDate     bool
}

func GetConfigPath(forge string, org string) (string, error) {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDirectory, ".config", "gitverify", forge, org, "gitverify.json"), nil
}

func LoadConfig(configPath string) (*ParsedConfig, error) {
	if configPath == "" {
		return nil, fmt.Errorf("empty config path")
	}

	var p string
	if strings.HasPrefix(configPath, "/") {
		p = configPath
	} else if strings.HasPrefix(configPath, "~") {
		return nil, fmt.Errorf("~ not supported in config file path: %s", configPath)
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		p = filepath.Join(cwd, configPath)
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", p, err)
	}

	config := &Config{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	err = dec.Decode(config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file %s: %w", p, err)
	}

	parsed, err := parseConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", p, err)
	}

	return parsed, nil
}

func parseConfig(config *Config) (*ParsedConfig, error) {
	prefix := "https://supply-chain-tools.github.io/schemas/gitverify/"
	if !strings.HasPrefix(config.Type, prefix) {
		return nil, fmt.Errorf("unsupported schema %s, expect %s", config.Type, prefix)
	}

	version := config.Type[len(prefix):]

	expectedVersion := "v0.1"
	if version != expectedVersion {
		return nil, fmt.Errorf("got schema version %s, expected %s", version, expectedVersion)
	}

	parsedRepos := make([]ParsedRepository, 0)

	for _, repo := range config.Repositories {
		uri, err := validateUri(repo.Uri)
		if err != nil {
			return nil, err
		}

		after, err := validateAfter(repo.After)
		if err != nil {
			return nil, err
		}

		identities, err := combineIdentities(config.Identities, repo.Identities)
		if err != nil {
			return nil, err
		}

		maintainers, err := combineMaintainers(config.Maintainers, repo.Maintainers)
		if err != nil {
			return nil, err
		}

		contributors, err := combineContributors(config.Contributors, repo.Contributors)
		if err != nil {
			return nil, err
		}

		err = ensurePresent(identities, maintainers, contributors)
		if err != nil {
			return nil, err
		}

		rules, err := combineRules(config.Rules, repo.Rules)
		if err != nil {
			return nil, err
		}

		parsedRules := ParsedRules{
			AllowSSHSignatures:     false,
			RequireSSHUserPresent:  true,
			RequireSSHUserVerified: true,
			AllowGPGSignatures:     false,
			RequireSignedTags:      true,
			RequireMergeCommits:    true,
			RequireUpToDate:        true,
		}

		if rules != nil {
			if rules.AllowSSHSignatures != nil {
				parsedRules.AllowSSHSignatures = *rules.AllowSSHSignatures
			}

			if rules.RequireSSHUserPresent != nil {
				parsedRules.RequireSSHUserPresent = *rules.RequireSSHUserPresent
			}

			if rules.RequireSSHUserVerified != nil {
				parsedRules.RequireSSHUserVerified = *rules.RequireSSHUserVerified
			}

			if rules.AllowGPGSignatures != nil {
				parsedRules.AllowGPGSignatures = *rules.AllowGPGSignatures
			}

			if rules.RequireSignedTags != nil {
				parsedRules.RequireSignedTags = *rules.RequireSignedTags
			}

			if rules.RequireMergeCommits != nil {
				parsedRules.RequireMergeCommits = *rules.RequireMergeCommits
			}

			if rules.RequireUpToDate != nil {
				parsedRules.RequireUpToDate = *rules.RequireUpToDate
			}
		}

		forgeRules, err := combineForgeRules(config.ForgeRules, repo.ForgeRules)
		if err != nil {
			return nil, err
		}

		protectedBranches, err := combineProtectedBranches(config.ProtectedBranches, repo.ProtectedBranches)
		if err != nil {
			return nil, err
		}

		parsedRepos = append(parsedRepos, ParsedRepository{
			Uri:               uri,
			After:             after,
			Identities:        identities,
			Maintainers:       maintainers,
			Contributors:      contributors,
			Rules:             parsedRules,
			ProtectedBranches: protectedBranches,
			ForgeRules:        forgeRules,
			ExemptedTags:      repo.ExemptTags,
		})
	}

	parsedConfig := ParsedConfig{
		ForgeId:      config.ForgeId,
		Repositories: parsedRepos,
	}

	return &parsedConfig, nil
}

func ensurePresent(identities []Identity, maintainers []string, contributors []string) error {
	identityEmails := hashset.New[string]()

	for _, identity := range identities {
		identityEmails.Add(identity.Email)
	}

	maintainerEmails := hashset.New[string](maintainers...)
	contributorEmails := hashset.New[string](contributors...)

	maintainerDiff := maintainerEmails.Difference(identityEmails)
	contributorDiff := contributorEmails.Difference(identityEmails)

	if maintainerDiff.Size() > 0 {
		return fmt.Errorf("maintainers '%s' not present in identities", strings.Join(maintainerDiff.Values(), ","))
	}

	if contributorDiff.Size() > 0 {
		return fmt.Errorf("contributors '%s' not present in identities", strings.Join(contributorDiff.Values(), ","))
	}

	return nil
}

func validateUri(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	// https://spdx.github.io/spdx-spec/v2.3/package-information/#77-package-download-location-field
	gitHTTPS := "git+https"
	gitSSH := "git+ssh"
	if !(u.Scheme == gitHTTPS || u.Scheme == gitSSH) {
		return "", fmt.Errorf("got scheme '%s' for repo uri '%s', expected '%s' or '%s'", u.Scheme, uri, gitHTTPS, gitSSH)
	}

	ext := path.Ext(u.Path)
	if ext != ".git" {
		return "", fmt.Errorf("got extension '%s' for repo uri '%s', expected '.git'", ext, uri)
	}

	return uri, nil
}

func validateAfter(after []After) ([]After, error) {
	allBranches := hashset.New[string]()
	allSHA1 := hashset.New[string]()
	allSHA512 := hashset.New[string]()

	for _, a := range after {
		if a.SHA1 == nil && a.SHA512 == nil {
			return nil, fmt.Errorf("either after.sha1 or after.sha512 must be set, or both")
		}

		if a.SHA1 != nil {
			match, err := regexp.MatchString(hexSHA1Regex, *a.SHA1)
			if err != nil {
				return nil, err
			}

			if !match {
				return nil, fmt.Errorf("after.sha1 '%s' must be a 40 character hex", *a.SHA1)
			}

			if allSHA1.Contains(*a.SHA1) {
				return nil, fmt.Errorf("after SHA1 '%s' must be unique", *a.SHA1)
			}
		}

		if a.SHA512 != nil {
			match, err := regexp.MatchString(hexSHA512Regex, *a.SHA512)
			if err != nil {
				return nil, err
			}

			if !match {
				return nil, fmt.Errorf("after.sha512 '%s' must be a 128 character hex", *a.SHA512)
			}

			if allSHA512.Contains(*a.SHA512) {
				return nil, fmt.Errorf("after.sha512 '%s' must be unique", *a.SHA512)
			}
		}

		if a.Branch != nil {
			if allBranches.Contains(*a.Branch) {
				return nil, fmt.Errorf("duplicate branch '%s'", *a.Branch)
			}
			allBranches.Add(*a.Branch)
		}
	}

	return after, nil
}

func combineIdentities(global []Identity, local []Identity) ([]Identity, error) {
	if len(local) != 0 {
		return local, nil
	} else if len(global) != 0 {
		return global, nil
	} else {
		return nil, fmt.Errorf("no identities specified")
	}
}

func combineMaintainers(global []string, local []string) ([]string, error) {
	if len(local) != 0 {
		return local, nil
	} else if len(global) != 0 {
		return global, nil
	} else {
		return nil, fmt.Errorf("no maintainers specified")
	}
}

func combineContributors(global []string, local []string) ([]string, error) {
	if len(local) != 0 {
		return local, nil
	} else {
		return global, nil
	}
}

func combineRules(global *Rules, local *Rules) (*Rules, error) {
	if local != nil {
		return local, nil
	} else if global != nil {
		return global, nil
	} else {
		return nil, fmt.Errorf("no rules specified")
	}
}

func combineProtectedBranches(global []string, local []string) ([]string, error) {
	if len(local) != 0 {
		return local, nil
	} else if len(global) != 0 {
		return global, nil
	} else {
		return nil, nil
	}
}

func combineForgeRules(global *ForgeRules, local *ForgeRules) (*ForgeRules, error) {
	if local != nil {
		return local, nil
	} else {
		return global, nil
	}
}
