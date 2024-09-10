#!/bin/bash
echo -n "foo" > data
cat data | ssh-keygen -Y sign -n file -f ed25519 > ed25519.sig 
cat data | ssh-keygen -Y sign -n file -f ecdsa256 > ecdsa256.sig
cat data | ssh-keygen -Y sign -n file -f rsa4096 > rsa4096.sig

cat data | ssh-keygen -Y sign -n file -f ed25519-sk > ed25519-sk.sig
cat data | ssh-keygen -Y sign -n file -f ecdsa-sk > ecdsa-sk.sig

#sha256
cat data | ssh-keygen -Y sign -n file -f ed25519 -O hashalg=sha256 > ed25519-sha256.sig
