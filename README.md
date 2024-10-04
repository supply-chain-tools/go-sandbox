# Supply Chain Tools: Go Sandbox

**This code is still considered experimental: it should not be relied on for important
stuff and breaking changes are to be expected.**

## Overview

CLIs
 - [cmd/gitverify](cmd/gitverify) - verify signatures and integrity of Git repositories
 - [cmd/gitrelease](cmd/gitrelease) - create `tag` and `tag.link` for a release
 - [cmd/githash](cmd/githash) - compute Git hashes with alternative hash functions
 - [cmd/gohash](cmd/gohash) - compute the hashes of Go packages in Git repositories
 - [cmd/dsse](cmd/dsse) - convenience CLI for [DSSE](https://github.com/secure-systems-lab/dsse/blob/master/envelope.md) files
 - [cmd/gitsearch](cmd/gitsearch) - a multi-string, multi-git-repo, all history, exact/fuzzy searcher
 - [cmd/repofetch](cmd/repofetch) - fetch all repos for a GitHub user or org

Libraries
 - [search](search) - the Trie based search that powers `gitsearch`
 - [gitkit](gitkit) - a collection Git related functionality, including searching through Git history
 - [gitsearch](gitsearch) - ties together `gitkit` and `search`
 - [iana](iana) - used to get TLDs for typosquatting
 - [hashset](hashset) - hashsets are used a lot in the code and not part of the Go standard library
 - [githash](githash) - compute git hashes with alternative hash functions

## Getting started

The tools in `cmd/` can be installed with `go`
```sh
go install github.com/supply-chain-tools/go-sandbox/cmd/gitverify@latest
```

Further information in the README for each tool.