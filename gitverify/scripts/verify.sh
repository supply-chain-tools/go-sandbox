#!/bin/bash
cat data | ssh-keygen -Y check-novalidate -n file -f ed25519 -s ed25519.sig 
cat data | ssh-keygen -Y check-novalidate -n file -f ecdsa256 -s ecdsa256.sig
cat data | ssh-keygen -Y check-novalidate -n file -f rsa4096 -s rsa4096.sig

cat data | ssh-keygen -Y check-novalidate -n file -f ed25519-sk -s ed25519-sk.sig
cat data | ssh-keygen -Y check-novalidate -n file -f ecdsa-sk -s ecdsa-sk.sig

cat data | ssh-keygen -Y check-novalidate -n file -f ed25519 -s ed25519-sha256.sig
