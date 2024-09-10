package gitverify

import (
	"encoding/json"
	"testing"
)

func TestConfig(t *testing.T) {
	config := `
{
  "_type": "https://supply-chain-tools.github.io/schemas/gitverify/v0.1",
  "identities": [
    {
      "email": "a@example.internal",
      "sshPublicKeys": ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIAQv90+kSOSKZYlMoWO0eX6QZ1Nt5n2BviA4vFx3lgK"]
    },
    {
      "email": "c@example.internal",
      "sshPublicKeys": ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIAQv90+kSOSKZYlMoWO0eX6QZ1Nt5n2BviA4vFx3lgK"]
    }
  ],
  "maintainers": ["a@example.internal"],
  "contributors": ["c@example.internal"],
  "rules": {
    "allowSSHSignatures": true,
    "requireSSHUserPresent": true,
    "requireSSHUserVerified": true
  },
  "forgeId": "github.com",
  "repositories": [
    {
      "uri": "git+https://github.com/foo/bar.git",
      "after": [{
          "SHA1": "0000000000000000000000000000000000000000"
      }]
    },
    {
      "uri": "git+ssh://github.com/foo/baz.git",
      "after": [{
          "SHA1": "ffffffffffffffffffffffffffffffffffffffff"
      }],
      "identities": [
        {
          "email": "b@example.internal",
          "sshPublicKeys": ["ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBH2r8kV3iq50ugWjL3l4OaLEhGNUhMPc/A2UWQSix/I5XEG6sfnXZre06ROUF2DaWxiACUiLhO1UDUY0guun3ZQ="],
          "additionalEmails" : ["b2@example.internal"],
          "forgeUsername" : "b",
          "forgeUserId" : "1234"
        },
        {
          "email": "d@example.internal",
          "sshPublicKeys": ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIAQv90+kSOSKZYlMoWO0eX6QZ1Nt5n2BviA4vFx3lgK"]
        }
      ],
      "maintainers": ["b@example.internal"],
      "contributors": ["d@example.internal"],
      "rules": {
        "allowSSHSignatures": true,
        "requireSSHUserPresent": false,
        "requireSSHUserVerified": false
      },
      "forgeRules": {
        "allowMergeCommits": true,
        "allowContentCommits": false
      }
    }
  ]
}
`
	runnerConfig := &Config{}
	err := json.Unmarshal([]byte(config), runnerConfig)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := parseConfig(runnerConfig)
	if err != nil {
		t.Fatal(err)
	}

	if *parsed.ForgeId != "github.com" {
		t.Errorf("ForgeId=%q, want %q", *parsed.ForgeId, "github.com")
	}

	repo0 := parsed.Repositories[0]
	if repo0.Uri != "git+https://github.com/foo/bar.git" {
		t.Errorf("repo0.Uri=%q, want %q", repo0.Uri, "git+https://github.com/foo/bar.git")
	}

	if *repo0.After[0].SHA1 != "0000000000000000000000000000000000000000" {
		t.Errorf("repo0.Since[0].SHA1=%v, want %q", *repo0.After[0].SHA1, "0000000000000000000000000000000000000000")
	}

	if repo0.Identities[0].Email != "a@example.internal" {
		t.Errorf("repo0.Identities[0].Email=%q, want %q", repo0.Identities[0].Email, "a@example.internal")
	}

	if repo0.Identities[0].AdditionalEmails != nil {
		t.Errorf("repo0.Identities[0].AdditionalEmails=%q, want nil", repo0.Identities[0].AdditionalEmails)
	}

	if repo0.Identities[0].SSHPublicKeys[0] != "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIAQv90+kSOSKZYlMoWO0eX6QZ1Nt5n2BviA4vFx3lgK" {
		t.Errorf("repo0.Identities[0].SSHPublicKeys[0]=%q, want %q", repo0.Identities[0].SSHPublicKeys[0], "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIAQv90+kSOSKZYlMoWO0eX6QZ1Nt5n2BviA4vFx3lgK")
	}

	if repo0.Maintainers[0] != "a@example.internal" {
		t.Errorf("repo0.Maintainers[0]=%q, want %q", repo0.Maintainers[0], "a@example.internal")
	}

	if repo0.Contributors[0] != "c@example.internal" {
		t.Errorf("repo0.Contributors[0]=%q, want %q", repo0.Contributors[0], "c@example.internal")
	}

	if repo0.Rules.AllowSSHSignatures != true {
		t.Errorf("repo0.Rules.AllowSSHSignature=%t, want %t", repo0.Rules.AllowSSHSignatures, true)
	}

	if repo0.Rules.RequireSSHUserPresent != true {
		t.Errorf("repo0.Rules.RequireSSHUserPresent=%t, want %t", repo0.Rules.RequireSSHUserPresent, true)
	}

	if repo0.Rules.RequireSSHUserVerified != true {
		t.Errorf("repo0.Rules.RequireSSHUserVerified=%t, want %t", repo0.Rules.RequireSSHUserVerified, true)
	}

	if repo0.Rules.AllowGPGSignatures != false {
		t.Errorf("repo0.Rules.AllowGPGSignatures=%t, want %t", repo0.Rules.AllowGPGSignatures, false)
	}

	if repo0.ForgeRules != nil {
		t.Errorf("repo0.ForgeRules not nil")
	}

	repo1 := parsed.Repositories[1]
	if repo1.Uri != "git+ssh://github.com/foo/baz.git" {
		t.Errorf("repo1.Uri=%q, want %q", repo1.Uri, "git+ssh://github.com/foo/baz.git")
	}

	if *repo1.After[0].SHA1 != "ffffffffffffffffffffffffffffffffffffffff" {
		t.Errorf("repo1.Since[0].SHA1=%v, want %q", *repo1.After[0].SHA1, "ffffffffffffffffffffffffffffffffffffffff")
	}

	if repo1.Identities[0].Email != "b@example.internal" {
		t.Errorf("repo1.Identities[0].Email=%q, want %q", repo1.Identities[0].Email, "a@example.internal")
	}

	if repo1.Identities[0].AdditionalEmails[0] != "b2@example.internal" {
		t.Errorf("repo1.Identities[0].AdditionalEmails[0]=%q, want %q", repo1.Identities[0].AdditionalEmails[0], "b2@example.internal")
	}

	if repo1.Identities[0].SSHPublicKeys[0] != "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBH2r8kV3iq50ugWjL3l4OaLEhGNUhMPc/A2UWQSix/I5XEG6sfnXZre06ROUF2DaWxiACUiLhO1UDUY0guun3ZQ=" {
		t.Errorf("repo1.Identities[0].SSHPublicKeys[0]=%q, want %q", repo1.Identities[0].SSHPublicKeys[0], "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBH2r8kV3iq50ugWjL3l4OaLEhGNUhMPc/A2UWQSix/I5XEG6sfnXZre06ROUF2DaWxiACUiLhO1UDUY0guun3ZQ=")
	}

	if *repo1.Identities[0].ForgeUsername != "b" {
		t.Errorf("repo1.Identities[0].ForgeUsername=%q, want %q", *repo1.Identities[0].ForgeUsername, "b")
	}

	if *repo1.Identities[0].ForgeUserId != "1234" {
		t.Errorf("repo1.Identities[0].ForgeUserId=%q, want %q", *repo1.Identities[0].ForgeUserId, "1234")
	}

	if repo1.Maintainers[0] != "b@example.internal" {
		t.Errorf("repo1.Maintainers[0]=%q, want %q", repo1.Maintainers[0], "b@example.internal")
	}

	if repo1.Contributors[0] != "d@example.internal" {
		t.Errorf("repo1.Contributors[0]=%q, want %q", repo1.Contributors[0], "d@example.internal")
	}

	if repo1.Rules.AllowSSHSignatures != true {
		t.Errorf("repo1.Rules.AllowSSHSignature=%t, want %t", repo1.Rules.AllowSSHSignatures, true)
	}

	if repo1.Rules.RequireSSHUserPresent != false {
		t.Errorf("repo1.Rules.RequireSSHUserPresent=%t, want %t", repo1.Rules.RequireSSHUserPresent, false)
	}

	if repo1.Rules.RequireSSHUserVerified != false {
		t.Errorf("repo1.Rules.RequireSSHUserVerified=%t, want %t", repo1.Rules.RequireSSHUserVerified, false)
	}

	if repo1.Rules.AllowGPGSignatures != false {
		t.Errorf("repo1.Rules.AllowGPGSignatures=%t, want %t", repo1.Rules.AllowGPGSignatures, false)
	}

	if repo1.ForgeRules.AllowMergeCommits != true {
		t.Errorf("repo1.ForgeRules.AllowMergeCommits=%t, want %t", repo1.ForgeRules.AllowMergeCommits, true)
	}

	if repo1.ForgeRules.AllowContentCommits != false {
		t.Errorf("repo1.ForgeRules.AllowContentCommits=%t, want %t", repo1.ForgeRules.AllowContentCommits, false)
	}
}
