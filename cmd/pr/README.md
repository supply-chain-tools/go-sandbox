# pr

A prototype for how to locally sign PRs (creation and approvals). It's an extension of the built-in
concept `mergetag`, to allow for more than one approval.

## Example
Create PR (create a PR tag)
```
pr --hash <commit> --branch main --message "Added feature X" --tag "user/1" create
```

Get hash for tag (create an approval tag)
```
git rev-parse user/1
```

Approve PR (merges the PR tag)
```
pr --hash <pr tag> --message "LGTM" --tag "user/1-approve" approve
```

Merge PR
```
pr --hash <pr tag> --message "Merging feature X" merge
```

## Format

### Create PR tag
Create a PR tag by tagging a commit with the message
```
message in regular way

Type: pr
Base-repository: <uri: e.g. git+https://github.com/supply-chain-tools/go-sandbox.git>
Base-branch: <branch: e.g. main>
Object-sha512: <SHA-512 of the commit the tag points to>
```

### Approve PR tag
Tag the PR tag with the message
```
message in regular way

Type: pr-approve
Object-sha512: <SHA-512 of the pr tag>
```

### Merge commit
Merge commit with a tag, with the message
```
message in regular way

Approve-tag: <add each approve-tag inline like a mergetag>
```

