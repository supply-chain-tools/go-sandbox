# How to migrate to `gitverify`

This is the recommended way to start using `gitverify` with an existing repository.

 - For additional config settings, see [config.md](config.md).
 - For FAQ, see [README.md](README.md)

### Basic, permissive config
- Fairly permissive config that checks commit signatures, but not tag signatures.
- The forge is allowed to make changes: PR merge, PR squash, as well as edits done through the UI on github.com will be accepted.
- All types of supported SSH signatures are allowed including security keys without user present and user verified.
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
    }
  ],
  "maintainers": [
    "stian.kristoffersen@telenor.no"
  ],
  "rules": {
    "allowSshSignatures": true,
    "requireSshUserPresent": false,
    "requireSshUserVerified": false,
    "allowGpgSignatures": true,
    "requireSignedTags": false,
    "requireMergeCommits": false
  },
  "forgeId": "github.com",
  "forgeRules": {
    "allowMergeCommits": true,
    "allowContentCommits": true
  },
  "repositories": [
    {
      "uri": "git+https://github.com/supply-chain-tools/go-sandbox.git",
      "after": [{
          "sha1": "1f46f2053221c040ce5bcba0239bc09214a37658"
        }]
    }
  ]
}
```

### `repository.after`
Currently, all commits in a repository must be compliant with config. If not the existing
state should either be removed or ignored using `after`. Both branches and tags might point
to existing commits that are not compliant. To get a list of `after` candidates run
```sh
$ gitverify after-candidates
<commit SHA> refs/heads/main
<commit SHA> refs/tags/test
...
```

In  the ideal case, only `main` will be listed here. Optionally clean delete and garbage collect the branches and 
tags that are not needed and rerun `after-candidates` to ge a shorter list.  Then include the list 
JSON list is in the last line as `after` in the config.

### `repository.exemptTags`
Even if `requireSignedTags: false` is set the integrity of annotated tags is still verified. Since annotated tags are
only allowed to be done my `maintainers`, old tags might not be compliant.

To get a list of all existing tags to use as `exemptTags` run
```sh
gitverify exempt-tags
```

This includes all tags, also those that might be in compliance, and light weight tags, which are not verified
is `requireSignedTags: false` is set. Either manually clean up the list or use as is in the config.