package gitrelease

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/supply-chain-tools/go-sandbox/githash"
	"github.com/supply-chain-tools/go-sandbox/gitkit"
	"github.com/supply-chain-tools/go-sandbox/gohash"
	"strconv"
	"strings"
	"time"
)

const (
	intotoType    = "https://in-toto.io/Statement/v1"
	predicateType = "https://in-toto.io/attestation/link/v0.3"
	payloadType   = "application/vnd.in-toto+json"
)

// TagLink https://github.com/in-toto/attestation/blob/main/spec/predicates/link.md
type TagLink struct {
	// Required
	Type string `json:"_type"`

	// Required
	Subject []Subject `json:"subject"`

	// Required
	PredicateType string `json:"predicateType"`

	// Required
	Predicate LinkPredicate `json:"predicate"`
}

// Subject https://github.com/in-toto/attestation/blob/main/spec/v1/statement.md
type Subject struct {
	// Optional, should be unique within subject
	Name string `json:"name,omitempty"`

	// https://github.com/in-toto/attestation/blob/main/spec/v1/field_types.md#resourceuri
	// Optional, should be unique within subject
	Uri string `json:"uri,omitempty"`

	// https://github.com/in-toto/attestation/blob/main/spec/v1/digest_set.md
	// Required for subject
	Digest map[string]string `json:"digest"`
}

// LinkPredicate https://github.com/in-toto/attestation/blob/main/spec/predicates/link.md
type LinkPredicate struct {
	// Required
	Name string `json:"name"`

	// Optional
	Command []string `json:"command"`

	// Optional
	Materials []LinkMaterial `json:"materials"`

	// Optional
	Environment map[string]string `json:"environment"`
}

// LinkMaterial https://github.com/in-toto/attestation/blob/main/spec/v1/resource_descriptor.md
type LinkMaterial struct {
	// Required for link, must be unique among materials
	Name string `json:"name"`

	// Optional
	Uri string `json:"uri,omitempty"`

	// https://github.com/in-toto/attestation/blob/main/spec/v1/digest_set.md
	// Required for link
	Digest map[string]string `json:"digest"`
}

type Envelope struct {
	// https://github.com/in-toto/attestation/blob/main/spec/v1/envelope.md
	// https://github.com/secure-systems-lab/dsse/blob/v1.0.0/envelope.md
	PayLoadType string      `json:"payloadType"`
	Payload     string      `json:"payload"`
	Signatures  []Signature `json:"signatures"`
}

type Signature struct {
	KeyID string `json:"keyid,omitempty"`
	Sig   string `json:"sig"`
	Cert  string `json:"cert,omitempty"`
}

type TagMetadata struct {
	Type              string            `json:"_type"`
	RepositoryUri     string            `json:"repositoryUri"`
	CommitHash        CommitHash        `json:"commitHash"`
	PreviousTag       *PreviousTag      `json:"previousTag,omitempty"`
	ProtectedBranches []ProtectedBranch `json:"protectedBranches"`
	GoRelease         *GoRelease        `json:"goRelease,omitempty"`
}

type CommitHash struct {
	SHA1   *string `json:"sha1,omitempty"`
	SHA256 *string `json:"sha256,omitempty"`
}

type PreviousTag struct {
	SHA1   *string `json:"sha1,omitempty"`
	SHA256 *string `json:"sha256,omitempty"`
}

type ProtectedBranch struct {
	Name   string  `json:"name"`
	SHA1   *string `json:"sha1,omitempty"`
	SHA256 *string `json:"sha256,omitempty"`
}

type GoRelease struct {
	Uri           string `json:"uri"`
	DirectoryHash string `json:"directoryHash"`
	GoModHash     string `json:"goModHash"`
}

func CreateTagLink(repo *git.Repository, commitHash plumbing.Hash, tag *object.Tag, repoUrl string, command []string, tagMetadata *TagMetadata) (*TagLink, error) {
	gh := githash.NewGitHash(repo, sha256.New())
	tagSHA256, err := gh.TagSum(tag.Hash)
	if err != nil {
		return nil, err
	}

	timestamp := time.Now()
	timestampString := timestamp.Format(time.RFC3339)

	subjects := []Subject{
		{
			Name: "tag",
			Uri:  wrapGitURLTagOrCommit(repoUrl, tag.Name),
			Digest: map[string]string{
				"gitTag":        tag.Hash.String(),
				"gitTag_sha256": hex.EncodeToString(tagSHA256),
			},
		},
	}

	materials := []LinkMaterial{
		{
			Name: "commit",
			Uri:  wrapGitURLTagOrCommit(repoUrl, commitHash.String()),
			Digest: map[string]string{
				"gitCommit":        commitHash.String(),
				"gitCommit_sha256": *tagMetadata.CommitHash.SHA256,
			},
		},
		{
			Name: "refs/remotes/origin/main",
			Uri:  wrapGitURLTagOrCommit(repoUrl, "main"),
			Digest: map[string]string{
				"gitCommit":        *tagMetadata.ProtectedBranches[0].SHA1,
				"gitCommit_sha256": *tagMetadata.ProtectedBranches[0].SHA256,
			},
		},
		// TODO include all protected or all refs/remotes/origin/*, not just main
	}

	if tagMetadata.GoRelease != nil {
		h, err := h1InToToDirectoryHash(tagMetadata.GoRelease.DirectoryHash)
		if err != nil {
			return nil, err
		}

		materials = append(materials, LinkMaterial{
			// https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst#golang
			Name: "goRelease",
			Uri:  tagMetadata.GoRelease.Uri,
			Digest: map[string]string{
				"dirHash1": h,
			},
		})

		h, err = h1InToToDirectoryHash(tagMetadata.GoRelease.GoModHash)
		if err != nil {
			return nil, err
		}

		materials = append(materials, LinkMaterial{
			Name: "goRelease#go.mod",
			Uri:  tagMetadata.GoRelease.Uri + "#go.mod",
			Digest: map[string]string{
				"dirHash1": h,
			},
		})
	}

	return &TagLink{
		Type:          intotoType,
		Subject:       subjects,
		PredicateType: predicateType,
		Predicate: LinkPredicate{
			Name:      "tag",
			Command:   command,
			Materials: materials,
			Environment: map[string]string{
				"timestamp": timestampString,
			},
		},
	}, nil
}

func wrapGitURLTagOrCommit(url string, tagOrCommit string) string {
	// https://spdx.github.io/spdx-spec/v2.3/package-information/#77-package-download-location-field
	return "git+" + url + ".git@" + tagOrCommit
}

func wrapGitURL(url string) string {
	// https://spdx.github.io/spdx-spec/v2.3/package-information/#77-package-download-location-field
	return "git+" + url + ".git"
}

// https://github.com/in-toto/attestation/blob/main/spec/v1/digest_set.md#supported-algorithms
func h1InToToDirectoryHash(h1 string) (string, error) {
	prefix := "h1:"
	if strings.HasPrefix(h1, prefix) {
		data, err := base64.StdEncoding.DecodeString(h1[len(prefix):])
		if err != nil {
			return "", fmt.Errorf("invalid h1: %s", h1)
		}

		return hex.EncodeToString(data), nil
	}

	return "", fmt.Errorf("invalid h1: %s", h1)
}

func CreateEnvelope(payload string, signature string) *Envelope {
	return &Envelope{
		PayLoadType: payloadType,
		Payload:     payload,
		Signatures: []Signature{
			{
				Sig: signature,
			},
		},
	}
}

func PreAuthenticationEncoding(payload []byte) ([]byte, error) {
	// https://github.com/secure-systems-lab/dsse/blob/v1.0.0/protocol.md
	// "DSSEv1" + SP + LEN(type) + SP + type + SP + LEN(payload) + SP + payload
	if len(payload) < 1 {
		return nil, fmt.Errorf("signature payload must be at least 1 byte long")
	}

	sp := " "
	sb := strings.Builder{}
	sb.WriteString("DSSEv1")
	sb.WriteString(sp)
	sb.WriteString(strconv.Itoa(len(payloadType)))
	sb.WriteString(sp)
	sb.WriteString(payloadType)
	sb.WriteString(sp)
	sb.WriteString(strconv.Itoa(len(payload)))
	sb.WriteString(sp)
	sb.Write(payload)

	return []byte(sb.String()), nil
}

func CreatePreviousTagPayload(repo *git.Repository, tagName string, commitHash plumbing.Hash, previousTag *object.Tag, repoUrl string) (*TagMetadata, error) {
	gh := githash.NewGitHash(repo, sha256.New())
	h, err := gh.CommitSum(commitHash)
	if err != nil {
		return nil, err
	}

	commitSHA1 := commitHash.String()
	commitSHA256 := hex.EncodeToString(h)

	remoteMain := plumbing.NewRemoteReferenceName("origin", "main")
	remoteRef, err := repo.Reference(remoteMain, false)
	if err != nil {
		return nil, fmt.Errorf("failed ot find origin/main: %w", err)
	}

	remoteMainHash := remoteRef.Hash().String()
	h, err = gh.CommitSum(commitHash)
	if err != nil {
		return nil, err
	}
	remoteMainSHA256 := hex.EncodeToString(h)

	goHashes, err := gohash.GitDirHashAllVersion(repo, commitHash, &tagName)
	if err != nil {
		return nil, err
	}

	var goRelease *GoRelease = nil
	for _, goHash := range goHashes {
		if goHash.Directory == "" {
			goRelease = &GoRelease{
				// https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst#golang
				Uri:           "pkg:golang/" + goHash.Path + "@" + goHash.Version,
				DirectoryHash: goHash.DirectoryHash,
				GoModHash:     goHash.GoModHash,
			}
		}
	}

	tagMetadata := TagMetadata{
		Type:          "https://supply-chain-tools.github.io/schemas/tag-metadata/v0.0",
		RepositoryUri: wrapGitURL(repoUrl),
		CommitHash: CommitHash{
			SHA1:   &commitSHA1,
			SHA256: &commitSHA256,
		},
		PreviousTag: nil,
		ProtectedBranches: []ProtectedBranch{
			{
				Name:   "main",
				SHA1:   &remoteMainHash,
				SHA256: &remoteMainSHA256,
			},
		},
		GoRelease: goRelease,
	}

	if previousTag != nil {
		gh := githash.NewGitHash(repo, sha256.New())
		tagSHA256, err := gh.TagSum(previousTag.Hash)
		if err != nil {
			return nil, err
		}

		previousTagSHA1 := previousTag.Hash.String()
		previousTagSHA256 := hex.EncodeToString(tagSHA256)
		tagMetadata.PreviousTag = &PreviousTag{
			SHA1:   &previousTagSHA1,
			SHA256: &previousTagSHA256,
		}
	}

	return &tagMetadata, nil
}

func FindPreviousTag(repo *git.Repository) (*object.Tag, bool, error) {
	tags, err := repo.Tags()
	if err != nil {
		return nil, false, err
	}

	state := gitkit.LoadRepoState(repo)
	var previousTag *object.Tag

	tagHashes := make(map[string]*object.Tag)
	previousTagHashes := make(map[string]*TagMetadata)

	err = tags.ForEach(func(tag *plumbing.Reference) error {
		t, isAnnotatedTag := state.TagMap[tag.Hash()]

		if isAnnotatedTag {
			tagMetadata, err := decodeTagMetadata(t)
			if err != nil {
				return err
			}

			tagHashes[tag.Hash().String()] = t

			if tagMetadata.PreviousTag != nil {
				// TODO also verify SHA-256
				targetHash := tagMetadata.PreviousTag.SHA1
				_, found := previousTagHashes[*targetHash]
				if found {
					return fmt.Errorf("duplicate previous tag '%s'", *targetHash)
				}

				previousTagHashes[*targetHash] = tagMetadata
			}
		} else {
			return fmt.Errorf("lightweight tags are not supported %s", tag.Hash().String())
		}

		return nil
	})
	if err != nil {
		return nil, false, err
	}

	count := 0
	for hash, tag := range tagHashes {
		if _, found := previousTagHashes[hash]; !found {
			previousTag = tag
			count++
		}
	}

	if count == 0 {
		return nil, false, nil
	}

	if count != 1 {
		return nil, false, fmt.Errorf("expected exactly one tag not pointed to by another tag, got %d", count)
	}

	return previousTag, previousTag != nil, nil
}

func decodeTagMetadata(tag *object.Tag) (*TagMetadata, error) {
	// TODO \r?
	lines := strings.Split(strings.TrimSuffix(tag.Message, "\n"), "\n")

	prefix := "Tag-metadata: "

	var payload string

	count := 0
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			payload = strings.TrimPrefix(line, prefix)
			count++
		}
	}

	if count == 0 {
		return nil, fmt.Errorf("no line in the tag '%s' starts with '%s'", tag.Name, prefix)
	}

	if count > 1 {
		return nil, fmt.Errorf("more than one line in the tag '%s' starts with '%s'", tag.Name, prefix)
	}

	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}

	tagMetadata := &TagMetadata{}
	err = json.Unmarshal(data, tagMetadata)
	if err != nil {
		return nil, err
	}

	return tagMetadata, nil
}
