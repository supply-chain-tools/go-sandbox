# githash

Cryptographic agility for git hashes: hashes are computed in the same way as git, but
the hash function can be swapped out.

Get the `sha256` of a commit
```
githash <sha1 commit hash>
```

Get the `blake2b` (or: `sha512`, `sha3-256`, `sha3-512`, `blake2b`, `sha1`)
```
githash --algorithm blake2b <sha1 commit hash>
```

Get the `sha256` of `HEAD`
```
githash
```

Get the `sha256` of a tree
```
githash --object-type tree <sha1 hash>
```

Get the `sha256` of a blob
```
githash --object-type blob <sha1 hash>
```

Get the `sha256` of a tag (*lightweight tags are not supported*)
```
githash --object-type tag <sha1 hash>
```
