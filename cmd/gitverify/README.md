# gitverify

**This is still a prototype and needs cleanup/better testing.**

## Demo: validate this repo
*For actual use the trust in the config file must be established before it can be used for validation.*
```sh
$ gitverify --config-file ../../gitverify.json --repository-uri git+https://github.com/supply-chain-tools/go-sandbox.git
validating...
OK
```

For `github.com` repos it is also possible place the config in 
```sh
~/.config/gitverify/github.com/supply-chain-tools/gitverify.json
# i.e. ~/config/gitverify/<forge>/<organization>/gitverify.json
```
and run
```
gitverify
```
This will use the first upstream URL in the git config to infer the forge, organization and repository name.

The default when inferring the config is also to store local state (described in [threat-model.md](threat-model.md)). It will be
placed in
```sh
~/.config/gitverify/github.com/supply-chain-tools/go-sandbox/local.json
# i.e. ~/config/gitverify/<forge>/<organization>/<repository>/local.json
```

## Migration Guide
See [migrate.md](migrate.md).

## Config File
See [config.md](config.md).

## Command line

### Verify that expected commit, tag, and/or branch is present

To verify that a `commit`, `tag` and/or `branch` exist in the repository
```sh
gitverify --commit 1f46f2053221c040ce5bcba0239bc09214a37658 --tag v0.0.1 --branch main
```
If `commit` and `tag` are both specified, they are verified to point to the same commit. If `branch` is specified
and at least one of `commit` and `tag` is specified, the commit they point to is verified to be present in the `branch`.

`--verify-at-tip` can be added to verify that `--commit/--tag` is at the tip of `--branch`.

### Verify that expected commit, tag, and/or branch is pointed to by HEAD

To verify that a `commit`, `tag` and/or `branch` is what `HEAD` points to
```sh
gitverify --commit 1f46f2053221c040ce5bcba0239bc09214a37658 --tag v0.0.1 --branch main --verify-on-head
```

## Threat Model
See [threat-model.md](threat-model.md).

## FAQ / Troubleshooting

### Performance issues
We aim to make this useful for large repositories like the Linux kernel, but for the time being it should be used
on smaller repos.

### Shallow repositories
Shallow repositories are currently not supported. All the repository state is needed to verify `SHA-1` and `SHA-256` hashes recursively.

### At most two parent commits
More than two parent commits is not supported.

### When migrating: `after` is set but verification is failing
You might have additional commits that either needs to be cleaned up or added to the list of `after`. To get a list 
of all commits that is not pointed to by other commits, run
```sh
gitverify after-candidates
```
Either add this list to `after` or delete the tags/branches and run garbage collection in the repository.

### When migrating: `requireSignedTags: false` but fails on tag verification
`gitverify` was designed to be strict and will still verify annotated tags for integrity, even if they are not signed.
To get a list of existing tags run
```sh
gitverify exempt-tags
```
which can be added as `exemptTags` in the config.

### Recover from incorrect state in the repository
The main way to recover is to manually verify that the state is not dangerous and to update `after` to ignore the wrong
commits. For tags they can either be deleted or added to `exemptTags`.

If commits or tags is deleted, and `local state` is used, it might need to be patched or recreated. There is no convenience
tooling for this at the moment.

### Git stash issue
**NB this will remove state, use with care!**

Git stash will create local commits that are not signed and will not pass the checks.

The stashes can be deleted as follows (**NB! only run this is you are fine with losing the data in the stashes**)
```sh
git stash clear # Remove all stashes
```

Remove the dangling commits
```sh
git gc --prune=now
```
### Remove local unsigned/wrong commits
**NB this will remove state, use with care!**

If commits that should not have been introduced, i.e. not signed or signed with the wrong key and have yet to be pushed, 
they can be removed.

Make sure that no branch points to the commit anymore, e.g. with `reset --hard` or `push --force`

Expire all dangling objects
```sh
git reflog expire --expire-unreachable=now --all
```

Run gc
```sh
git gc --prune=now
```

### Merge Tags
Currently not supported.

### Tags not pointing to commits
Currently not supported.