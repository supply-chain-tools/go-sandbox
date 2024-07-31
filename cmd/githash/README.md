# githash

Cryptographic agility for git hashes: hashes are computed in the same way as git, but
the hash function can be swapped out.

Get the `sha256` of a commit (this recomputes the commit, tree, and blob hashes recursively)
```
githash <commit>
```

Get the `blake2b` (or: `sha512`, `sha3-256`, `sha3-512`, `blake2b`, `sha1`)
```
githash --algorithm blake2b <commit>
```

Get the `sha256` of `HEAD`
```
githash
```

Get the `sha256` of a tree
```
githash --object-type tree <commit>
```

Get the `sha256` of a blob
```
githash --object-type blob <commit>
```
