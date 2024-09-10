# Config file

### All config options
```json
{
  "_type": "https://supply-chain-tools.github.io/schemas/gitverify/v0.1",
  "identities": [
    {
      "email": "stian.kristoffersen@telenor.no",
      "forgeUsername" : "stiankri-telenor",
      "forgeUserId": "155450741",
      "sshPublicKeys": [
        "sk-ssh-ed25519@openssh.com AAAAGnNrLXNzaC1lZDI1NTE5QG9wZW5zc2guY29tAAAAIHa4MOkvaZbhdeWuYUFQ1sywWYkpW9x9uVTX+RlDdMvXAAAABHNzaDo="
      ]
    },
    {
      "email": "33384479+dev-bio@users.noreply.github.com",
      "forgeUsername" : "dev-bio",
      "forgeUserId": "33384479",
      "sshPublicKeys": [
        "sk-ssh-ed25519@openssh.com AAAAGnNrLXNzaC1lZDI1NTE5QG9wZW5zc2guY29tAAAAIDTCGpjJM/to9icZbLRiyYzz1UoPDTSbqhwLRotpWd4sAAAABHNzaDo="
      ]
    }
  ],
  "maintainers": [
    "stian.kristoffersen@telenor.no"
  ],
  "contributors": [
    "33384479+dev-bio@users.noreply.github.com"
  ],
  "rules": {
    "allowSshSignatures": false,
    "requireSshUserPresent": true,
    "requireSshUserVerified": true,
    "allowGpgSignatures": false,
    "requireSignedTags": true,
    "requireMergeCommits": true
  },
  "protectedBranches": ["main"],
  "forgeId": "github.com",
  "forgeRules": {
    "allowMergeCommits": false,
    "allowContentCommits": false
  },
  "repositories": [
    {
      "uri": " git+https://github.com/supply-chain-tools/go-sandbox.git",
      "after": [{
          "sha1": "1f46f2053221c040ce5bcba0239bc09214a37658",
          "branch": "main"
        }],
      "exemptTags": [{"ref":"refs/tags/0.0.1","hash":{"sha1":"1f46f2053221c040ce5bcba0239bc09214a37658"}}]
    }
  ]
}
```

### Identities
| Config                      | Value                   | Required | Description                                                                                                                       |
|-----------------------------|-------------------------|----------|-----------------------------------------------------------------------------------------------------------------------------------|
| `identities`                | list of `identity`      | yes      |                                                                                                                                   |
| `identity.email`            | email                   | yes      | Must be unique for a `repository`                                                                                                 |
| `identity.sshPublicKeys`    | list of SSH public keys | no       | Must be unique for a `repository`, same format as in SSH public files without the comment                                         |
| `identity.gpgPublicKeys`    | list of GPG public keys | no       | Must be unique for a `repository`, only one GPG key is currently supported, standard armored string with newlines encoded as `\n` |
| `identity.forgeUsername`    | string                  | no       | E.g. GitHub login name                                                                                                            |
| `identity.forgeUserId`      | string                  | no       | E.g. GitHub user id                                                                                                               |
| `identity.additionalEmails` | list of emails          | no       | If more than one email should be associated with this identity                                                                    |

### Maintainers and Contributors
Maintainers are allowed to sign any commit or tag. Contributors are not allowed to sign tags. Merge commits into
`protectedBranches` will be verified to be from maintainers, not contributors.
If `forge.allowMergeCommit` or `forge.allowContentCommit` is set, then the author is verified to match `maintainers`
or `contributors` following the same rules as if they made the commit themselves.

The difference between `maintainers` and `contributors` might change in the future. The main goal is to allow for outside
contributions without a maintainer committing the change.

| Config        | Value              | Required | Description                        |
|---------------|--------------------|----------|------------------------------------|
| `maintainers` | list of emails     | yes      | Must reference an `identity.email` |

| Config         | Value              | Required | Description                        |
|----------------|--------------------|----------|------------------------------------|
| `contributors` | list of emails     | no       | Must reference an `identity.email` |

### Rules
| Config                         | Value                       | Required | Description                                                                                                                                                                                                        |
|--------------------------------|-----------------------------|----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `rules`                        | object                      | no       |                                                                                                                                                                                                                    |
| `rules.allowSshSignatures`     | `true`, `false` (default)   | no       | `maintainers` and `contributors` are allowed to use SSH signatures                                                                                                                                                 |
| `rules.requireSshUserPresent`  | `true` (default), `false`   | no       | `maintainers` and `contributors` are required to touch security key when signing. Only `sk-ssh-ed25519@openssh.com` and `sk-ecdsa-sha2-nistp256@openssh.com` is supported and will fail on other key types.        |
| `rules.requireSshUserVerified` | `true` (default), `false`   | no       | `maintainers` and `contributors` are required to use PIN with security key when signing. Only `sk-ssh-ed25519@openssh.com` and `sk-ecdsa-sha2-nistp256@openssh.com` is supported and will fail on other key types. |
| `rules.allowGpgSignatures`     | `true`, `false` (default)   | no       | `maintainers` and `contributors` are allowed to use GPG signatures                                                                                                                                                 |
| `rules.requireSignedTags`      | `true` (default), `false`   | no       | Allow unsigned tags, `repository.exemptTags` is an alternative                                                                                                                                                     |
| `rules.requireMergeCommits`    | `true` (default), `false`   | no       | Require protected branches to use merge commits. Any conflicts must be resolved before merging.                                                                                                                    |
| `rules.requireUpToDate`        | `true` (default), `false`   | no       | For merges commits into protected branches, require the other branch to be up to date with the protected branch before merging.                                                                                    |


### Forge
| Config                           | Value                     | Required | Description                                                                                    |
|----------------------------------|---------------------------|----------|------------------------------------------------------------------------------------------------|
| `forgeId`                        | `github.com`              | no       | Used to verify forge commits and interpret `identity.forgeUsername` and `identity.forgeUserId` |
| `forgeRules`                     | object                    | no       |                                                                                                |
| `forgeRules.allowMergeCommits`   | `true`, `false` (default) | no       | The forge is allowed to make merge commits (e.g. merge PRs)                                    |
| `forgeRules.allowContentCommits` | `true`, `false` (default) | no       | The forge is allowed to make content changes (e.g. a user makes a change through web UI)       |

### Protected branches
Merge commits into protected branches are required to be done by a maintainer and cannot contain content changes.
When `requireMergeCommits` is set, only merge commits are allowed into the protected branch (no rebase/squash/plain commit).

| Config              | Value                | Required | Description  |
|---------------------|----------------------|----------|--------------|
| `protectedBranches` | list of branch names | no       | E.g. `main`  |

### Repository
| Config                  | Value                | Required                                   | Description                                                                                                                                         |
|-------------------------|----------------------|--------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------|
| `repositories`          | list of `repository` | yes                                        | Used to verify forge commits and interpret `identities.forgeUsername` and `identities.forgeUserId`                                                  |
| `repository.uri`        | repo URI             | yes                                        | E.g. `git+https://github.com/supply-chain-toosl/go-sandbox.git`                                                                                     |
| `repository.after`      | list of `after`      | yes                                        |                                                                                                                                                     |
| `after.sha1`            | git commit SHA-1     | yes, unless `after.sha256` is set          | The commit pointed to by `after.sha1` and it's ancestors will be ignored. If both `sha1` and `sha256` are set they must point to the same commit.   |
| `after.sha256`          | git commit SHA-256   | yes, unless `after.sha1` is set            | The commit pointed to by `after.sha256` and it's ancestors will be ignored. If both `sha1` and `sha256` are set they must point to the same commit. |
| `after.branch`          | branch name          | no, unless `protectedBranches` are used    | Associate the `after` hashes with a branch. This is used to verify protected branches.                                                              |
| `repository.exemptTags` | list of `exemptTag`  | no                                         | List of tags that will not be verified                                                                                                              |
| `exemptTag.ref`         | name of tag          | yes                                        | E.g. `refs/tags/v0.0.1`                                                                                                                             |
| `exemptTag.hash`        | object               | yes                                        | All hashes must point to the same tag                                                                                                               |
| `exemptTag.hash.sha1`   | git SHA-1            | yes, unless `exemptTag.hash.sha256` is set | Contents of `repository.exemptTags.ref`: hash of an annotated tag or a commit (for lightweight tags)                                                |
| `exemptTag.hash.sha256` | git SHA-256          | yes, unless `exemptTag.hash.sha1` is set   | Contents of `repository.exemptTags.ref`: hash of an annotated tag or a commit (for lightweight tags)                                                |

Generate `repositories.exemptTags`:
```sh
$ gitverify exempt-tags
[{"ref":"refs/tags/0.0.1","hash":{"sha1":"1f46f2053221c040ce5bcba0239bc09214a37658"}}]
```

Generate candidates for `after`
```sh
$ gitverify after-candidates
```

| Per repository overrides       | Value                | Required | Description                                 |
|--------------------------------|----------------------|----------|---------------------------------------------|
| `repository.identities`        | `identities`         | no       | Override global `identities` section        |
| `repository.maintainers`       | `maintainers`        | no       | Override global `maintainers` section       |
| `repository.contributors`      | `contributors`       | no       | Override global `contributors` section      |
| `repository.rules`             | `rules`              | no       | Override global `rules` section             |
| `repository.protectedBranches` | `protectedBranches`  | no       | Override global `protectedBranches` section |
| `repository.forgeRules`        | `forgeRules`         | no       | Override global `forgeRules` section        |

