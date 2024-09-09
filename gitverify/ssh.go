package gitverify

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/ssh"
	"strings"
)

type SSHSig struct {
	MagicPreamble [6]byte
	SigVersion    uint32
	PublicKey     string
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Signature     string
}

func validateSSH(content string, signature string, identity identity, config *RepoConfig) error {
	if !config.allowSSHSignatures {
		return fmt.Errorf("SSH signatures not allowed")
	}

	sshSig, err := decodeAndParseSSHSignature(signature)
	if err != nil {
		return err
	}

	trustedKey, found := identity.sshPublicKeys[sshSig.PublicKey]
	if found {
		err = verifySignature(*trustedKey, content, sshSig, namespaceSSH)
		if err != nil {
			return err
		}

		if config.requireSSHUserPresent || config.requireSSHUserVerified {
			publicKey, err := parsePublicKey(sshSig)
			if err != nil {
				return err
			}

			if !(publicKey.KeyType == "sk-ssh-ed25519@openssh.com" || publicKey.KeyType == "sk-ecdsa-sha2-nistp256@openssh.com") {
				return fmt.Errorf("unsupported public key type %s for user present/verified", publicKey.KeyType)
			}

			signature, err := parseU2FSignature(sshSig)
			if err != nil {
				return err
			}

			if config.requireSSHUserPresent && !signature.userPresent() {
				return fmt.Errorf("user present missing")
			}

			if config.requireSSHUserVerified && !signature.userVerified() {
				return fmt.Errorf("user verified missing")
			}
		}
	} else {
		return fmt.Errorf("matching SSH key not found for '%s'", identity.email)
	}

	return nil
}

func verifySSHSignature(key string, signature string, data string, namespace string) error {
	publicKey, err := decodeAndParseSSHPublicKey(key)
	if err != nil {
		return err
	}

	sshSig, err := decodeAndParseSSHSignature(signature)
	if err != nil {
		return err
	}

	err = verifySignature(publicKey, data, sshSig, namespace)
	if err != nil {
		return err
	}

	return nil
}

func decodeAndParseSSHPublicKey(key string) (ssh.PublicKey, error) {
	parts := strings.Split(key, " ")
	rawKey, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %v", err)
	}

	publicKey, err := ssh.ParsePublicKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return publicKey, nil
}

func decodeAndParseSSHSignature(signature string) (*SSHSig, error) {
	rawSignature, err := unwrapSshSignature(signature)
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap signature: %v", err)
	}

	rawSig, err := base64.StdEncoding.DecodeString(rawSignature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %v", err)
	}

	sshSig := &SSHSig{}
	err = ssh.Unmarshal(rawSig, sshSig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal signature: %w", err)
	}

	return sshSig, nil
}

func verifySignature(maintainerAllowedKey ssh.PublicKey, message string, signature *SSHSig, namespace string) error {
	var h []byte

	switch signature.HashAlgorithm {
	case "sha256":
		r := sha256.Sum256([]byte(message))
		h = r[:]
	case "sha512":
		r := sha512.Sum512([]byte(message))
		h = r[:]
	default:
		return fmt.Errorf("unsupported hash algorithm: %s", signature.HashAlgorithm)
	}

	sshSig := SshSig{
		Namespace:     namespace,
		Reserved:      "",
		HashAlgorithm: signature.HashAlgorithm,
		Hash:          string(h),
	}

	signedBlob := ssh.Marshal(sshSig)
	signedBlob = append([]byte("SSHSIG"), signedBlob...)

	sig := &ssh.Signature{}
	err := ssh.Unmarshal([]byte(signature.Signature), sig)
	if err != nil {
		return err
	}

	err = maintainerAllowedKey.Verify(signedBlob, sig)
	if err != nil {
		return err
	}

	return nil
}

func unwrapSshSignature(signature string) (string, error) {
	header := "-----BEGIN SSH SIGNATURE-----\n"
	footer := "-----END SSH SIGNATURE-----"

	signature = strings.Trim(signature, "\n ")

	if !strings.HasPrefix(signature, header) {
		return "", fmt.Errorf("signature does not start with header")
	}

	if !strings.HasSuffix(signature, footer) {
		return "", fmt.Errorf("signature does not end with footer")
	}

	subset := signature[len(header) : len(signature)-len(footer)-1]
	result := strings.ReplaceAll(subset, "\n", "")
	return result, nil
}

type SSHPublicKey struct {
	KeyType string
	Key     string
	Scope   string
}

func parsePublicKey(sshSig *SSHSig) (*SSHPublicKey, error) {
	publicKey := &SSHPublicKey{}

	err := ssh.Unmarshal([]byte(sshSig.PublicKey), publicKey)
	if err != nil {
		return nil, err
	}

	return publicKey, nil
}

type U2FSignature struct {
	// https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.u2f
	Type      string
	Signature []byte
	Flags     byte
	Counter   uint32
}

func parseU2FSignature(sshSig *SSHSig) (*U2FSignature, error) {
	signature := &U2FSignature{}

	err := ssh.Unmarshal([]byte(sshSig.Signature), signature)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

func (u *U2FSignature) userPresent() bool {
	return (u.Flags & 1) != 0
}

func (u *U2FSignature) userVerified() bool {
	return (u.Flags >> 2 & 1) != 0
}

type SshSig struct {
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Hash          string
}
