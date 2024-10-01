# gohash

**This is still a prototype and needs cleanup/better testing.**

Compute the hashes of a Go packages in Git repositories.

The intention is to have a local copy of a repo containing a Go package, then for a
given commit:
1. run `gitverify` to check the
integrity, then
2. run `gohash` to check that the hashes match what's in
[Go's Checksum Database](https://sum.golang.org/).

```
# In github.com/supply-chain-tools/go-verify
$ gohash 96411d0af0481933a468561e0443e01b8044c952
== this is experimental and should not be relied on yet ==
# https://sum.golang.org/lookup/github.com/supply-chain-tools/go-sandbox@v0.0.0-20241001105810-96411d0af048
github.com/supply-chain-tools/go-sandbox v0.0.0-20241001105810-96411d0af048 h1:8+z0fzdvep//qddY1t4RBNln/kOPBuVZHlNmLSXjcaI=
github.com/supply-chain-tools/go-sandbox v0.0.0-20241001105810-96411d0af048/go.mod h1:rpMx5VkEw0ZJdk27NK6DPObmkfD3GlFK5XZh1lAWBiw=
```

Those hashes can then be compared with
 - Hashes in `go.sum` after running `go get github.com/supply-chain-tools/go-sandbox@5fa47d130dceb3f4ea3a53370986c7570df0c9a8`
 - It could also be used to audit the Go Checksum Database