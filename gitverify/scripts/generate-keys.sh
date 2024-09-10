#!/bin/bash
ssh-keygen -t ed25519 -C "" -f ed25519
ssh-keygen -t ecdsa -b 256 -C "" -f ecdsa256
ssh-keygen -t rsa -b 4096 -C "" -f rsa4096

ssh-keygen -t ed25519-sk -C "" -f ed25519-sk
ssh-keygen -t ecdsa-sk -C "" -f ecdsa-sk
