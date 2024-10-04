# gitrelease

**This is experimental code: the format is subject to change, and it might not work on all platform due to how it calls `git` and `ssh`.**

Creates a `tag` (regular Git tag) and `tag.link` ([in-toto](https://in-toto.io/) attestation) file.

## Use

### Create the first `tag` and `tag.link`
The first time the tool is run `--init` must be added to show that no existing tags are expected
```
gitrelease --keyfile <path to SSH private key> --init <tag> <commit>
```
This will prompt you to sign twice, first to sign the `tag`, then the `tag.link`. The output will be placed
in `tag.link.intoto.jsonl`, which is the `tag.link` ([example tag.link](example.tag.link.json)) wrapped in [DSSE](https://github.com/secure-systems-lab/dsse/blob/master/envelope.md).

The previous tag will be recoded in the tag message. It will be on the form `Tag-metadata: <base64 encoded metadata>`, as
seen in the [example tag](example.tag.txt) ([example metadata](example.tag.metadata.json)). This is to be able to order
the releases. The most recent tag is found by going through all the tags and choosing the tag that is not pointed to by
any other tag.

[](example.tag.metadata.json)

### Countersign
```
gitrelease --keyfile <path to SSH private key> --countersign tag.link.intoto.jsonl <tag> <commit>
```

This will add a signature to `tag.link.intoto.jsonl`.

### Verification
*Proper verification has yet to be implemented.*

`gitverify` can be used to verify the tag signature.

## Design considerations
`gitrelease` is designed to add further assurance on top of `gitverify`. The idea is that users can pick a mitigation level
according to their needs. `gitrelease` can be used without the `tag.link` files if threshold signatures are
not needed, since all the other mitigations are provided by the metadata in the `tag`.

 - **Explicit release**: a maintainer's intent to make a release must be verifiable. Making a release based on a branch or lightweight tag is not enough since these can be tampered with.
 - **Ordering of releases**: each `tag` contains the hash of the previous `tag`. This helps build servers detect if the forge is trying to reorder or omit releases.
 - **Repository teleportation mitigation**: the URI of the repository being released is included, so it cannot be confused with another copy of the same repo (e.g. a fork).
 - **SHA1 collision mitigation**: the `SHA-256` hashes of the `commit` and `tag` being released are included using `githash`.
 - **Threshold signatures**: more than one signature can be added to the `tag.link`, enabling requiring two or more maintainers to sign off on a release.
 - **Repository state**: additional state is recorded, like the tip of protected branches (only `main` is currently included, this should read use the `gitverify` config instead).
This can help detect other teleportation, rollback, and deletion attacks.
- **Timestamp**: having a timestamp in the release can prevent a forge from pushing a stale release to a build server.

`gitrelease` is inspired by the Reference State Log (RSL) in the paper [On Omitting Commits and Committing Omissions:
Preventing Git Metadata Tampering That
(Re)introduces Software Vulnerabilities](https://www.usenix.org/conference/usenixsecurity16/technical-sessions/presentation/torres-arias).
A fundamental shortcoming of `gitrelease`: it only records the state at releases, not with every `fetch` and `push` like the RSL is intended to do.