# Supply Chain Tools: Go Sandbox

**This code is still considered experimental: it should not be relied on for important
stuff and breaking changes are to be expected.**

## Overview

CLIs
 - [cmd/gitsearch](cmd/gitsearch) - a multi-string, multi-git-repo, all history, exact/fuzzy searcher.
 - [cmd/repofetch](cmd/repofetch) - fetch all repos for a GitHub user or org
 
Libraries
 - [search](search) - the Trie based search that powers `gitsearch`
 - [gitkit](gitkit) - a collection Git related functionality, including searching through Git history
 - [gitsearch](gitsearch) - ties together `gitkit` and `search`
 - [iana](iana) - used to get TLDs for typosquatting
 - [hashset](hashset) - hashsets are used a lot in the code and not part of the Go standard library


## Getting started

### Install
```sh
go install github.com/supply-chain-tools/go-sandbox/cmd/repofetch@latest
go install github.com/supply-chain-tools/go-sandbox/cmd/gitsearch@latest
```

### Fetch and search repos
[cmd/repofetch](cmd/repofetch) supports downloading repos from GitHub, including for entire orgs. 
```sh
repofetch github.com/supply-chain-security
```

[cmd/gitsearch](cmd/gitsearch) has a wrapper script [cmd/gitsearch/gs](cmd/gitsearch/gs) with colors/pager.
Example search
```sh
$  gs test
```

Print the help instructions
```sh
$ gs -h
```

There is also a [tutorial](docs/gitsearch-tutorial.md).
